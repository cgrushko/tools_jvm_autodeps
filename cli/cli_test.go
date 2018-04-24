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

package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/loadertest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	equateErrorMessage = cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	sortSlices = cmpopts.SortSlices(func(a, b string) bool { return a < b })
)

func TestFilesToParse(t *testing.T) {
	workspaceRoot := "/blabla/workspace/"
	tests := []struct {
		arg          string
		existingPkgs map[string]*bazel.Package
		want         []string
		wantErr      error
	}{
		{
			arg:  "Foo.java",
			want: []string{workspaceRoot + "Foo.java"},
		},
		{
			arg:  "/some/other/path/Foo.java",
			want: []string{"/some/other/path/Foo.java"},
		},
		{
			arg:          "//:Foo",
			existingPkgs: map[string]*bazel.Package{},
			wantErr:      fmt.Errorf("Rule not found: //:Foo"),
		},
		{
			arg: "//x:Foo",
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"Foo": bazel.NewRule("dontcare", "x", "Foo", map[string]interface{}{"srcs": []string{"Bar1.java", "subdir/Bar2.java", "//other:Bar3.java"}}),
					},
				},
			},
			want: []string{"x/Bar1.java", "x/subdir/Bar2.java", "other/Bar3.java"},
		},
	}

	for _, tt := range tests {
		got, err := FilesToParse(tt.arg, workspaceRoot, &loadertest.StubLoader{Pkgs: tt.existingPkgs})
		if diff := cmp.Diff(tt.wantErr, err, equateErrorMessage); diff != "" {
			t.Errorf("FilesToParse(%v) returned diff in error (-want +got):\n%s", tt.arg, diff)
		}
		if diff := cmp.Diff(tt.want, got); diff != "" {
			t.Errorf("FilesToParse(%v) returned diff (-want +got):\n%s", tt.arg, diff)
		}
	}
}

func TestRulesToFix(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("Can't create temp directory:\n%v", err)
		return
	}
	workspaceRoot := filepath.Join(tmpDir, "jadep")

	tests := []struct {
		arg           string
		relWorkingDir string
		createFiles   []string
		existingPkgs  map[string]*bazel.Package
		want          []*bazel.Rule
		wantErr       error
	}{
		{
			arg: "//x:Foo",
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"Foo": bazel.NewRule("dontcare", "x", "Foo", nil),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("dontcare", "x", "Foo", nil),
			},
		},
		{
			arg:         "x/Foo.java",
			createFiles: []string{"x/BUILD"},
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"x": bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
			},
		},
		{
			arg:           "Foo.java",
			relWorkingDir: "x",
			createFiles:   []string{"x/BUILD"},
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"x": bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
			},
		},
		{
			arg:           filepath.Join(workspaceRoot, "x/Foo.java"),
			relWorkingDir: "doesn't matter",
			createFiles:   []string{"x/BUILD"},
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"x": bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
			},
		},
		{
			arg:     "/x/Foo.java",
			wantErr: fmt.Errorf(`"/x/Foo.java" is not a relative path nor in a subdirectory of %q`, workspaceRoot),
		},
	}

	for _, tt := range tests {
		func() {
			createFiles(t, workspaceRoot, tt.createFiles)
			defer os.RemoveAll(workspaceRoot)
			config := jadeplib.Config{Loader: &loadertest.StubLoader{Pkgs: tt.existingPkgs}, WorkspaceDir: workspaceRoot}
			got, err := RulesToFix(context.Background(), config, tt.relWorkingDir, tt.arg, nil, "")
			if diff := cmp.Diff(tt.wantErr, err, equateErrorMessage); diff != "" {
				t.Errorf("RulesToFix(%v) returned diff in error (-want +got):\n%s", tt.arg, diff)
			}
			if diff := cmp.Diff(tt.want, got, sortSlices); diff != "" {
				t.Errorf("RulesToFix(%v) returned diff (-want +got):\n%s", tt.arg, diff)
			}
		}()
	}
}

func TestRulesToFixCreatesNewRule(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("Can't create temp directory:\n%v", err)
		return
	}
	workspaceRoot := filepath.Join(tmpDir, "jadep")
	createFiles(t, workspaceRoot, []string{"WORKSPACE", "x/BUILD"})

	config := jadeplib.Config{Loader: &loadertest.StubLoader{}, WorkspaceDir: workspaceRoot}
	got, err := RulesToFix(context.Background(), config, "", "x/Foo.java", nil, "java_test")
	if err != nil {
		t.Errorf("RulesToFix returned error %v, want nil", err)
	}
	want := []*bazel.Rule{bazel.NewRule("java_test", "x", "Foo", map[string]interface{}{"srcs": []string{"Foo.java"}})}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("RulesToFix returned diff (-want +got):\n%s", diff)
	}

	b, err := ioutil.ReadFile(filepath.Join(workspaceRoot, "x/BUILD"))
	if err != nil {
		t.Fatal(err)
	}
	wantBuildContent := `java_test(
    name = "Foo",
    srcs = ["Foo.java"],
)
`
	if string(b) != wantBuildContent {
		t.Errorf("RulesToFix created a BUILD file with content\n%s\nwant\n%s", string(b), wantBuildContent)
	}
}

