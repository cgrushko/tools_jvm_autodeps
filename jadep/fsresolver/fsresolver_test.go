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

package fsresolver

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/pkgloaderfakes"
	"github.com/google/go-cmp/cmp"
)

func TestClassToFiles(t *testing.T) {
	var tests = []struct {
		input jadeplib.ClassName
		want  []string
	}{
		{
			"Foo",
			[]string{"java/Foo.java", "test/Foo.java"},
		},
		{
			"com.google.devtools.build.lib.view.python.PythonProtoAspect",
			[]string{
				"java/com/google/devtools/build/lib/view/python/PythonProtoAspect.java",
				"test/com/google/devtools/build/lib/view/python/PythonProtoAspect.java",
			},
		},
		{
			"com.google.hello",
			[]string{"java/com/google/hello.java", "test/com/google/hello.java"},
		},
		{
			"foo.java",
			[]string{"java/foo/java.java", "test/foo/java.java"},
		},
		{
			"hello.Foo.ok",
			[]string{"java/hello/Foo/ok.java", "test/hello/Foo/ok.java"},
		},
		{
			"com.google.Foo",
			[]string{"java/com/google/Foo.java", "test/com/google/Foo.java"},
		},
	}
	for _, test := range tests {
		got := classToFiles([]string{"java/", "test"}, test.input)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("classToFile(%s) = %s, want %s", test.input, got, test.want)
		}
	}
}

