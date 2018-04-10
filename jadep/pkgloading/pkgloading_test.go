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

package pkgloading

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/loadertest"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/pkgloaderfakes"
	"github.com/google/go-cmp/cmp"
)

func TestLoadRules(t *testing.T) {
	var tests = []struct {
		desc         string
		fixture      map[string]*bazel.Package
		labelsToLoad []bazel.Label
		want         map[bazel.Label]*bazel.Rule
	}{
		{
			"basic",
			map[string]*bazel.Package{"x": pkgloaderfakes.Pkg([]*bazel.Rule{pkgloaderfakes.JavaLibrary("x", "Foo", []string{"Foo.java"}, nil, nil)})},
			[]bazel.Label{"//x:Foo"},
			map[bazel.Label]*bazel.Rule{"//x:Foo": pkgloaderfakes.JavaLibrary("x", "Foo", []string{"Foo.java"}, nil, nil)},
		},
		{
			"two labels from the same package",
			map[string]*bazel.Package{
				"x": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("x", "Foo", []string{"Foo.java"}, nil, nil),
					pkgloaderfakes.JavaLibrary("x", "Bar", []string{"Bar.java"}, nil, nil),
				})},
			[]bazel.Label{"//x:Foo", "//x:Bar"},
			map[bazel.Label]*bazel.Rule{
				"//x:Foo": pkgloaderfakes.JavaLibrary("x", "Foo", []string{"Foo.java"}, nil, nil),
				"//x:Bar": pkgloaderfakes.JavaLibrary("x", "Bar", []string{"Bar.java"}, nil, nil),
			},
		},
		{
			"two labels from different packages",
			map[string]*bazel.Package{
				"x": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("x", "Foo", []string{"Foo.java"}, nil, nil),
				}),
				"y": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("y", "Bar", []string{"Bar.java"}, nil, nil),
				}),
			},
			[]bazel.Label{"//x:Foo", "//y:Bar"},
			map[bazel.Label]*bazel.Rule{
				"//x:Foo": pkgloaderfakes.JavaLibrary("x", "Foo", []string{"Foo.java"}, nil, nil),
				"//y:Bar": pkgloaderfakes.JavaLibrary("y", "Bar", []string{"Bar.java"}, nil, nil),
			},
		},
	}

	for _, tt := range tests {
		actual, actualPkgs, err := LoadRules(context.Background(), &loadertest.StubLoader{Pkgs: tt.fixture}, tt.labelsToLoad)
		if err != nil {
			t.Errorf("%s: LoadRules(%v) has error %v", tt.desc, tt.labelsToLoad, err)
		}
		if diff := cmp.Diff(actual, tt.want); diff != "" {
			t.Errorf("%s: LoadRules(%v) rules diff: (-got +want)\n%s", tt.desc, tt.labelsToLoad, diff)
		}
		if diff := cmp.Diff(actualPkgs, tt.fixture); diff != "" {
			t.Errorf("%s: LoadRules(%v) pkgs diff: (-got +want)\n%s", tt.desc, tt.labelsToLoad, diff)
		}
	}
}

func TestLoadPackageGroups(t *testing.T) {
	var tests = []struct {
		desc         string
		fixture      map[string]*bazel.Package
		labelsToLoad []bazel.Label
		want         map[bazel.Label]*bazel.PackageGroup
	}{
		{
			desc: "basic",
			fixture: map[string]*bazel.Package{
				"x": {PackageGroups: map[string]*bazel.PackageGroup{"group": {Specs: []string{"foo/..."}}}},
			},
			labelsToLoad: []bazel.Label{"//x:group"},
			want:         map[bazel.Label]*bazel.PackageGroup{"//x:group": {Specs: []string{"foo/..."}}},
		},
	}

	for _, tt := range tests {
		actual, err := LoadPackageGroups(context.Background(), &loadertest.StubLoader{Pkgs: tt.fixture}, tt.labelsToLoad)
		if err != nil {
			t.Errorf("%s: LoadPackageGroups(%v) has error %v", tt.desc, tt.labelsToLoad, err)
		}
		if diff := cmp.Diff(actual, tt.want); diff != "" {
			t.Errorf("%s: LoadPackageGroups(%v) diff: (-got +want)\n%s", tt.desc, tt.labelsToLoad, diff)
		}
	}
}