func TestRulesToFixCreatesNewRule_absFileName(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("Can't create temp directory:\n%v", err)
		return
	}
	workspaceRoot := filepath.Join(tmpDir, "jadep")
	createFiles(t, workspaceRoot, []string{"WORKSPACE", "x/BUILD"})

	config := jadeplib.Config{Loader: &loadertest.StubLoader{}, WorkspaceDir: workspaceRoot}
	got, err := RulesToFix(context.Background(), config, "", filepath.Join(workspaceRoot, "x/Foo.java"), nil, "java_test")
	if err != nil {
		t.Errorf("RulesToFix returned error %v, want nil", err)
	}
	want := []*bazel.Rule{bazel.NewRule("java_test", "x", "Foo", map[string]interface{}{"srcs": []string{"Foo.java"}})}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("RulesToFix returned diff (-want +got):\n%s", diff)
	}

	b, err := ioutil.ReadFile(filepath.Join(workspaceRoot, "x/BUILD"))
	if err != nil {
		t.Fatal(err)
	}
	wantBuildContent := `java_test(
    name = "Foo",
    srcs = ["Foo.java"],
)
`
	if string(b) != wantBuildContent {
		t.Errorf("RulesToFix created a BUILD file with content\n%s\nwant\n%s", string(b), wantBuildContent)
	}
}

func TestWorkspace(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Can't create temp directory:\n%v", err)
	}
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed:\n%v", err)
	}
	testRoot := filepath.Join(tmpDir, "jadep")

	tests := []struct {
		desc              string
		existingFiles     []string
		workingDir        string
		workspaceFlag     string
		wantRoot          string
		wantRelWorkingDir string
		wantErr           error
	}{
		{
			desc:              "-workspace='', working dir has a WORKSPACE file. Return it",
			existingFiles:     []string{filepath.Join(testRoot, "repo", "WORKSPACE")},
			workingDir:        filepath.Join(testRoot, "repo"),
			wantRelWorkingDir: "",
			wantRoot:          filepath.Join(testRoot, "repo"),
		},
		{
			desc:              "-workspace='', parent of working dir has a WORKSPACE file. Return it",
			existingFiles:     []string{filepath.Join(testRoot, "repo", "WORKSPACE")},
			workingDir:        filepath.Join(testRoot, "repo", "subdir"),
			wantRelWorkingDir: "subdir",
			wantRoot:          filepath.Join(testRoot, "repo"),
		},
		{
			desc:       "-workspace='', no parent of working dir has a WORKSPACE file. Return an error",
			workingDir: filepath.Join(testRoot, "repo"),
			wantErr:    fmt.Errorf("couldn't find a parent of %v that has a WORKSPACE file", filepath.Join(testRoot, "repo")),
		},
		{
			desc:              "-workspace points at a directory with a WORKSPACE file, and working dir doesn't contain WORKSPACE. Return the value of -workspace",
			existingFiles:     []string{filepath.Join(testRoot, "repo", "WORKSPACE")},
			workingDir:        filepath.Join(testRoot, "some", "where", "else"),
			workspaceFlag:     filepath.Join(testRoot, "repo"),
			wantRoot:          filepath.Join(testRoot, "repo"),
			wantRelWorkingDir: "",
		},
		{
			desc:          "-workspace points at a directory without a WORKSPACE file (though its parent does have WORKSPACE). Return an error.",
			existingFiles: []string{filepath.Join(testRoot, "repo", "WORKSPACE")},
			workingDir:    filepath.Join(testRoot, "some", "where", "else"),
			workspaceFlag: filepath.Join(testRoot, "repo", "subdir"),
			wantErr:       fmt.Errorf("directory %v has no file named WORKSPACE", filepath.Join(testRoot, "repo", "subdir")),
		},
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Couldn't get working directory: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			createFiles(t, "", tt.existingFiles)
			os.MkdirAll(tt.workingDir, os.ModePerm)
			defer os.RemoveAll(testRoot)

			if err := os.Chdir(tt.workingDir); err != nil {
				t.Fatalf("can't chdir: %v", err)
			}
			defer os.Chdir(originalWd)

			gotRoot, gotRelWorkingDir, err := Workspace(tt.workspaceFlag)
			if diff := cmp.Diff(tt.wantErr, err, equateErrorMessage); diff != "" {
				t.Fatalf("Workspace() returned diff in error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantRoot, gotRoot); diff != "" {
				t.Errorf("Workspace() returned diff in root (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantRelWorkingDir, gotRelWorkingDir); diff != "" {
				t.Errorf("Workspace() returned diff in relWorkingDir (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClassNamesToResolve(t *testing.T) {
	// Test that classNamesToResolve extracts top-level class names from --classnames, if possible.
	ctx := context.Background()
	in := []string{"com.google.Foo.BAZ", "com.google.g_Foo"}
	want := []jadeplib.ClassName{"com.google.Foo", "com.google.g_Foo"}
	got := ClassNamesToResolve(ctx, "", nil, "", in, nil, nil)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("classNamesToResolve with --classnames=%v differs: (-got +want)\n%s", in, diff)
	}
}

func createFiles(t *testing.T, workDir string, fileNames []string) func() {
	for _, f := range fileNames {
		os.MkdirAll(filepath.Join(workDir, filepath.Dir(f)), os.ModePerm)
		err := ioutil.WriteFile(filepath.Join(workDir, f), []byte(""), os.ModePerm)
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