func TestResolve(t *testing.T) {
	type Attrs = map[string]interface{}

	workDir, err := createWorkspace()
	if err != nil {
		t.Error(err)
	}
	var tests = []struct {
		classnames   []jadeplib.ClassName
		existingPkgs map[string]*bazel.Package
		want         map[jadeplib.ClassName][]*bazel.Rule
	}{
		// Basic test where there is only one rule with the class name in its sources.
		{
			[]jadeplib.ClassName{"x.Foo"},
			map[string]*bazel.Package{
				"java/x": pkgloaderfakes.Pkg([]*bazel.Rule{pkgloaderfakes.JavaLibrary("java/x", "Foo", []string{"Foo.java"}, nil, nil)}),
			},
			map[jadeplib.ClassName][]*bazel.Rule{
				"x.Foo": {pkgloaderfakes.JavaLibrary("java/x", "Foo", []string{"Foo.java"}, nil, nil)},
			},
		},
		// Basic test where there is no rule with the class name in its sources.
		{
			[]jadeplib.ClassName{"m.Foo"},
			map[string]*bazel.Package{
				"java/m": pkgloaderfakes.Pkg([]*bazel.Rule{pkgloaderfakes.JavaLibrary("java/m", "Boo", nil, nil, nil)}),
			},
			map[jadeplib.ClassName][]*bazel.Rule{},
		},
		// Assert that the unresolved classnames slice gets populated when the classname has no rule that can satisfy it.
		{
			[]jadeplib.ClassName{"f.Foo"},
			nil,
			map[jadeplib.ClassName][]*bazel.Rule{},
		},
		// This tests the exports functionality.
		{
			[]jadeplib.ClassName{"y.Foo"},
			map[string]*bazel.Package{
				"java/y": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("java/y", "Foo", []string{"Foo.java"}, nil, nil),
					pkgloaderfakes.JavaLibrary("java/y", "Bar", nil, nil, []string{"Foo"}),
					pkgloaderfakes.JavaLibrary("java/y", "Bez", nil, nil, []string{"Foo", "Bar"}),
				}),
			},
			map[jadeplib.ClassName][]*bazel.Rule{
				"y.Foo": {
					pkgloaderfakes.JavaLibrary("java/y", "Bar", nil, nil, []string{"Foo"}),
					pkgloaderfakes.JavaLibrary("java/y", "Bez", nil, nil, []string{"Foo", "Bar"}),
					pkgloaderfakes.JavaLibrary("java/y", "Foo", []string{"Foo.java"}, nil, nil),
				},
			},
		},
		// Tests that we only return java_library rules.
		{
			[]jadeplib.ClassName{"a.Foo"},
			map[string]*bazel.Package{
				"java/a": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("java/a", "Foo", []string{"Foo.java", "Bazel.java"}, nil, nil),
					pkgloaderfakes.JavaBinary("java/a", "D", []string{"Foo.java", "Foo_test.java"}, nil, nil),
				}),
			},
			map[jadeplib.ClassName][]*bazel.Rule{
				"a.Foo": {pkgloaderfakes.JavaLibrary("java/a", "Foo", []string{"Foo.java", "Bazel.java"}, nil, nil)},
			},
		},
		// This tests resolving multiple class names.
		{
			[]jadeplib.ClassName{"b.Foo", "b.Jadep"},
			map[string]*bazel.Package{
				"java/b": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("java/b", "Foo", []string{"Foo.java", "Bazel.java"}, nil, nil),
					pkgloaderfakes.JavaLibrary("java/b", "G", []string{"Hello.java", "Jadep.java"}, nil, nil),
				}),
			},
			map[jadeplib.ClassName][]*bazel.Rule{
				"b.Foo":   {pkgloaderfakes.JavaLibrary("java/b", "Foo", []string{"Foo.java", "Bazel.java"}, nil, nil)},
				"b.Jadep": {pkgloaderfakes.JavaLibrary("java/b", "G", []string{"Hello.java", "Jadep.java"}, nil, nil)},
			},
		},
		// This tests multiple class name inputs with multiple build files.
		{
			[]jadeplib.ClassName{"d.Foo", "d.Jadep", "c.Foo", "c.Jadep"},
			map[string]*bazel.Package{
				"java/c": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("java/c", "Foo", []string{"Bazel.java", "Foo.java"}, nil, nil),
					pkgloaderfakes.JavaLibrary("java/c", "G", []string{"Hello.java", "Jadep.java"}, nil, nil),
				}),
				"java/d": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("java/d", "Foo", []string{"Bazel.java", "Foo.java"}, nil, nil),
					pkgloaderfakes.JavaLibrary("java/d", "G", []string{"Hello.java", "Jadep.java"}, nil, nil),
				}),
			},
			map[jadeplib.ClassName][]*bazel.Rule{
				"c.Foo":   {pkgloaderfakes.JavaLibrary("java/c", "Foo", []string{"Bazel.java", "Foo.java"}, nil, nil)},
				"c.Jadep": {pkgloaderfakes.JavaLibrary("java/c", "G", []string{"Hello.java", "Jadep.java"}, nil, nil)},
				"d.Foo":   {pkgloaderfakes.JavaLibrary("java/d", "Foo", []string{"Bazel.java", "Foo.java"}, nil, nil)},
				"d.Jadep": {pkgloaderfakes.JavaLibrary("java/d", "G", []string{"Hello.java", "Jadep.java"}, nil, nil)},
			},
		},
		// A java_library srcs a filegroup which srcs Foo.java. Assert that we return the java_library as if it srcs-ed Foo.java directly.
		{
			classnames: []jadeplib.ClassName{"x.Foo"},
			existingPkgs: map[string]*bazel.Package{
				"java/x": pkgloaderfakes.Pkg([]*bazel.Rule{
					bazel.NewRule("java_library", "java/x", "Foo", Attrs{"srcs": []string{"Bar"}}),
					bazel.NewRule("filegroup", "java/x", "Bar", Attrs{"srcs": []string{"Foo.java"}}),
				}),
			},
			want: map[jadeplib.ClassName][]*bazel.Rule{
				"x.Foo": {bazel.NewRule("java_library", "java/x", "Foo", Attrs{"srcs": []string{"Bar"}})},
			},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			var pkgNames []string
			for p := range test.existingPkgs {
				pkgNames = append(pkgNames, p)
			}
			cleanup, err := createBuildFileDir(t, pkgNames, workDir)
			if err != nil {
				t.Error(err)
			}
			defer cleanup()
			resolver := NewResolver([]string{"java/", "javatest"}, workDir, &testLoader{test.existingPkgs})
			actual, err := resolver.Resolve(context.Background(), test.classnames, nil)
			if err != nil {
				t.Errorf("Resolve(%s) failed: %v. On iteration %v.", test.classnames, err, i)
				return
			}
			for _, rules := range actual {
				sort.Slice(rules, func(i, j int) bool { return rules[i].Label() < rules[j].Label() })
			}
			if diff := cmp.Diff(actual, test.want); diff != "" {
				t.Errorf("Resolve(%s) diff: (-got +want)\n%s", test.classnames, diff)
			}
		})
	}
}

