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

// Package bazeldepsresolver resolves Java class names to precompiled jars set up by https://github.com/johnynek/bazel-deps/.
// NewResolver queries the file-system under the provided directory, and follows java_library, bind and java_import rules to
// the jars they reference. It then lists their contents.
// When resolving class names, it follows reverse edges back from class names to the java_library rules that transitively export them.
package bazeldepsresolver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/listclassesinjar"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/pkgloading"
)

// Resolver resolves class names according to a third-party directory structue created by https://github.com/johnynek/bazel-deps/.
type Resolver struct {
	thirdPartyDir string

	parent      map[*bazel.Rule]*bazel.Rule
	classToRule map[jadeplib.ClassName]*bazel.Rule

	loader pkgloading.Loader
}

// NewResolver returns a new Resolver.
// thirdPartyDir is a directory relative to the workspace directory, e.g. "thirdparty/jvm".
func NewResolver(ctx context.Context, workspaceDir, thirdPartyDir string, loader pkgloading.Loader) (*Resolver, error) {
	if filepath.IsAbs(thirdPartyDir) {
		return nil, fmt.Errorf("thirdPartyDir %s must be a relative path", thirdPartyDir)
	}
	stopwatch := time.Now()

	// Prefetch 'external' because bazel-deps always uses it.
	go loader.Load(ctx, []string{"external"})

	dirs := allPackages(workspaceDir, thirdPartyDir)
	pkgs, err := loader.Load(ctx, dirs)
	if err != nil {
		return nil, err
	}

	var layer []*bazel.Rule
	parent := make(map[*bazel.Rule]*bazel.Rule)
	classToRule := make(map[jadeplib.ClassName]*bazel.Rule)
	seenLabels := make(map[bazel.Label]bool)

	for _, pkg := range pkgs {
		for _, rule := range pkg.Rules {
			layer = append(layer, rule)
		}
	}

	// BFS
	for len(layer) > 0 {
		var toLoad []bazel.Label
		parentLabels := make(map[bazel.Label]*bazel.Rule)
		// Construct next layer
		for _, rule := range layer {
			var candidates []bazel.Label
			switch rule.Schema {
			case "java_library":
				candidates = append(candidates, rule.LabelListAttr("exports")...)
			case "bind":
				if l, err := rule.LabelAttr("actual"); err == nil {
					candidates = append(candidates, l)
				}
			case "java_import":
				listJar(ctx, rule, pkgs, classToRule)
			}

			for _, candidate := range candidates {
				if !seenLabels[candidate] {
					seenLabels[candidate] = true
					toLoad = append(toLoad, candidate)
					parentLabels[candidate] = rule
				}
			}
		}
		// Load rules in next layer
		rules, newPkgs, err := pkgloading.LoadRules(ctx, loader, toLoad)
		if err != nil {
			return nil, err
		}
		for pkgName, pkg := range newPkgs {
			pkgs[pkgName] = pkg
		}
		// Fill out nextLayer
		var nextLayer []*bazel.Rule
		for u, v := range parentLabels {
			if ru := rules[u]; ru != nil {
				nextLayer = append(nextLayer, ru)
				parent[ru] = v
			}
		}
		layer = nextLayer
	}
	log.Printf("Created bazel-deps resolver (%dms)", int64(time.Now().Sub(stopwatch)/time.Millisecond))

	return &Resolver{thirdPartyDir, parent, classToRule, loader}, nil
}

// allPackages returns all directories rooted at 'dir', relative to workspaceDir.
func allPackages(workspaceDir, dir string) []string {
	var result []string
	filepath.Walk(filepath.Join(workspaceDir, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			relPath, err := filepath.Rel(workspaceDir, path)
			if err != nil {
				return err
			}
			result = append(result, relPath)
		}
		return nil
	})
	return result
}

func listJar(ctx context.Context, rule *bazel.Rule, pkgs map[string]*bazel.Package, classToRule map[jadeplib.ClassName]*bazel.Rule) error {
	pkg := pkgs[rule.PkgName]
	if pkg == nil {
		return fmt.Errorf("can't find package object for rule %s - this is a bug in bazeldepsresolver", rule.Label())
	}
	pkgPath := pkg.Path
	for _, jar := range rule.StringListAttr("jars") {
		fileName := filepath.Join(pkgPath, jar)
		cls, err := listclassesinjar.List(fileName)
		if err != nil {
			log.Printf("Warning: unable to list classes in jar %s", fileName)
		}
		for _, c := range cls {
			classToRule[c] = rule
		}
	}
	return nil
}

// Name returns a description of the resolver.
func (r *Resolver) Name() string {
	return "github.com/johnynek/bazel-deps/"
}

// Resolve resolves class names according to an in-memory map.
func (r *Resolver) Resolve(ctx context.Context, classNames []jadeplib.ClassName, consumingRules map[bazel.Label]map[bazel.Label]bool) (map[jadeplib.ClassName][]*bazel.Rule, error) {
	result := make(map[jadeplib.ClassName][]*bazel.Rule)
	for _, cls := range classNames {
		rule := r.classToRule[cls]
		if rule == nil {
			continue
		}
		for r.parent[rule] != nil {
			rule = r.parent[rule]
		}
		result[cls] = append(result[cls], rule)
	}

	return result, nil
}
