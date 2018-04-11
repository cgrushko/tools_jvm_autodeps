// Copyright 2018 The Jadep Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package filter provides functions to filter rules based on their use site.
package filter

import (
	"fmt"
	"strings"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/compat"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
)

const (
	pkgVisibilityName         = "__pkg__"
	subpackagesVisibilityName = "__subpackages__"
)

// JavaDependencyRuleKinds lists the kinds of Java rule that can be a dependency of a Java rule.
// These typically don't include binary rules.
var JavaDependencyRuleKinds = map[string]bool{
	"android_library":            true,
	"java_import":                true,
	"java_library":               true,
	"java_lite_proto_library":    true,
	"java_mutable_proto_library": true,
	"java_plugin":                true,
	"java_proto_library":         true,
	"java_wrap_cc":               true,
	"proto_library":              true,
}

// JavaEditableRuleKinds lists the kinds of rules that Jadep will edit.
var JavaEditableRuleKinds = map[string]bool{
	"android_binary":           true,
	"android_library":          true,
	"android_local_test":       true,
	"android_robolectric_test": true,
	"android_test":             true,
	"java_binary":              true,
	"java_library":             true,
	"java_plugin":              true,
	"java_test":                true,
}

// RuleKindsToLoad lists the kinds of rules that Jadep requests from a PackageLoader server.
// This should list all kinds that Jadep interacts with in any way.
var RuleKindsToLoad = map[string]bool{
	"android_binary":             true,
	"android_library":            true,
	"android_local_test":         true,
	"android_robolectric_test":   true,
	"android_test":               true,
	"bind":                       true,
	"filegroup":                  true,
	"java_binary":                true,
	"java_import":                true,
	"java_library":               true,
	"java_lite_proto_library":    true,
	"java_mutable_proto_library": true,
	"java_plugin":                true,
	"java_proto_library":         true,
	"java_test":                  true,
	"java_wrap_cc":               true,
	"proto_library":              true,
}

// IsValidDependency returns false if dep should not be used as a dependency.
// It only relies on information inside the rule itself (e.g., kind, tags).
// For visibility tests, see CheckVisibility().
func IsValidDependency(dep *bazel.Rule) bool {
	if !JavaDependencyRuleKinds[dep.Schema] {
		return false
	}

	tags := dep.StringListAttr("tags")
	for _, tag := range tags {
		if tag == "avoid_dep" {
			return false
		}
	}

	return true
}

// VisQuery represents the question, "is rule Rule visible to the package Pkg".
type VisQuery struct {
	Rule *bazel.Rule
	Pkg  string
}

// CheckVisibility returns true if rules are visible to packages.
// It computes the answer for multiple queries with the minimal amount of package loads possible.
// We assume the loader is a CachingLoader for performance.
// The result satisfies result[R, P]==true iff rule R is visibile to package P.
func CheckVisibility(ctx context.Context, loader pkgloading.Loader, query map[VisQuery]bool) (map[VisQuery]bool, error) {
	ctx, endSpan := compat.NewLocalSpan(ctx, "Jade: CheckVisibility")
	defer endSpan()

	ret := make(map[VisQuery]bool)
	undecided := make(map[VisQuery]bool)

	for vq := range query {
		switch localVisibleTo(vq.Rule, vq.Pkg) {
		case yes:
			ret[vq] = true
		case unknown:
			undecided[vq] = true
		}
	}

	visited := make(map[VisQuery]map[bazel.Label]bool)
	var nodes map[VisQuery][]bazel.Label = pkgGroupsInVisibility(undecided, visited)

	// BFS over 'nodes'.
	// Edges are realized from package_group()s that incldue= other package_group()s.
	// Unlike in classical BFS, each layer is handled together to make a minimal number of BUILD package loads.
	// Whenever we can satisfy a query, we stop walking all nodes that originated from that query (newlyDecided below).
	for len(nodes) > 0 {
		pkgGroups, err := pkgloading.LoadPackageGroups(ctx, loader, labels(nodes))
		if err != nil {
			return nil, fmt.Errorf("Error loading package_group()s %v:\n%v", labels(nodes), err)
		}
		newNodes := make(map[VisQuery][]bazel.Label)
		for vq, pkgGroupLabels := range nodes {
			for _, pkgGroupLabel := range pkgGroupLabels {
				pg := pkgGroups[pkgGroupLabel]
				if pg == nil {
					continue
				}
				switch specVisibleTo(pg.Specs, vq.Pkg) {
				case no:
					ret[vq] = false
				case yes:
					ret[vq] = true
				case unknown:
					for _, inc := range pg.Includes {
						if !visited[vq][inc] {
							if visited[vq] == nil {
								visited[vq] = make(map[bazel.Label]bool)
							}
							visited[vq][inc] = true
							newNodes[vq] = append(newNodes[vq], inc)
						}
					}
				}
			}
		}
		for n := range newNodes {
			if _, ok := ret[n]; ok {
				delete(newNodes, n)
			}
		}
		nodes = newNodes
	}

	return ret, nil
}