func TestFindPackageName(t *testing.T) {
	tmpRoot, err := ioutil.TempDir("", "")
	defer os.RemoveAll(tmpRoot)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	workspaceDir := filepath.Join(tmpRoot, "workspace")

	var tests = []struct {
		desc             string
		filename         string
		existingPackages []string
		wantPkgName      string
	}{
		{
			desc:             "Test BUILD file is one directory away from the filename.",
			filename:         "java/com/javatools/Jade.java",
			existingPackages: []string{"java/com"},
			wantPkgName:      "java/com",
		},
		{
			desc:             "Test BUILD file is in the same directory as the filename.",
			filename:         "java/com/Jadep.java",
			existingPackages: []string{"java/com"},
			wantPkgName:      "java/com",
		},
		{
			desc:        "Test for when there is are BUILD file.",
			filename:    "java/com/Jade.java",
			wantPkgName: "",
		},
	}
	for _, test := range tests {
		test := test
		func() {
			for _, p := range test.existingPackages {
				os.MkdirAll(filepath.Join(workspaceDir, p), os.ModePerm)
				if err := ioutil.WriteFile(filepath.Join(workspaceDir, p, "BUILD"), nil, 0666); err != nil {
					t.Error(err)
					t.FailNow()
				}
			}
			defer os.RemoveAll(workspaceDir)
			actual := findPackageName(context.Background(), workspaceDir, test.filename)
			if actual != test.wantPkgName {
				t.Errorf("%s: findPackageName(%s) = %s, want %s", test.desc, test.filename, actual, test.wantPkgName)
			}
		}()
	}
}

func TestCachingLoaderLoad(t *testing.T) {
	var tests = []struct {
		desc string
		// Each []string will be passed sequentially into Load()
		pkgNameSets [][]string

		// wantUnderlyingLoadCalls is the expected input passed into the underlying/wrapped loader.
		wantUnderlyingLoadCalls [][]string
		// wantPkgs is the expected result of the _last_ call to Load()
		wantPkgs map[string]*bazel.Package
	}{
		{
			desc:                    "basic sanity check: loading a,b and then c,d loads all packages",
			pkgNameSets:             [][]string{{"a", "b"}, {"c", "d"}},
			wantUnderlyingLoadCalls: [][]string{{"a", "b"}, {"c", "d"}},
			wantPkgs:                map[string]*bazel.Package{"c": {}, "d": {}},
		},
		{
			desc:                    "'b' is cached from the first load, so it isn't loaded on the second call to Load()",
			pkgNameSets:             [][]string{{"b"}, {"b", "c"}},
			wantUnderlyingLoadCalls: [][]string{{"b"}, {"c"}},
			wantPkgs:                map[string]*bazel.Package{"b": {}, "c": {}},
		},
		{
			desc:                    "Loading a,b,c then b,c causes no loading on the second call because everything is cached",
			pkgNameSets:             [][]string{{"a", "b", "c"}, {"b", "c"}},
			wantUnderlyingLoadCalls: [][]string{{"a", "b", "c"}},
			wantPkgs:                map[string]*bazel.Package{"b": {}, "c": {}},
		},
	}

	for _, tt := range tests {
		l := &loadertest.StubLoader{Pkgs: tt.wantPkgs}
		cl := NewCachingLoader(l)
		var got map[string]*bazel.Package
		for _, pkgNameSet := range tt.pkgNameSets {
			var err error
			got, err = cl.Load(context.Background(), pkgNameSet)
			if err != nil {
				t.Errorf("%s: Load(%v) has error %v, expected nil", tt.desc, pkgNameSet, err)
			}
		}
		if diff := cmp.Diff(l.RecordedCalls, tt.wantUnderlyingLoadCalls); diff != "" {
			t.Errorf("%s: Recorded calls diff: (-got +want)\n%s", tt.desc, diff)
		}
		if diff := cmp.Diff(got, tt.wantPkgs); diff != "" {
			t.Errorf("%s: Load() diff: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

// TestCachingLoaderLoadNonExistingPackage tests that
// non-existent packages are cached (so they're not requested again), but are not returned to the caller.
func TestCachingLoaderLoadNonExistingPackage(t *testing.T) {
	l := &loadertest.StubLoader{Pkgs: nil}
	cl := NewCachingLoader(l)
	var got map[string]*bazel.Package
	for i := 0; i < 2; i++ {
		pkgNameSet := []string{"a"}
		var err error
		got, err = cl.Load(context.Background(), pkgNameSet)
		if err != nil {
			t.Errorf("Load(%v) has error %v, expected nil", pkgNameSet, err)
		}
	}
	wantUnderlyingLoadCalls := [][]string{{"a"}}
	if diff := cmp.Diff(l.RecordedCalls, wantUnderlyingLoadCalls); diff != "" {
		t.Errorf("Recorded calls diff: (-got +want)\n%s", diff)
	}

	// Check that the _last_ call to cl.Load() returns the expected result.
	if len(got) > 0 {
		t.Errorf("Load unexpectedly returned packages: %v", got)
	}
}

func TestFilteringLoader(t *testing.T) {
	l := &loadertest.StubLoader{}
	fl := &FilteringLoader{l, map[string]bool{"third_party/maven/repository/central": true}}

	in := []string{"a", "b", "third_party/maven/repository/central"}
	_, err := fl.Load(context.Background(), in)
	if err != nil {
		t.Errorf("Load(%v) has error %v, expected nil", in, err)
	}
	wantUnderlyingLoadCalls := [][]string{{"a", "b"}}
	if diff := cmp.Diff(l.RecordedCalls, wantUnderlyingLoadCalls); diff != "" {
		t.Errorf("Recorded calls diff: (-got +want)\n%s", diff)
	}
}
