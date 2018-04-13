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

// Package listclassesinjar lists class names in Jar files.
package listclassesinjar

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
)

// List returns the list of Java class names in the Jar named fileName.
// Only top-level class names are returned.
func List(fileName string) ([]jadeplib.ClassName, error) {
	var result []jadeplib.ClassName

	r, err := zip.OpenReader(fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s:\n%v", fileName, err)
	}
	defer r.Close()

	for _, f := range r.File {
		fn := f.Name
		if strings.Contains(fn, "$") || !strings.HasSuffix(fn, ".class") || strings.HasSuffix(fn, "/package-info.class") {
			continue
		}
		c := strings.Replace(strings.TrimSuffix(fn, ".class"), "/", ".", -1)
		result = append(result, jadeplib.ClassName(c))
	}
	return result, nil
}
