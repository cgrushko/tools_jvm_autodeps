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

package xrefs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"flag"
	lpb "github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/lang"
	"github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/ast"
	"github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/node"
	"context"
	"github.com/google/go-cmp/cmp"

	// Import the java parser so it can register itself.
	_ "github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/java"
)

const testPath = ""

var (
	resolveBenchmarkFile = flag.String("resolve_benchmark_file", "", "Name of a .java file to use when benchmarking Resolve()")

	equateNodes = cmp.Comparer(func(x, y ast.Node) bool {
		return x.Tree() == y.Tree() && x.LocalID() == y.LocalID()
	})
)

type Want map[string]struct {
	types   map[string]string
	members map[string][]string
}

func TestBuildSymbolTables(t *testing.T) {
	var tests = []struct {
		desc   string
		source string
		want   Want
	}{
		{
			"Basic",
			`«1»class A { «2»class B { } int «3»field; }`,
			Want{
				"«root»": {
					types: map[string]string{"A": "«1»"},
				},
				"«1»": {
					types:   map[string]string{"B": "«2»"},
					members: map[string][]string{"field": {"«3»"}},
				},
			},
		},
		{
			"Everything together",
			`«1»public class Outer1 {
				int «2»field1;
				int «3»field2 = 3, «4»field3 = 5;
				«5»void method1() {}
				«6»void method1(Object dontcare) {}
				«7»class Inner1 {
					«8»class Inner1Inner1 {
					«9»void method1() {}
					}
				}
				«10»enum Enum1 {
					«11»CONSTANT1("2"), «12»CONSTANT2;
				}
				«13»interface Interface1 {
					«14»static class Interface1Inner1 {
					«15»void method1() { }
					}
				}
			}
			«16»class Outer2 {
				«17»void f() {
					Object dontcare = («18»new Object() {
						«19»void foo() { }
					});
				}
			}`,
			Want{
				"«root»": {
					types: map[string]string{"Outer1": "«1»", "Outer2": "«16»"},
				},
				"«1»": {
					types: map[string]string{"Inner1": "«7»", "Enum1": "«10»", "Interface1": "«13»"},
					members: map[string][]string{
						"field1": {"«2»"}, "field2": {"«3»"}, "field3": {"«4»"},
						"method1": {"«5»", "«6»"},
					},
				},
				"«7»": {
					types: map[string]string{"Inner1Inner1": "«8»"},
				},
				"«8»": {
					members: map[string][]string{"method1": {"«9»"}},
				},
				"«10»": {
					members: map[string][]string{"CONSTANT1": {"«11»"}, "CONSTANT2": {"«12»"}},
				},
				"«13»": {
					types: map[string]string{"Interface1Inner1": "«14»"},
				},
				"«14»": {
					members: map[string][]string{"method1": {"«15»"}},
				},
				"«16»": {
					members: map[string][]string{"f": {"«17»"}},
				},
				"«18»": {
					members: map[string][]string{"foo": {"«19»"}},
				},
			},
		},
		{
			"Type parameters",
			`«1»class A<«2»T> {
				«3»<«4»S>void foo(S t) { }
				«5»<«6»S>A() { }
			}`,
			Want{
				"«root»": {
					types: map[string]string{"A": "«1»"},
				},
				"«1»": {
					types:   map[string]string{"T": "«2»"},
					members: map[string][]string{"foo": {"«3»"}},
				},
				"«3»": {
					types: map[string]string{"S": "«4»"},
				},
				"«5»": {
					types: map[string]string{"S": "«6»"},
				},
			},
		},
		{
			desc: "Imports contrbute to the root's symbol table.",
			source: `«1»import static a.b.assertThat;
					«2»import static assertThat;
					«3»import static java.util.Comparator.*;
					«4»import java.io.IOException;
					«5»import java.util.*;
					«6»import static com.foo.Bar.CONSTANT;`,
			want: Want{
				"«root»": {
					types: map[string]string{"IOException": "«4»"},
					members: map[string][]string{
						"assertThat": {"«1»", "«2»"},
						"CONSTANT":   {"«6»"},
					},
				},
			},
		},
		{
			desc: "Annotation types",
			source: `«1»class A {
						«2»@interface Annotation { }
					}`,
			want: Want{
				"«root»": {
					types: map[string]string{"A": "«1»"},
				},
				"«1»": {
					types: map[string]string{"Annotation": "«2»"},
				},
			},
		},
		{
			desc: "Enum constants",
			source: `«1»enum A {
						«2»C {
							«3»class B {}
							int «4»f;
							«5»void m1() {};
						};
						«6»void m2() {};
					}`,
			want: Want{
				"«root»": {
					types: map[string]string{"A": "«1»"},
				},
				"«1»": {
					members: map[string][]string{"m2": {"«6»"}, "C": {"«2»"}},
				},
				"«2»": {
					types:   map[string]string{"B": "«3»"},
					members: map[string][]string{"f": {"«4»"}, "m1": {"«5»"}},
				},
			},
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		source, markers := extractMarkers(test.source)
		tree, err := ast.Build(ctx, lpb.Language_JAVA, testPath, source, ast.Options{Type: ast.Full})
		if err != nil {
			t.Error(err)
			continue
		}
		markersToNodes := markersToNodes(tree, markers, nil)
		wantST, err := buildWantSymbolTable(test.want, test.desc, markersToNodes)
		if err != nil {
			t.Error(err)
			continue
		}

		got := buildSymbolTables(tree)
		if diff := cmp.Diff(wantST, got, equateNodes); diff != "" {
			t.Errorf("%s: diff: (-want +got)\n%s", test.desc, diff)
		}
	}
}

func TestResolveTypes(t *testing.T) {
	var tests = []struct {
		desc   string
		source string
		want   map[string]string
	}{
		{
			desc: "Resolves sibling type",
			source: `class A {
						class Inner1 extends «&1»Inner2 {}
						«1»class Inner2 {}
					}`,
			want: map[string]string{"«&1»": "«1»"},
		},
		{
			desc:   "Resolves container type",
			source: `«1»interface A { class Inner1 extends «&1»A {} }`,
			want:   map[string]string{"«&1»": "«1»"},
		},
		{
			desc: "Resolves qualified name with multiple parts. Tolerates missing types (here, Goo)",
			source: `class A {
						«1»class Foo {
							«2»class Baz {}
						}
						class Bar {
							void f() {
								«&1»Foo.«&2»Baz.Goo a;
							}
						}
					}`,
			want: map[string]string{"«&1»": "«1»", "«&2»": "«2»"},
		},
		{
			desc: "Test that we resolve to the closest lexical scope",
			source: `«1»class A {
						class B {
							«2»class A {
								void f() {
									new «&2»A();
								}
							}
						}
						void f() {
							new «&1»A();
						}
					}`,
			want: map[string]string{"«&1»": "«1»", "«&2»": "«2»"},
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			source, markers := extractMarkers(test.source)
			tree, err := ast.Build(ctx, lpb.Language_JAVA, testPath, source, ast.Options{Type: ast.Full})
			if err != nil {
				t.Fatalf("ast.Build(%q) failed with %v", source, err)
			}
			want := transformMarkersMapToNodeMap(t, test.want, markersToNodes(tree, markers, wantJavaIdentifierForAts(markers)))
			got := make(map[ast.Node]ast.Node)
			NewResolver(tree).resolveTypes(got)
			got = intersect(got, want)
			if diff := cmp.Diff(want, got, equateNodes); diff != "" {
				t.Errorf("%s: diff: (-want +got)\n%s", test.desc, diff)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	var tests = []struct {
		desc   string
		source string
		want   map[string]string
	}{
		{
			desc: "Resolve field (goo) in an expression after 2 classes (Foo.Baz)." +
				"This ensures we start resolving fields with the correct type (Baz) after a chain of types (Foo.Baz)",
			source: `class A {
						«1»class Foo {
							«2»class Baz {
								Object «3»goo;
							}
						}
						class Bar {
							void f() {
								«&1»Foo.«&2»Baz.«&3»goo.f();
							}
						}
					}`,
			want: map[string]string{
				"«&1»": "«1»",
				"«&2»": "«2»",
				"«&3»": "«3»",
			},
		},
		{
			desc: "Resolve chain of field accesses (b.c.d). Note that fields are defined before their types",
			source: `class A {
						«&2»B «1»b;
						«2»static class B {
							«&4»C «3»c;
							«4»static class C {
								int «5»d;
							}
						}

						int i = «&1»b.«&3»c.«&5»d;
					}`,
			want: map[string]string{
				"«&1»": "«1»",
				"«&2»": "«2»",
				"«&3»": "«3»",
				"«&4»": "«4»",
				"«&5»": "«5»",
			},
		},
		{
			desc: "Resolve a field of an inner type (B.C)",
			source: `class A {
						B.C c;
						static class B {
							C c;
							static class C {
								int «1»d;
							}
						}

						int i = c.«&1»d;
					}`,
			want: map[string]string{
				"«&1»": "«1»",
			},
		},
		{
			desc: "Resolve local variables",
			source: `class A {
						int «1»n = 43;
						void f() {
							int «2»i = 17;
							int «3»j;
							for (int «4»k = 0; k < «&1»n; k++) {
								«&3»j = g(«&2»i + «&4»k);
							}
						}
					}`,
			want: map[string]string{
				"«&1»": "«1»",
				"«&2»": "«2»",
				"«&3»": "«3»",
				"«&4»": "«4»",
			},
		},
		{
			desc: "For's init clause creates a new block. Here, 'x = 5' resolves to 'int x = 0', not to 'int x = 5' or 'int x : new Integer..'.",
			source: `class A {
						int «1»x = 0;
						void f() {
							for (int «2»x = 5; «&2a»x < 10; «&2b»x++) System.out.println(«&2c»x);
							for (int «3»x : new Integer[]{1, 2}) System.out.println(«&3»x);
							«&1»x = 5;
						}
					}`,
			want: map[string]string{
				"«&1»":  "«1»",
				"«&2a»": "«2»",
				"«&2b»": "«2»",
				"«&2c»": "«2»",
				// TODO: Enable once bug is fixed.
				// "«&3»":  "«3»",
			},
		},
		{
			desc: "Switch-case creates a new block. Here, 'x = 5' resolves to 'int x = 0', not to 'int x = 5'.",
			source: `class A {
						int «1»x = 0;
						void f() {
							switch(0){
								case 5:
									int «2»x = 5;
									«&2»x++;
							}
							«&1»x = 5;
						}
					}`,
			want: map[string]string{
				"«&1»": "«1»",
				"«&2»": "«2»",
			},
		},
		{
			desc: "symbols are resolved to import statements that imported them.",
			source: `«1»import static com.foo.Bar.CONSTANT;
					«2»import com.foo.Foo;
					class A {
						A() {
							new «&2»Foo();
							Object a = «&1»CONSTANT;
						}
					}`,
			want: map[string]string{
				"«&1»": "«1»",
				"«&2»": "«2»",
			},
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			source, markers := extractMarkers(test.source)
			tree, err := ast.Build(ctx, lpb.Language_JAVA, testPath, source, ast.Options{Type: ast.Full})
			if err != nil {
				t.Fatalf("ast.Build(%q) failed with %v", source, err)
			}
			want := transformMarkersMapToNodeMap(t, test.want, markersToNodes(tree, markers, wantJavaIdentifierForAts(markers)))
			got := intersect(NewResolver(tree).Resolve(), want)
			if diff := cmp.Diff(want, got, equateNodes); diff != "" {
				t.Errorf("%s: diff: (-want +got)\n%s", test.desc, diff)
			}
		})
	}
}

func BenchmarkResolve(b *testing.B) {
	ctx := context.Background()
	if *resolveBenchmarkFile == "" {
		b.Fatal("Must provide a .java file name in --resolve_benchmark_file")
	}
	bytes, err := ioutil.ReadFile(*resolveBenchmarkFile)
	if err != nil {
		b.Fatalf("Error reading %q:\n%v", *resolveBenchmarkFile, err)
	}
	tree, err := ast.Build(ctx, lpb.Language_JAVA, *resolveBenchmarkFile, string(bytes), ast.Options{})
	if err != nil {
		b.Fatalf("Error parsing %q:\n%v", *resolveBenchmarkFile, err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewResolver(tree).Resolve()
	}
}

// wantJavaIdentifierForAts maps keys from 'markers' which have a "«&" prefix, to node.JavaIdentifier.
// The output is used in markersToNodes() to return identifiers for "at" markers.
func wantJavaIdentifierForAts(markers map[string]int) map[string]node.Type {
	wantNodeTypes := make(map[string]node.Type)
	for m := range markers {
		if strings.HasPrefix(m, "«&") {
			wantNodeTypes[m] = node.JavaIdentifier
		}
	}
	return wantNodeTypes
}

func transformMarkersMapToNodeMap(t *testing.T, m map[string]string, markersToNodes map[string]ast.Node) map[ast.Node]ast.Node {
	ret := make(map[ast.Node]ast.Node)
	for key, value := range m {
		u, ok := markersToNodes[key]
		if !ok {
			t.Errorf("Didn't find a node for marker %s", key)
		}
		v, ok := markersToNodes[value]
		if !ok {
			t.Errorf("Didn't find a node for marker %s", value)
		}
		ret[u] = v
	}
	return ret
}

// intersect returns the key-value pairs from m1 for which the key exists in m2.
// In other words, it removes keys from m1 that aren't in m2.
// It's used to filter what we get from e.g. Resolve() to only contain what we have in 'want' of tests.
func intersect(m1, m2 map[ast.Node]ast.Node) map[ast.Node]ast.Node {
	ret := make(map[ast.Node]ast.Node)
	for k, v := range m1 {
		if _, ok := m2[k]; ok {
			ret[k] = v
		}
	}
	return ret
}

// buildWantSymbolTable constructs a symbolTable from markers (e.g., "«1»").
func buildWantSymbolTable(want Want, testDesc string, markersToNodes map[string]ast.Node) (map[ast.Node]*SymbolTable, error) {
	result := make(map[ast.Node]*SymbolTable)
	for containing, contained := range want {
		curr := &SymbolTable{make(map[string]ast.Node), make(map[string][]ast.Node)}
		for name, marker := range contained.types {
			n, ok := markersToNodes[marker]
			if !ok {
				return nil, fmt.Errorf("%s: When constructing wantST, no such marker: %s", testDesc, marker)
			}
			curr.Types[name] = n
		}
		for name, markers := range contained.members {
			for _, marker := range markers {
				n, ok := markersToNodes[marker]
				if !ok {
					return nil, fmt.Errorf("%s: When constructing wantST, no such marker: %s", testDesc, marker)
				}
				curr.Members[name] = append(curr.Members[name], n)
			}
		}

		n, ok := markersToNodes[containing]
		if !ok {
			return nil, fmt.Errorf("%s: When constructing wantST, no such marker: %s", testDesc, containing)
		}
		result[n] = curr
	}
	return result, nil
}

// extractMarkers takes source code annotated with markers ("«i»"), and returns
// (1) the source code stripped of «i», and
// (2) a map from marker to the offset it starts at.
func extractMarkers(source string) (string, map[string]int) {
	var filteredSource bytes.Buffer
	markers := make(map[string]int)
	inMarker := false
	var currID string
	for _, rune := range source {
		switch rune {
		case '«':
			inMarker = true
			currID = string(rune)
		case '»':
			inMarker = false
			markers[currID+string(rune)] = filteredSource.Len()
		default:
			if inMarker {
				currID = currID + string(rune)
			} else {
				filteredSource.WriteRune(rune)
			}
		}
	}
	return filteredSource.String(), markers
}

// Constructs a map from marker ("«i»") to ast.Node, based on their locations in the tree.
// Finds the least specific node that begins at the marker, which is of type expectedTypes[«i»] if present. Otherwise, ignores type.
// The special marker "«root»" points at tree.Root().
func markersToNodes(tree *ast.Tree, markers map[string]int, expectedTypes map[string]node.Type) map[string]ast.Node {
	result := make(map[string]ast.Node)
	result["«root»"] = tree.Root()
	for marker, offset := range markers {
		expType := expectedTypes[marker]
		n := tree.Root()
		// find the least specific node that begins at 'offset' and has type expectedTypes[marker] (if specified).
	outer:
		for {
			for child := n.FirstChild(); child.IsValid(); child = child.NextSibling() {
				if child.Offset() == offset && (expType == node.NoType || child.Type() == expType) {
					n = child
					break outer
				}
				if child.Offset() <= offset && offset <= child.EndOffset() {
					n = child
					continue outer
				}
			}
			panic(fmt.Sprintf("Didn't find a node at offset %d for marker %s", offset, marker))
		}
		result[marker] = n
	}
	return result
}
