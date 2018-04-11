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

// Package fsresolver resolves class names to Bazel rules using the file system.
package fsresolver

import (
	"log"
	"path/filepath"
	"strings"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/filter"
	"github.com/bazelbuild/tools_jvm_autodeps/graphs"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
	"github.com/bazelbuild/tools_jvm_autodeps/vlog"
)

// Resolver uses the file system to resolve class names to Bazel rules.
type Resolver struct {
	// contentRoots specifies where the Java files are located.
	contentRoots []string
	// workspaceDir is a path to the root of a Bazel workspace.
	workspaceDir string

	// loader loads BUILD files.
	loader pkgloading.Loader
}

// NewResolver returns a new Resolver.
func NewResolver(contentRoots []string, workspaceDir string, loader pkgloading.Loader) *Resolver {
	return &Resolver{contentRoots, workspaceDir, loader}
}

// Name returns a description of the resolver.
func (r *Resolver) Name() string {
	return "file system"
}

// Resolve finds Bazel rules that provide the classes in
// 'classnames', by looking at the local filesystem. A classname
// 'a.b.c.D' is transformed into the filename '[content root]/a/b/c/D.java".
// We then look for a Bazel rule that has that filename in its 'srcs' attribute.
//
// Returns:
// (1) a map from each classname to a list of java_library rules that provide the classnames
// (2) a list of classnames that could not be resolved by this file system approach.
func (r *Resolver) Resolve(ctx context.Context, classNames []jadeplib.ClassName, consumingRules map[bazel.Label]map[bazel.Label]bool) (map[jadeplib.ClassName][]*bazel.Rule, error) {
	classToFile := make(map[jadeplib.ClassName][]string)
	var filenames []string

	for _, cls := range classNames {
		classToFiles := classToFiles(r.contentRoots, cls)
		classToFile[cls] = classToFiles
		filenames = append(filenames, classToFiles...)
	}

	packages, fileToPkgName, err := pkgloading.Siblings(ctx, r.loader, r.workspaceDir, filenames)
	if err != nil {
		return nil, err
	}

	result := make(map[jadeplib.ClassName][]*bazel.Rule)

	for _, cls := range classNames {
		for _, filename := range classToFile[cls] {
			if pkgName, ok := fileToPkgName[filename]; ok {
				pkg := packages[pkgName]
				if pkg == nil {
					vlog.V(3).Printf("Package %s for file %s was not returned from Loader", pkgName, filename)
					continue
				}
				relativeFilename, err := filepath.Rel(pkgName, filename)
				if err != nil {
					log.Printf("Error in filepath.Rel(%s, %s):%v", pkgName, filename, err)
					continue
				}
				graph := make(map[string][]string)

				for ruleName, rule := range pkg.Rules {
					for _, s := range rule.StringListAttr("exports") {
						graph[s] = append(graph[s], ruleName)
					}
					for _, src := range rule.StringListAttr("srcs") {
						if src == relativeFilename {
							graph[relativeFilename] = append(graph[relativeFilename], ruleName)
						}
						if r := pkg.Rules[src]; r != nil && r.Schema == "filegroup" {
							graph[src] = append(graph[src], ruleName)
						}
					}
				}

				graphs.DFS(graph, relativeFilename, func(node string) {
					if rule, ok := pkg.Rules[node]; ok {
						if filter.JavaDependencyRuleKinds[rule.Schema] {
							result[cls] = append(result[cls], rule)
						}
					}
				})
			}
		}
	}
	return result, nil
}

// classToFiles converts a class name into a file name by changing
// package to directory separators and adding the content
// root to the front of the class name and the ".java" to the back.
// For example if the contentRoot is java and the class name is
// com.Foo then the output will be java/com/Foo.java.
func classToFiles(contentRoots []string, className jadeplib.ClassName) []string {
	var result []string
	for _, root := range contentRoots {
		segments := append([]string{root}, strings.Split(string(className), ".")...)
		result = append(result, filepath.Join(segments...)+".java")
	}
	return result
}