// TestResolvePackageNotReturned tests that Resolve() tolerates the Loader not returning a package it requested.
// It creates java/x/BUILD, which makes Resolve request java/x from Loader. But, the Loader returns nothing.
// This simulates loading packages that are so malformed that a PackageLoader will return nothing for them.
func TestResolvePackageNotReturned(t *testing.T) {
	workDir, err := createWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)
	err = os.MkdirAll(filepath.Join(workDir, filepath.FromSlash("java/x")), 0700)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(workDir, filepath.FromSlash("java/x/BUILD")), []byte("#"), 0500)
	if err != nil {
		t.Fatal(err)
	}

	resolver := NewResolver([]string{"java/", "javatest"}, workDir, &testLoader{nil})
	got, err := resolver.Resolve(context.Background(), []jadeplib.ClassName{"x.Foo"}, nil)
	if err != nil {
		t.Fatalf("Resolve returned error %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("Resolve returned %v, want empty map", got)
	}
}

func BenchmarkResolve(b *testing.B) {
	existingPkgs := make(map[string]*bazel.Package)
	for i := 0; i < 100; i++ {
		pkgName := fmt.Sprintf("java/x%d", i)
		var rules []*bazel.Rule
		for j := 0; j < 100; j++ {
			rules = append(rules, pkgloaderfakes.JavaLibrary(pkgName, fmt.Sprintf("Foo%d", j), []string{fmt.Sprintf("Foo%d.java", j)}, nil, nil))
		}
		existingPkgs[pkgName] = pkgloaderfakes.Pkg(rules)
	}

	var classNames []jadeplib.ClassName
	for i := 0; i < 500; i++ {
		classNames = append(classNames, jadeplib.ClassName(fmt.Sprintf("x%d.Foo0", i)))
	}

	workDir, err := createWorkspace()
	if err != nil {
		b.Error(err)
	}
	var pkgNames []string
	for p := range existingPkgs {
		pkgNames = append(pkgNames, p)
	}
	cleanup, err := createBuildFileDir(nil, pkgNames, workDir)
	if err != nil {
		b.Error(err)
	}
	defer cleanup()
	resolver := NewResolver([]string{"java/", "javatest"}, workDir, &testLoader{existingPkgs})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.Resolve(context.Background(), classNames, nil)
	}
}

func createWorkspace() (string, error) {
	root, err := ioutil.TempDir("", "jadep")
	if err != nil {
		return "", fmt.Errorf("error called ioutil.TempDir: %v", err)
	}
	workDir := filepath.Join(root, "google3")
	if err := os.MkdirAll(filepath.Join(workDir, "tools/build_rules"), 0700); err != nil {
		return "", fmt.Errorf("error called MkdirAll: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(workDir, "tools/build_rules/BUILD"), nil, 0666); err != nil {
		return "", fmt.Errorf("error called WriteFile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(workDir, "tools/build_rules/prelude_-redacted-"), []byte("# must be non-empty"), 0666); err != nil {
		return "", fmt.Errorf("error called WriteFile: %v", err)
	}
	return workDir, nil
}

func createBuildFileDir(t *testing.T, pkgNames []string, workDir string) (func(), error) {
	for _, p := range pkgNames {
		err := os.MkdirAll(filepath.Join(workDir, filepath.FromSlash(p)), 0700)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(filepath.Join(workDir, filepath.FromSlash(p+"/BUILD")), []byte("#"), 0500)
		if err != nil {
			return nil, err
		}
	}
	return func() {
		for _, p := range pkgNames {
			os.RemoveAll(filepath.Join(workDir, p))
		}
	}, nil
}

type testLoader struct {
	pkgs map[string]*bazel.Package
}

func (l *testLoader) Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error) {
	result := make(map[string]*bazel.Package)
	for _, pkgName := range packages {
		if p, ok := l.pkgs[pkgName]; ok {
			result[pkgName] = p
		}
	}
	return result, nil
}
