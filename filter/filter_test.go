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

package filter

import (
	"testing"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/loadertest"
	"github.com/google/go-cmp/cmp"
)

var equateErrorMessage = cmp.Comparer(func(x, y error) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return x.Error() == y.Error()
})

func TestIsValidDependency(t *testing.T) {
	type Attrs = map[string]interface{}

	var tests = []struct {
		desc string
		dep  *bazel.Rule
		want bool
	}{
		{
			"allow java_library rules",
			&bazel.Rule{Schema: "java_library"},
			true,
		},
		{
			"don't allow filegroup() dependencies",
			&bazel.Rule{Schema: "filegroup"},
			false,
		},
		{
			"don't allow rules with tags=avoid_dep",
			&bazel.Rule{"java_library", "x", Attrs{"tags": []string{"avoid_dep"}}},
			false,
		},
		{
			"don't allow deprecated rules",
			&bazel.Rule{"java_library", "x", Attrs{"deprecation": "don't use this rule!"}},
			false,
		},
	}

	for _, tt := range tests {
		got := IsValidDependency(tt.dep)
		if got != tt.want {
			t.Errorf("%s: IsValidDependency(%v) = %v, want %v", tt.desc, tt.dep, got, tt.want)
		}
	}
}

func TestLocalVisibleTo(t *testing.T) {
	type Attrs = map[string]interface{}

	var tests = []struct {
		desc        string
		consPkgName string
		dep         *bazel.Rule
		want        tri
	}{
		{
			desc:        "A rule is visible to rules in same package",
			consPkgName: "x",
			dep:         bazel.NewRule("java_library", "x", "Dep", nil),
			want:        yes,
		},
		{
			desc:        "A rule is visible to rules in same java/ package (Google specific)",
			consPkgName: "java/x",
			dep:         bazel.NewRule("java_library", "java/x", "Dep", nil),
			want:        yes,
		},
		{
			desc:        "A rule is visible to rules in same javatests/ package (Google specific)",
			consPkgName: "javatests/x",
			dep:         bazel.NewRule("java_library", "javatests/x", "Dep", nil),
			want:        yes,
		},
		{
			desc:        "A private rule is not visible to rules in other packages",
			consPkgName: "y",
			dep:         bazel.NewRule("java_library", "x", "Dep", Attrs{"visibility": []string{"//visibility:private"}}),
			want:        no,
		},
		{
			desc:        "A public rule is visible to rules in other packages",
			consPkgName: "y",
			dep:         bazel.NewRule("java_library", "x", "Dep", Attrs{"visibility": []string{"//visibility:public"}}),
			want:        yes,
		},
		{
			desc:        "An empty visibility=[] is the same as //visibility:private",
			consPkgName: "y",
			dep:         bazel.NewRule("java_library", "x", "Dep", Attrs{"visibility": []string{}}),
			want:        no,
		},
		{
			desc:        "java/$X is always visible to javatests/$X (Google specific)",
			consPkgName: "javatests/foo/bar",
			dep:         bazel.NewRule("java_library", "java/foo/bar", "Dep", nil),
			want:        yes,
		},
		{
			desc:        "//y:Dep has visibility=//x:__subpackages__, so it's visible from 'x'",
			consPkgName: "x",
			dep:         bazel.NewRule("java_library", "y", "Dep", Attrs{"visibility": []string{"//x:__subpackages__"}}),
			want:        yes,
		},
		{
			desc:        "//y:Dep has visibility=//x:__subpackages__, so it's visible from 'x/subx'",
			consPkgName: "x/subx",
			dep:         bazel.NewRule("java_library", "y", "Dep", Attrs{"visibility": []string{"//x:__subpackages__"}}),
			want:        yes,
		},
		{
			desc:        "//y:Dep has visibility=//x:__pkg__, so it's visible from 'x'",
			consPkgName: "x",
			dep:         bazel.NewRule("java_library", "y", "Dep", Attrs{"visibility": []string{"//x:__pkg__"}}),
			want:        yes,
		},
		{
			desc: "//y:Dep has visibility=//x:__pkg__, so we don't know whether it's visible from 'x'. " +
				"(note: localVisibleTo is very simple, and doesn't consider all entries in visibility, so in this case it leaves the decision to other filters)",
			consPkgName: "x/subx",
			dep:         bazel.NewRule("java_library", "y", "Dep", Attrs{"visibility": []string{"//x:__pkg__"}}),
			want:        unknown,
		},
	}

	for _, tt := range tests {
		got := localVisibleTo(tt.dep, tt.consPkgName)
		if got != tt.want {
			t.Errorf("%s: localVisibleTo(%v, %v) = %v, want %v", tt.desc, tt.dep, tt.consPkgName, got, tt.want)
		}
	}
}

