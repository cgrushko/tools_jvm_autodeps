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

// Package loadertest provides test Loaders.
package loadertest

import (
	"sort"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
)

// StubLoader is a Loader that returns a preset response.
// It also records the parameters to its Load function.
type StubLoader struct {
	Pkgs map[string]*bazel.Package

	// RecordedCalls[i] contains the 'packages' parameter passed to the i'th call to the Load method (after sorting).
	RecordedCalls [][]string
}

// Load returns bazel.Package objects that are both in packages and in StubLoader.Pkgs.
func (l *StubLoader) Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error) {
	l.recordCall(packages)

	result := make(map[string]*bazel.Package)
	for _, pkgName := range packages {
		if p, ok := l.Pkgs[pkgName]; ok {
			result[pkgName] = p
		}
	}
	return result, nil
}

func (l *StubLoader) recordCall(packages []string) {
	sortedPkgs := make([]string, len(packages))
	copy(sortedPkgs, packages)
	sort.Strings(sortedPkgs)
	l.RecordedCalls = append(l.RecordedCalls, sortedPkgs)
}