// tri represents a tri-state: true, false or unknown.
type tri int

const (
	no tri = iota
	yes
	unknown
)

// localVisibleTo checks whether dep is visible to consPkgName without loading BUILD packages.
// It only considers individual entries in dep's visiblity attribute.
// It returns 'yes' if it's certain dep is visible from consPkgName (e.g., it's //visibility:public),
// 'no' when it isn't (e.g. //visibility:private), and 'unknown' otherwise.
func localVisibleTo(dep *bazel.Rule, consPkgName string) tri {
	// Google-specific:
	if consPkgName == dep.PkgName ||
		strings.TrimPrefix(consPkgName, "javatests/") ==
			strings.TrimPrefix(dep.PkgName, "java/") {
		return yes
	}

	vis := dep.LabelListAttr("visibility")
	if len(vis) == 0 {
		return no
	}
	for _, v := range vis {
		if v == "//visibility:public" || v == "//visibility:legacy_public" {
			return yes
		}
		if v == "//visibility:private" {
			return no
		}
		visPkgName, visName := v.Split()
		if visName == pkgVisibilityName && visPkgName == consPkgName {
			return yes
		}
		if visName == subpackagesVisibilityName && subPackageOf(consPkgName, visPkgName) {
			return yes
		}
	}

	return unknown
}

// specVisibleTo returns 'yes' if any package_group spec in 'specs' grants visibility to consPkgName.
// For example, it returns 'yes' for specs = [x/...] and consPkgName = "x/subx".
// Reminder: the values for 'specs' are detailed in bazel.PackageGroup.Specs in bazel/bazel.go.
func specVisibleTo(specs []string, consPkgName string) tri {
	for _, spec := range specs {
		if spec == "//..." || spec == consPkgName {
			return yes
		}
		wildcard := strings.LastIndex(spec, "/...")
		if wildcard != -1 && subPackageOf(consPkgName, spec[:wildcard]) {
			return yes
		}
	}
	return unknown
}

// pkgGroupsInVisibility returns the list of package_group()s referenced in the visibility attribute of rules in query.
// We need to know which query ("is R visible to pkg?") will be answered by each package_group(), hence the return type.
//
// 'visited' is modified: visited[vq][lbl] == true iff result[vq] contains lbl.
func pkgGroupsInVisibility(query map[VisQuery]bool, visited map[VisQuery]map[bazel.Label]bool) map[VisQuery][]bazel.Label {
	ret := make(map[VisQuery][]bazel.Label)
	for vq := range query {
		vis := vq.Rule.LabelListAttr("visibility")
		for _, pkgGroupLabel := range vis {
			_, visName := pkgGroupLabel.Split()
			if visName == pkgVisibilityName || visName == subpackagesVisibilityName {
				continue
			}
			if !visited[vq][pkgGroupLabel] {
				if visited[vq] == nil {
					visited[vq] = make(map[bazel.Label]bool)
				}
				visited[vq][pkgGroupLabel] = true
				ret[vq] = append(ret[vq], pkgGroupLabel)
			}
		}
	}
	return ret
}

func labels(nodes map[VisQuery][]bazel.Label) []bazel.Label {
	var ret []bazel.Label
	for _, labels := range nodes {
		for _, lbl := range labels {
			ret = append(ret, lbl)
		}
	}
	return ret
}

// subPackageOf returns true if subpackage is a sub-package of pkg.
// For example,
//   subPackageOf("a/b", "a") == true
//   subPackageOf("a", "a") == true
//   subPackageOf("a", "a/b") == false
func subPackageOf(subpackage, pkg string) bool {
	if subpackage == pkg {
		return true
	}
	if pkg == "" {
		return true
	}
	return strings.HasPrefix(subpackage, pkg) && subpackage[len(pkg)] == '/'
}