func TestSpecVisibleTo(t *testing.T) {
	var tests = []struct {
		desc        string
		consPkgName string
		specs       []string
		want        tri
	}{
		{
			desc:        "A public rule is visible to rules in other packages",
			consPkgName: "y",
			specs:       []string{"//..."},
			want:        yes,
		},
		{
			desc:        "//y:Dep has visibility=//x:__subpackages__, so it's visible from 'x'",
			consPkgName: "x",
			specs:       []string{"x/..."},
			want:        yes,
		},
		{
			desc:        "//y:Dep has visibility=//x:__subpackages__, so it's visible from 'x/subx'",
			consPkgName: "x/subx",
			specs:       []string{"x/..."},
			want:        yes,
		},
		{
			desc:        "//y:Dep has visibility=//x:__pkg__, so it's visible from 'x'",
			consPkgName: "x",
			specs:       []string{"x"},
			want:        yes,
		},
		{
			desc: "//y:Dep has visibility=//x:__pkg__, so we don't know whether it's visible from 'x'. " +
				"(note: specVisibleTo is very simple, and doesn't consider all entries in visibility, so in this case it leaves the decision to other filters)",
			consPkgName: "x/subx",
			specs:       []string{"x"},
			want:        unknown,
		},
	}

	for _, tt := range tests {
		got := specVisibleTo(tt.specs, tt.consPkgName)
		if got != tt.want {
			t.Errorf("%s: specVisibleTo(%v, %v) = %v, want %v", tt.desc, tt.specs, tt.consPkgName, got, tt.want)
		}
	}
}

func TestPkgGroupsInVisibility(t *testing.T) {
	type Attrs = map[string]interface{}

	rule := bazel.NewRule("java_library", "y", "Dep", Attrs{"visibility": []string{"//x:__pkg__", "//y:__subpackages__", ":group1", "//z:group2"}})

	tests := []struct {
		desc        string
		query       map[VisQuery]bool
		visited     map[VisQuery]map[bazel.Label]bool
		want        map[VisQuery][]bazel.Label
		wantVisited map[VisQuery]map[bazel.Label]bool
	}{
		{
			desc:        "Don't return __pkg__ or __subpackages__",
			query:       map[VisQuery]bool{{Rule: rule, Pkg: "x"}: true},
			visited:     map[VisQuery]map[bazel.Label]bool{},
			want:        map[VisQuery][]bazel.Label{{Rule: rule, Pkg: "x"}: {"//y:group1", "//z:group2"}},
			wantVisited: map[VisQuery]map[bazel.Label]bool{{Rule: rule, Pkg: "x"}: {"//y:group1": true, "//z:group2": true}},
		},
	}

	for _, tt := range tests {
		got := pkgGroupsInVisibility(tt.query, tt.visited)
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Errorf("%s: Diff in pkgGroupsInVisibility (-got +want).\n%s", tt.desc, diff)
		}
		// tt.visited is modified by pkgGroupsInVisibility.
		if diff := cmp.Diff(tt.visited, tt.wantVisited); diff != "" {
			t.Errorf("%s: Diff in visited (-got +want).\n%s", tt.desc, diff)
		}
	}
}

