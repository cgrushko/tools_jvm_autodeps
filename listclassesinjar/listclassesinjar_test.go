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

package listclassesinjar

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/google/go-cmp/cmp"
)

func TestList(t *testing.T) {
	type args struct {
		fileName     string
		zipFileNames []string
	}
	tests := []struct {
		name string
		args args
		want []jadeplib.ClassName
	}{
		{
			name: "basic",
			args: args{
				fileName: "",
				zipFileNames: []string{
					"com/foo/Bar.class",
					"com/Zoo.class",
					"com/google/common/truth/package-info.class",
					"META-INF/MANIFEST.MF",
					"META-INF/maven/com.google.truth/truth/pom.properties",
					"com/foo/Bar$1.class",
					"com/foo/Bar$1Inner.class",
					"com/foo/$1.class",
				},
			},
			want: []jadeplib.ClassName{
				"com.foo.Bar",
				"com.Zoo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := writeZipFile(tt.args.fileName, tt.args.zipFileNames)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(fn)

			got, err := List(fn)
			if err != nil {
				t.Errorf("List() error = %v, want nil", err)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("List() returned diff (-got +want): %v", diff)
			}
		})
	}
}

func writeZipFile(fileName string, zipFileNames []string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "zip")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()

	w := zip.NewWriter(tmpfile)
	for _, fileName := range zipFileNames {
		f, err := w.Create(fileName)
		if err != nil {
			return "", err
		}
		_, err = f.Write([]byte(""))
		if err != nil {
			return "", err
		}
	}

	err = w.Close()
	if err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}
