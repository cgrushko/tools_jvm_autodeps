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

package buildozer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
)

func TestRef(t *testing.T) {
	type Attrs = map[string]interface{}
	tests := []struct {
		desc string
		rule *bazel.Rule
		want string
	}{
		{
			desc: "Rule instantiated directly",
			rule: bazel.NewRule("schema", "x/y", "foo", nil),
			want: "//x/y:foo",
		},
		{
			desc: "Rule instantiated by a macro that has a name attribute",
			rule: bazel.NewRule("schema", "x/y", "foo_lib", Attrs{"generator_name": "foo", "generator_function": "some_macro"}),
			want: "//x/y:foo",
		},
		{
			desc: "Rule instantiated by a macro that doesn't have a name attribute",
			rule: bazel.NewRule("schema", "x/y", "foo_lib", Attrs{"generator_location": "x/y/BUILD:78", "generator_function": "some_macro"}),
			want: "//x/y:%78",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			got, err := Ref(tt.rule)
			if err != nil {
				t.Errorf("Ref returns error %v, want nil", err)
				return
			}
			if got != tt.want {
				t.Errorf("Ref returned %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewRule(t *testing.T) {
	type Attrs = map[string]interface{}
	type file struct{ fileName, content string }
	tests := []struct {
		rule     *bazel.Rule
		wantFile file
	}{
		{
			rule: bazel.NewRule("java_test", "javatests/com", "FooTest", Attrs{"srcs": []string{"FooTest.java"}}),
			wantFile: file{
				fileName: "javatests/com/BUILD",
				content: `java_test(
    name = "FooTest",
    srcs = ["FooTest.java"],
)
`,
			},
		},
		{
			rule: bazel.NewRule("java_library", "java/com", "Foo", Attrs{"srcs": []string{"Foo.java"}}),
			wantFile: file{
				fileName: "java/com/BUILD",
				content: `java_library(
    name = "Foo",
    srcs = ["Foo.java"],
)
`,
			},
		},
		{
			rule: bazel.NewRule("java_library", "", "Foo", Attrs{"srcs": []string{"Foo.java"}}),
			wantFile: file{
				fileName: "BUILD",
				content: `java_library(
    name = "Foo",
    srcs = ["Foo.java"],
)
`,
			},
		},
		{
			rule: bazel.NewRule("android_test", "javatests/android", "FooTest", Attrs{"srcs": []string{"FooTest.java"}}),
			wantFile: file{
				fileName: "javatests/android/BUILD",
				content: `android_test(
    name = "FooTest",
    srcs = ["FooTest.java"],
)
`,
			},
		},
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("Can't create temp directory:\n%v", err)
		return
	}
	workspaceRoot := filepath.Join(tmpDir, "repo")

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.rule.Label()), func(t *testing.T) {
			createFiles(t, workspaceRoot, []string{"WORKSPACE"})
			os.MkdirAll(filepath.Join(workspaceRoot, tt.rule.PkgName), os.ModePerm)
			defer os.RemoveAll(workspaceRoot)
			err := NewRule(workspaceRoot, tt.rule)
			if err != nil {
				t.Fatalf("NewRule() returned error %v, want nil", err)
			}
			b, err := ioutil.ReadFile(filepath.Join(workspaceRoot, tt.wantFile.fileName))
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.wantFile.content {
				t.Errorf("NewRule created file %s with content\n%s\nbut wanted\n%s", tt.wantFile.fileName, string(b), tt.wantFile.content)
			}
		})
	}
}

func TestAddDepsToRules(t *testing.T) {
	type file struct{ fileName, content string }
	tests := []struct {
		desc           string
		missingDeps    map[*bazel.Rule][]bazel.Label
		buildFile      string
		initialContent string
		wantContent    string
	}{
		{
			desc: "basic",
			missingDeps: map[*bazel.Rule][]bazel.Label{
				bazel.NewRule("java_library", "x", "Foo", nil):  {"//y:Bar1", "//y:Bar2"},
				bazel.NewRule("java_test", "x", "FooTest", nil): {"//y:BarTest"},
			},
			buildFile: "x/BUILD",
			initialContent: `
java_library(name = "Foo")
java_test(name = "FooTest")
`,
			wantContent: `java_library(
    name = "Foo",
    deps = [
        "//y:Bar1",
        "//y:Bar2",
    ],
)

java_test(
    name = "FooTest",
    deps = ["//y:BarTest"],
)
`,
		},
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("Can't create temp directory:\n%v", err)
		return
	}
	workspaceRoot := filepath.Join(tmpDir, "repo")

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			createFiles(t, workspaceRoot, []string{"WORKSPACE", tt.buildFile})
			defer os.RemoveAll(workspaceRoot)
			err := ioutil.WriteFile(filepath.Join(workspaceRoot, tt.buildFile), []byte(tt.initialContent), os.ModePerm)
			if err != nil {
				t.Fatal(err)
			}
			err = AddDepsToRules(workspaceRoot, tt.missingDeps)
			if err != nil {
				t.Fatalf("AddDepsToRules returned error = %v, want nil", err)
			}
			b, err := ioutil.ReadFile(filepath.Join(workspaceRoot, tt.buildFile))
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.wantContent {
				t.Errorf("AddDepsToRules created file %s with content\n%s\nbut wanted\n%s", tt.buildFile, string(b), tt.wantContent)
			}
		})
	}
}

func createFiles(t *testing.T, workDir string, fileNames []string) func() {
	for _, f := range fileNames {
		err := os.MkdirAll(filepath.Join(workDir, filepath.Dir(f)), os.ModePerm)
		if err != nil {
			t.Fatalf("Can't create directory %q:\n%v", filepath.Join(workDir, filepath.Dir(f)), err)
		}
		err = ioutil.WriteFile(filepath.Join(workDir, f), []byte("# empty"), os.ModePerm)
		if err != nil {
			t.Fatalf("Can't create file %q:\n%v", f, err)
		}
	}
	return func() {
		for _, f := range fileNames {
			os.Remove(f)
		}
	}
}
