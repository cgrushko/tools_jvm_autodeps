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

// Package pkgloaderfakes provides fakes for structs returned from pkgloaderclient.
package pkgloaderfakes

import "github.com/bazelbuild/tools_jvm_autodeps/bazel"

// Pkg creates a Bazel package from a list of rules, as if they were returned by pkgloaderclient.Loader.
func Pkg(rules []*bazel.Rule) *bazel.Package {
	resFiles := map[string]string{"BUILD": ""}
	resRules := make(map[string]*bazel.Rule)
	for _, r := range rules {
		resRules[r.Attrs["name"].(string)] = r
		srcs, _ := r.Attrs["srcs"].([]string)
		for _, src := range srcs {
			resFiles[src] = ""
		}
	}
	return &bazel.Package{
		DefaultVisibility: []bazel.Label{"//visibility:private"},
		Files:             resFiles,
		Rules:             resRules,
	}
}

// JavaLibrary creates a java_library Bazel rule as if it were returned by pkgloaderclient.Loader.
// TODO: Replace this with a version that calls Rule() directly.
func JavaLibrary(pkgName string, name string, srcs []string, deps []string, exports []string) *bazel.Rule {
	return rule("java_library", pkgName, name, srcs, deps, exports)
}

// JavaBinary creates a java_binary Bazel rule as if it were returned by pkgloaderclient.Loader.
// TODO: Replace this with a version that calls Rule() directly.
func JavaBinary(pkgName string, name string, srcs []string, deps []string, exports []string) *bazel.Rule {
	return rule("java_binary", pkgName, name, srcs, deps, exports)
}

func rule(kind string, pkgName string, name string, srcs []string, deps []string, exports []string) *bazel.Rule {
	var ms []RuleModifier
	if srcs != nil {
		ms = append(ms, Srcs(srcs...))
	}
	if deps != nil {
		ms = append(ms, Deps(deps...))
	}
	if exports != nil {
		ms = append(ms, Exports(exports...))
	}
	return Rule(kind, pkgName, name, ms...)
}

// RuleModifier is used by Rule to modify a newly created bazel.Rule object.
// See AttrModifier for an example.
type RuleModifier interface {
	Apply(r *bazel.Rule)
}

// Rule creates a bazel.Rule and applies modifiers to it.
// Example: Rule("java_library", "x", "Foo", Srcs("Foo.java"))
func Rule(schema, pkgName, ruleName string, modifiers ...RuleModifier) *bazel.Rule {
	result := bazel.NewRule(schema, pkgName, ruleName, nil)
	for _, m := range modifiers {
		m.Apply(result)
	}
	return result
}

// AttrModifier is a RuleModifier that changes the attribute named AttrName to Value.
type AttrModifier struct {
	RuleModifier
	AttrName string
	Value    interface{}
}

// Attr returns a new AttrModifier that changes attrName to value.
// For example, Rule("java_library", "x", "Foo", Attr("deps", "Foo.java"))
func Attr(attrName string, value interface{}) *AttrModifier {
	return &AttrModifier{AttrName: attrName, Value: value}
}

// Apply applies the attribute change as specified by AttrModifier.
func (m *AttrModifier) Apply(rule *bazel.Rule) {
	rule.Attrs[m.AttrName] = m.Value
}

// Srcs is a convenience wrapper around Attr("srcs", value).
func Srcs(value ...string) *AttrModifier {
	return Attr("srcs", value)
}

// Deps is a convenience wrapper around Attr("srcs", value).
func Deps(value ...string) *AttrModifier {
	return Attr("deps", value)
}

// Exports is a convenience wrapper around Attr("srcs", value).
func Exports(value ...string) *AttrModifier {
	return Attr("exports", value)
}