func TestCheckVisibility(t *testing.T) {
	type Attrs = map[string]interface{}

	yDep1 := bazel.NewRule("java_library", "y", "Dep1", Attrs{"visibility": []string{":group"}})
	yDep2 := bazel.NewRule("java_library", "y", "Dep2", Attrs{"visibility": []string{"//z:group", "//y:group"}})
	yDep3 := bazel.NewRule("java_library", "y", "Dep3", Attrs{"visibility": []string{":group", "//x:__pkg__"}})
	zDep1 := bazel.NewRule("java_library", "z", "Dep1", Attrs{"visibility": []string{":group"}})

	var tests = []struct {
		desc          string
		existingPkgs  map[string]*bazel.Package
		query         map[VisQuery]bool
		want          map[VisQuery]bool
		wantErr       error
		expectedLoads [][]string
	}{
		{
			desc: "Scenario: //y:Dep1 has visibility = package_group(), which includes another package group, which specs x/... ." +
				"We expect //y:Dep1 to be visible to //x:*.",
			existingPkgs: map[string]*bazel.Package{
				"y": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Includes: []bazel.Label{"//z:group"},
						},
					},
				},
				"z": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Specs: []string{"x/..."},
						},
					},
				},
			},
			query:         map[VisQuery]bool{{Rule: yDep1, Pkg: "x"}: true},
			want:          map[VisQuery]bool{{Rule: yDep1, Pkg: "x"}: true},
			expectedLoads: [][]string{{"y"}, {"z"}},
		},
		{
			desc: "Scenario: //y:Dep1 has visibility = package_group(), which includes another package group, which specs z/... ." +
				"We expect //y:Dep1 to be non-visible to //x:*.",
			existingPkgs: map[string]*bazel.Package{
				"y": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Includes: []bazel.Label{"//z:group"},
						},
					},
				},
				"z": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Specs: []string{"z/..."},
						},
					},
				},
			},
			query:         map[VisQuery]bool{{Rule: yDep1, Pkg: "x"}: true},
			want:          map[VisQuery]bool{},
			expectedLoads: [][]string{{"y"}, {"z"}},
		},
		{
			desc: "//y:Dep1 has visibility=//y:group, which includes //z:group and has specs=x. " +
				"Test that we do not load the 'z' package because the spec is enough to prove that //y:Dep1 is visible to //x:*",
			existingPkgs: map[string]*bazel.Package{
				"y": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Specs:    []string{"x"},
							Includes: []bazel.Label{"//z:group"},
						},
					},
				},
			},
			query:         map[VisQuery]bool{{Rule: yDep1, Pkg: "x"}: true},
			want:          map[VisQuery]bool{{Rule: yDep1, Pkg: "x"}: true},
			expectedLoads: [][]string{{"y"}},
		},
		{
			desc: "//y:Dep2 has visibility=[//y:group, //z:group] where //y:group is enough to answer the query, and //z:group includes yet another group. " +
				"Test that we don't load 'w', because once //y:group proves visibility we stop walking both from //y:group and from //z:group.",
			existingPkgs: map[string]*bazel.Package{
				"y": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Specs: []string{"x"},
						},
					},
				},
				"z": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Includes: []bazel.Label{"//w:group"},
						},
					},
				},
			},
			query:         map[VisQuery]bool{{Rule: yDep2, Pkg: "x"}: true},
			want:          map[VisQuery]bool{{Rule: yDep2, Pkg: "x"}: true},
			expectedLoads: [][]string{{"y", "z"}},
		},
		{
			desc:          "//y:Dep3 has visibility=[//z:group, //x:__pkg__]. Test that we don't load 'z' at all, because //x:__pkg__ is enough to prove visibility.",
			query:         map[VisQuery]bool{{Rule: yDep3, Pkg: "x"}: true},
			want:          map[VisQuery]bool{{Rule: yDep3, Pkg: "x"}: true},
			expectedLoads: nil,
		},
		{
			desc: "Multiple consumers simultaneously. We query whether //y:Dep1 is visible from x1 and //z:Dep1 from x2. " +
				"In order to answer the first query we must load //y:group, whereas to answer the second query we must load //z:group and then //w:group. " +
				"Test that we load y+z together, and then w.",
			existingPkgs: map[string]*bazel.Package{
				"y": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Specs: []string{"x1"},
						},
					},
				},
				"z": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Includes: []bazel.Label{"//w:group"},
						},
					},
				},
				"w": {
					PackageGroups: map[string]*bazel.PackageGroup{
						"group": {
							Specs: []string{"x2"},
						},
					},
				},
			},
			query:         map[VisQuery]bool{{Rule: yDep1, Pkg: "x1"}: true, {Rule: zDep1, Pkg: "x2"}: true},
			want:          map[VisQuery]bool{{Rule: yDep1, Pkg: "x1"}: true, {Rule: zDep1, Pkg: "x2"}: true},
			expectedLoads: [][]string{{"y", "z"}, {"w"}},
		},
		{
			desc:          "Tolerate missing package_groups. //y:Dep1 has visibility=//y:group, but that group doesn't exist.",
			query:         map[VisQuery]bool{{Rule: yDep1, Pkg: "x"}: true},
			want:          map[VisQuery]bool{},
			expectedLoads: [][]string{{"y"}},
		},
	}

	for _, tt := range tests {
		loader := &loadertest.StubLoader{Pkgs: tt.existingPkgs}
		got, err := CheckVisibility(context.Background(), loader, tt.query)
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("%s: CheckVisibility(%#v) has diff (-want +got):\n%v", tt.desc, tt.query, diff)
		}
		if diff := cmp.Diff(tt.wantErr, err, equateErrorMessage); diff != "" {
			t.Errorf("%s, CheckVisibility(%#v) returned diff in error (-want +got):\n%s", tt.desc, tt.query, diff)
		}
		if diff := cmp.Diff(tt.expectedLoads, loader.RecordedCalls); diff != "" {
			t.Errorf("%s, CheckVisibility(%#v) diff in package loads (-want +got):\n%s", tt.desc, tt.query, diff)
		}
	}
}

func TestSubPackageOf(t *testing.T) {
	var tests = []struct {
		subpackage string
		pkg        string
		want       bool
	}{
		{"a", "a", true},
		{"a/b", "a", true},
		{"a/b/c/d", "a", true},
		{"ab", "a", false},
		{"a", "a/b", false},
		{"a", "", true},
	}
	for _, tt := range tests {
		got := subPackageOf(tt.subpackage, tt.pkg)
		if got != tt.want {
			t.Errorf("subPackageOf(%v, %v) = %v, want %v", tt.subpackage, tt.pkg, got, tt.want)
		}
	}
}
