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

// Package parser provides functions to parse Java code and reason about its contents.
package parser

import (
	"bytes"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/ast"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/node"
	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/lang/java/parser/xrefs"

	lpb "github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/lang"

	// Import the java parser so it can register itself.
	_ "github.com/bazelbuild/tools_jvm_autodeps/jadep/thirdparty/golang/parsers/java"
)

// ReferencedClasses returns the set of class names that the provided Java source files reference.
// This includes (a) imports (b) simple names we think are class names, which are assumed to be in the same package (c) fully-qualified names.
// implicitImports is a sorted slice of classes that do not require an import. In Java, these are the classes in java.lang, such as "System" and "Integer".
func ReferencedClasses(ctx context.Context, javaFileNames []string, implicitImports []string) []jadeplib.ClassName {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var result []jadeplib.ClassName
	classNameSeen := make(map[string]bool)
	for _, fileName := range javaFileNames {
		fileName := fileName
		wg.Add(1)
		go func() {
			defer wg.Done()
			source, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Printf("Error reading %q:\n%v", fileName, err)
				return
			}

			classes, err := referencedClasses(ctx, fileName, string(source), implicitImports)
			if err != nil {
				log.Printf("Error parsing %q:\n%v", fileName, err)
				return
			}

			mu.Lock()
			for _, c := range classes {
				if !classNameSeen[c] {
					classNameSeen[c] = true
					result = append(result, jadeplib.ClassName(c))
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	return result
}

// referencedClasses returns the set of class names that a Java source code references.
// An error is returned if the source can't be parsed.
// The path parameter is only used for tagging, not for reading a file.
func referencedClasses(ctx context.Context, path, source string, builtInClasses []string) ([]string, error) {
	tree, err := ast.Build(ctx, lpb.Language_JAVA, path, source, ast.Options{})
	if err != nil {
		return nil, err
	}
	pkg := packageName(tree)
	resolver := xrefs.NewResolver(tree)
	bindings := resolver.Resolve()

	seen := make(map[string]bool)
	// Mark all classes defined in this file as already seen, so we don't report them.
	// This is a workaround for the fact that Resolve() doesn't bind fully-qualified names yet.
	rootST := resolver.SymbolTables[tree.Root()]
	if rootST != nil {
		for className := range rootST.Types {
			if pkg != "" {
				seen[pkg+"."+className] = true
			}
		}
	}

	var result []string
	visit := func(n ast.Node) {
		switch n.Type() {
		case node.JavaTypeName,
			node.JavaTypeOrExprName,
			node.JavaExprName:
			ids := n.ChildrenOfType(node.JavaIdentifier)
			if _, ok := bindings[ids[0]]; ok {
				// Symbol is defined somewhere in this class, no need to report it.
				break
			}
			className, idx := ExtractClassNameFromQualifiedName(idsToStrs(ids))
			if idx < 0 {
				if n.Type() != node.JavaTypeName {
					break
				}
				className = joinIDs(ids)
			}
			if idx == 0 {
				if isBuiltin(builtInClasses, className) {
					break
				}
				if pkg != "" {
					className = pkg + "." + className
				}
			}
			if !seen[className] {
				seen[className] = true
				result = append(result, className)
			}

		case node.JavaImport:
			name := n.Child(node.OneOf(node.JavaName, node.JavaNameStar))
			onDemand := name.Type() == node.JavaNameStar
			static := n.FirstChildOfType(node.JavaStatic).IsValid()
			ids := name.ChildrenOfType(node.JavaIdentifier)
			className, idx := ExtractClassNameFromQualifiedName(idsToStrs(ids))
			if idx < 0 {
				if onDemand && !static {
					break
				}
				className = joinIDs(ids)
			}
			if !seen[className] {
				seen[className] = true
				result = append(result, className)
			}
		}
	}

	tree.ForEach(node.Any, visit)
	return result, nil
}

// isBuiltin returns true iff 's' is in 'strings'.
// 'strings' is assumed to be sorted.
// It is intended to filter out built-in class names, such as String, Object, etc.
func isBuiltin(strings []string, s string) bool {
	i := sort.SearchStrings(strings, s)
	return i < len(strings) && strings[i] == s
}

// packageName returns the package name of a tree, if it exists.
// For example, "com.google.common.collect".
func packageName(tree *ast.Tree) string {
	p := tree.Root().FirstChildOfType(node.JavaPackage)
	return joinIDs(p.FirstChildOfType(node.JavaName).ChildrenOfType(node.JavaIdentifier))
}

// joinIDs joins the texts of a set of nodes which are assumed to have type node.JavaIdentifier.
func joinIDs(ids []ast.Node) string {
	var b bytes.Buffer
	for i, id := range ids {
		if i > 0 {
			b.WriteString(".")
		}
		b.WriteString(id.Text())
	}
	return b.String()
}

func idsToStrs(ids []ast.Node) []string {
	ret := make([]string, len(ids))
	for i, id := range ids {
		ret[i] = id.Text()
	}
	return ret
}

// ExtractClassNameFromQualifiedName returns a top-level class name from parts and the index of the top-level part.
// If the class has an unconventional name (see below), returns an empty string and -1.
// By "class name" we mean a Java name according to convention, e.g. com.google.Foo.
// That is, a potentially empty package prefix (all lower case), then a top-level class name (starting with a capital letter).
func ExtractClassNameFromQualifiedName(parts []string) (string, int) {
	var result []string
	for i, txt := range parts {
		result = append(result, txt)
		if looksLikeSimpleClassName(txt) {
			return strings.Join(result, "."), i
		}
		if isLowerCase(txt) {
			continue
		}
		return "", -1
	}
	return "", -1
}

func isLowerCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

// looksLikeSimpleClassName returns true if 's' has the form ^[A-Z][a-zA-Z0-9]*$.
func looksLikeSimpleClassName(s string) bool {
	if s == "" {
		return false
	}
	ranges := []*unicode.RangeTable{unicode.Digit, unicode.Number, unicode.Letter}
	for index, r := range s {
		if index == 0 {
			if !unicode.IsUpper(r) {
				return false
			}
		} else if !unicode.IsOneOf(ranges, r) {
			return false
		}
	}
	return true
}
