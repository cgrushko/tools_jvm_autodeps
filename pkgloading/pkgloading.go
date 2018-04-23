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

// Package pkgloading defines the package loading interface and implements functionality on top of it.
package pkgloading

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/compat"
)

// Loader loads BUILD files.
type Loader interface {
	// Load loads the named packages and returns a mapping from names to loaded packages, or an error if any item failed.
	Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error)
}

// CachingLoader is a concurrent duplicate-supressing cache for results from a loader.
// It wraps another loader L, and guarantees each requested package is loaded exactly once.
//
// For example, Load(a, b) and then Load(b, c) will result in the following calls to the underlying loader:
//   L.Load(a, b)
//   L.Load(c)
// Notice that 'b' is only requested once.
// As a corollary, Load(P) will not call the underlying L.Load() at all if all of the packages in P have been previously loaded.
//
// Note that if an error occurred when loading a set of packages, the failure will be cached and no loading will be re-attempted.
// In particular, it's possible to poison the cache for P by loading [P, BadPkg] first.
//
// CachingLoader is concurrency-safe as long as the underlying loader's Load function is concurrency-safe.
type CachingLoader struct {
	loader Loader
	mu     sync.Mutex // guards cache
	cache  map[string]*entry
}

// NewCachingLoader returns a new CachingLoader wrapped around a loader.
func NewCachingLoader(loader Loader) *CachingLoader {
	return &CachingLoader{loader: loader, cache: make(map[string]*entry)}
}

type entry struct {
	pkgName string
	res     result
	ready   chan struct{} // closed when res is ready
}

type result struct {
	value *bazel.Package
	err   error
}

// Load loads packages using an underlying loader.
// It will load each package at most once, and is safe to call concurrently.
// The returned error is a concatentation of all errors from calls to the underlying loader that occurred in order to load 'packages'.
func (l *CachingLoader) Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error) {
	var work, all []*entry
	l.mu.Lock()
	for _, p := range packages {
		e, ok := l.cache[p]
		if !ok {
			e = &entry{pkgName: p, ready: make(chan struct{})}
			l.cache[p] = e
			work = append(work, e)
		}
		all = append(all, e)
	}
	l.mu.Unlock()

	if len(work) > 0 {
		var pkgsToLoad []string
		for _, e := range work {
			pkgsToLoad = append(pkgsToLoad, e.pkgName)
		}
		result, err := l.loader.Load(ctx, pkgsToLoad)
		for _, e := range work {
			e.res.value = result[e.pkgName]
			e.res.err = err
			close(e.ready)
		}
	}

	result := make(map[string]*bazel.Package)
	var errors []interface{}
	for _, e := range all {
		<-e.ready
		if e.res.value != nil {
			result[e.pkgName] = e.res.value
		}
		if e.res.err != nil {
			errors = append(errors, e.res.err)
		}
	}
	if len(errors) != 0 {
		return nil, fmt.Errorf("Errors when loading packages: %v", errors)
	}
	return result, nil
}

// LoadRules loads the packages containing labels and returns the bazel.Rules represented by them.
func LoadRules(ctx context.Context, loader Loader, labels []bazel.Label) (map[bazel.Label]*bazel.Rule, map[string]*bazel.Package, error) {
	if len(labels) == 0 {
		return map[bazel.Label]*bazel.Rule{}, nil, nil
	}
	pkgs, err := loader.Load(ctx, distinctPkgs(labels))
	if err != nil {
		return nil, nil, err
	}

	result := make(map[bazel.Label]*bazel.Rule)
	for _, label := range labels {
		pkgName, ruleName := label.Split()
		if pkg, ok := pkgs[pkgName]; ok {
			if rule, ok := pkg.Rules[ruleName]; ok {
				result[bazel.Label(label)] = rule
			}
		}
	}
	return result, pkgs, nil
}

// LoadPackageGroups loads the packages containing labels and returns the bazel.Rules represented by them.
func LoadPackageGroups(ctx context.Context, loader Loader, labels []bazel.Label) (map[bazel.Label]*bazel.PackageGroup, error) {
	if len(labels) == 0 {
		return map[bazel.Label]*bazel.PackageGroup{}, nil
	}
	pkgs, err := loader.Load(ctx, distinctPkgs(labels))
	if err != nil {
		return nil, err
	}

	result := make(map[bazel.Label]*bazel.PackageGroup)
	for _, label := range labels {
		pkgName, groupName := label.Split()
		if pkg, ok := pkgs[pkgName]; ok {
			if g, ok := pkg.PackageGroups[groupName]; ok {
				result[bazel.Label(label)] = g
			}
		}
	}
	return result, nil
}

// distinctPkgs returns the set of unique packages mentioned in a set of labels.
func distinctPkgs(labels []bazel.Label) []string {
	pkgsSeen := make(map[string]bool)
	var result []string
	for _, l := range labels {
		pkgName, _ := l.Split()
		if !pkgsSeen[pkgName] {
			pkgsSeen[pkgName] = true
			result = append(result, pkgName)
		}
	}
	return result
}

// Siblings returns all the targets in all the packages that define the files in 'fileNames'.
// For example, if fileNames = {'foo/bar/Bar.java'}, and there's a BUILD file in foo/bar/, we return all the targets in the package defined by that BUILD file.
func Siblings(ctx context.Context, loader Loader, workspaceDir string, fileNames []string) (packages map[string]*bazel.Package, fileToPkgName map[string]string, err error) {
	tctx, endSpan := compat.NewLocalSpan(ctx, "Jade: Find BUILD packages of files")
	var wg sync.WaitGroup
	var mu sync.Mutex
	var pkgs []string
	pkgsSet := make(map[string]bool)
	fileToPkgName = make(map[string]string)
	for _, f := range fileNames {
		f := f
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p := findPackageName(tctx, workspaceDir, f); p != "" {
				mu.Lock()
				fileToPkgName[f] = p
				if !pkgsSet[p] {
					pkgs = append(pkgs, p)
					pkgsSet[p] = true
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	endSpan()
	packages, err = loader.Load(ctx, pkgs)
	return packages, fileToPkgName, err
}

// findPackageName finds the name of the package that the file is in.
func findPackageName(ctx context.Context, workspaceDir string, filename string) string {
	for dir := filepath.Dir(filename); dir != "."; dir = filepath.Dir(dir) {
		if _, err := compat.FileStat(ctx, filepath.Join(workspaceDir, dir, "BUILD")); !os.IsNotExist(err) {
			return dir
		}
	}
	return ""
}

// FilteringLoader is a Loader that loads using another Loader, after filtering the list of requested packages.
type FilteringLoader struct {
	// Loader is the underlying Loader which we delegate Load() calls to.
	Loader Loader

	// blacklistedPackages is a set of packages we will not load.
	// The underlying Loader will not be asked to load packages in this set.
	BlacklistedPackages map[string]bool
}

// Load sends an RPC to a PkgLoader service, requesting it to interpret 'packages' (e.g., "foo/bar" to interpret <root>/foo/bar/BUILD)
func (l *FilteringLoader) Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error) {
	var filtered []string
	for _, p := range packages {
		if !l.BlacklistedPackages[p] {
			filtered = append(filtered, p)
		}
	}

	return l.Loader.Load(ctx, filtered)
}
