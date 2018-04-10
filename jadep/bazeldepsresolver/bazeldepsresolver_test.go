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

package bazeldepsresolver

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/loadertest"
	"github.com/google/go-cmp/cmp"
)

func TestEndToEnd(t *testing.T) {
	type attrs = map[string]interface{}

	tmpdir, err := ioutil.TempDir("", "bazel_deps_resolver")
	if err != nil {
		t.Errorf("error called ioutil.TempDir: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	root := filepath.Join(tmpdir, "root")
	externalRepositoriesRoot := filepath.Join(root, "external")
	workspace := filepath.Join(root, "workspace")
	thirdPartyDir := "thirdparty/jvm"

	type jar struct {
		fileName string
		files    []string
	}
	type newResolverArgs struct {
		pkgs map[string]*bazel.Package
		jars []jar
	}
	tests := []struct {
		name            string
		newResolverArgs newResolverArgs
		classNames      []jadeplib.ClassName
		want            map[jadeplib.ClassName][]bazel.Label
	}{
		{
			name: "basic",
			newResolverArgs: newResolverArgs{
				pkgs: map[string]*bazel.Package{
					"thirdparty/jvm/guava": {
						Path: filepath.Join(workspace, "thirdparty/jvm/guava"),
						Rules: map[string]*bazel.Rule{
							"guava": {
								Schema:  "java_library",
								PkgName: "thirdparty/jvm/guava",
								Attrs:   attrs{"name": "guava", "exports": []string{"//external:guava"}},
							},
						},
					},
					"thirdparty/jvm/org/junit": {
						Path: filepath.Join(workspace, "thirdparty/jvm/org/junit"),
						Rules: map[string]*bazel.Rule{
							"junit": {
								Schema:  "java_library",
								PkgName: "thirdparty/jvm/org/junit",
								Attrs:   attrs{"name": "junit", "exports": []string{"//external:junit"}},
							},
						},
					},
					"external": {
						Path: workspace,
						Rules: map[string]*bazel.Rule{
							"guava": {
								Schema:  "bind",
								PkgName: "external",
								Attrs:   attrs{"actual": "@guava//jar:guava"},
							},
							"junit": {
								Schema:  "bind",
								PkgName: "external",
								Attrs:   attrs{"actual": "@junit//jar:junit"},
							},
						},
					},
					"@guava//jar": {
						Path: filepath.Join(externalRepositoriesRoot, "guava/jar"),
						Rules: map[string]*bazel.Rule{
							"guava": {
								Schema:  "java_import",
								PkgName: "@guava//jar",
								Attrs:   attrs{"jars": []string{"guava.jar"}},
							},
						},
					},
					"@junit//jar": {
						Path: filepath.Join(externalRepositoriesRoot, "junit/jar"),
						Rules: map[string]*bazel.Rule{
							"junit": {
								Schema:  "java_import",
								PkgName: "@junit//jar",
								Attrs:   attrs{"jars": []string{"junit.jar"}},
							},
						},
					},
				},
				jars: []jar{
					{
						fileName: filepath.Join(externalRepositoriesRoot, "guava/jar/guava.jar"),
						files:    []string{"com/ImmutableList.class"},
					},
					{
						fileName: filepath.Join(externalRepositoriesRoot, "junit/jar/junit.jar"),
						files:    []string{"com/RunWith.class"},
					},
				},
			},
			classNames: []jadeplib.ClassName{"com.ImmutableList", "com.RunWith", "com.Unknown"},
			want: map[jadeplib.ClassName][]bazel.Label{
				"com.ImmutableList": {"//thirdparty/jvm/guava:guava"},
				"com.RunWith":       {"//thirdparty/jvm/org/junit:junit"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.RemoveAll(root)

			// Create BUILD files.
			err := createBuildFileDir(tt.newResolverArgs.pkgs)
			if err != nil {
				t.Fatal(err)
			}

			// Write Jar files.
			for _, jar := range tt.newResolverArgs.jars {
				err := writeZipFile(jar.fileName, jar.files)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Create resolver.
			loader := &loadertest.StubLoader{Pkgs: tt.newResolverArgs.pkgs}
			resolver, err := NewResolver(context.Background(), workspace, thirdPartyDir, loader)
			if err != nil {
				t.Fatalf("NewResolver: got err = %v, want nil", err)
			}

			// Resolve class names.
			got, err := resolver.Resolve(context.Background(), tt.classNames, nil)
			if err != nil {
				t.Fatalf("Resolver: got err = %v, want nil", err)
			}

			gotLabels := make(map[jadeplib.ClassName][]bazel.Label)
			for cls, rules := range got {
				for _, r := range rules {
					gotLabels[cls] = append(gotLabels[cls], r.Label())
				}
			}
			if diff := cmp.Diff(gotLabels, tt.want); diff != "" {
				t.Errorf("Resolve(%s) diff: (-got +want)\n%s", tt.classNames, diff)
			}
		})
	}
}

func createBuildFileDir(pkgs map[string]*bazel.Package) error {
	for _, pkg := range pkgs {
		err := os.MkdirAll(pkg.Path, 0700)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(fileName string, zipFileNames []string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for _, fileName := range zipFileNames {
		f, err := w.Create(fileName)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(""))
		if err != nil {
			return err
		}
	}

	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}
