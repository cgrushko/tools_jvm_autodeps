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

package parser

import (
	"testing"

	"github.com/bazelbuild/tools_jvm_autodeps/thirdparty/golang/parsers/parsers"
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const testPath = ""

func TestClasses(t *testing.T) {
	tests := []struct {
		desc   string
		source string
		want   []string
	}{
		{
			desc: "No duplicates",
			source: `import foo.Bar;
					import foo.Bar;
					class Dummy {
						void method() {
							new ClassB(ClassA.class);
							new ClassA();
						}
					}`,
			want: []string{"foo.Bar", "ClassA", "ClassB"},
		},
		{
			desc:   "Return values are considered",
			source: `class Dummy { ClassA method() {} }`,
			want:   []string{"ClassA"},
		},
		{
			desc: "Static methods cause classes to be included",
			source: `class Dummy {
						void method() {
							ImmutableList.of();
							com.google.common.collect.ImmutableMap.of();
							ExternalClassA.ExternalAInnerClass.create();
							com.google.common.ExternalClassB.ExternalBInnerClass.create();
							System.out.println();
							tempDeclaration.get();
							TAG_TYPES_TO_FILTER.run();
						}
					}`,
			want: []string{
				"com.google.common.collect.ImmutableMap",
				"ImmutableList",
				"ExternalClassA",
				"com.google.common.ExternalClassB",
			},
		},
		{
			desc: "Unresolved classes are assumed to be in the class's package",
			source: `package com.foo;
					class A extends B {}`,
			want: []string{"com.foo.B"},
		},
		{
			desc: "Field access causes classes to be included",
			source: `class Dummy {
						void method() {
							int i = com.google.common.ExternalClassC.SOME_CONSTANT;
							Object j = ExternalClassD.SOME_CONSTANT;
						}
					}`,
			want: []string{"com.google.common.ExternalClassC", "ExternalClassD"},
		},
		{
			desc: "Returns annotations",
			source: `class Dummy {
						@VisibleForTesting
						@Module(injects = Bla.class)
						@DefinedInSameFile
						void method() { }

						@interface DefinedInSameFile { }
					}`,
			want: []string{"VisibleForTesting", "Module", "Bla"},
		},
		{
			desc: "Ignore inner classes and self",
			source: `package com.company;
					import com.google.common.InnerClass;
					class Dummy {
						void method() {
							InnerClass.run();
							InnerClass.InnerInnerEnum.run();
							Dummy.staticMethod();
							com.company.Dummy.InnerClass.run();
						}

						public static class InnerClass {
							public enum InnerInnerEnum { }
						}
					}`,
			want: []string{"com.google.common.InnerClass"},
		},
		{
			desc: "Report fully-qualified class names",
			source: `package com.company;
					class Dummy {
						void method() {
							com.google.Foo.InnerClass.run();
						}
					}`,
			want: []string{"com.google.Foo"},
		},
		{
			desc: "Ignores inner classes called from lambda",
			source: `class Foo {
					static class UnsupportedException { }
					void foo() {
						Consumer<?> a = o -> new UnsupportedException();
					}
				}`,
			want: []string{"Consumer"},
		},
		{
			desc: "Ignores inner classes in synchronized",
			source: `class Dummy {
						private void f(OtherClass otherClass) {
							synchronized (otherClass) {
								Class c = InnerClass.class;
							}
						}
						public static class InnerClass { }
					}`,
			want: []string{"OtherClass"},
		},
		{
			desc: "Return class names containing static imports",
			source: `import static com.google.common.base.Preconditions.checkNotNull;
					import static com.foo.Bar.CONSTANT;
					import java.util.*;
					import static com.google.common.collect.Iterables.*;
					class A {
						A() {
							Object a = CONSTANT;
							checkNotNull();
						}
					}`,
			want: []string{"com.google.common.base.Preconditions", "com.foo.Bar", "com.google.common.collect.Iterables"},
		},
		{
			desc: "Ignore classes when in scope of declared type parameters",
			source: `class Foo<T> implements java.util.List<T> {
						class Bar<T> {
							void bla(T t) {}
						}
						void bla(T t) {}
						<U> void genericBla(U u) { U u = null; }
					}`,
			want: []string{"java.util.List"},
		},
		{
			desc: "Do not ignore type parameters appearing after a generic method",
			source: `class Foo {
						<U> void genericBla(U u) { U u = null; }
						abstract U concreteBla();
					}`,
			want: []string{"U"},
		},
		{
			desc: "Inner types of generic types",
			source: `class A {
					void foo(Map<Domain, Range>.Entry<Foo, Bar> bar) {}
				}`,
			want: []string{"Map", "Domain", "Range", "Foo", "Bar"},
		},
		{
			desc: "Throws clause is processed",
			source: `class A {
						void foo() throws IOException, java.io.FileNotFoundException {}
					}`,
			want: []string{"IOException", "java.io.FileNotFoundException"},
		},
		{
			desc: "All caps field names are not considered class names",
			source: `class A {
						FormattingLogger LOG = null;
						void foo() {
							LOG.warning();
						}
					}`,
			want: []string{"FormattingLogger"},
		},
		{
			desc:   "Dont report java lang inner classes",
			source: `class A extends Thread.UncaughtExceptionHandler { }`,
			want:   nil,
		},
		{
			desc: "report java.lang classes, even if they're passed in 'buildInClasses', if they appear in fully qualified form (either import or in code)",
			source: `import static java.lang.String.format;
					import java.lang.reflect.Method;`,
			want: []string{"java.lang.String", "java.lang.reflect.Method"},
		},
		{
			desc: "Java 8 features",
			source: `class A {
						void m() {
							Function o = (x) -> x;
						}
					}`,
			want: []string{"Function"},
		},
		{
			desc: "When an expression must be a class because of the grammar (org.g_Foo), but it doesn't look like a class name to us, we return it as is. If we can't tell from the grammar, we ignore it (org.g_Bar).",
			source: `package org;
					class A {
						void f() {
							new org.g_Foo();
							org.g_Bar.f();
						}
					}`,
			want: []string{"org.g_Foo"},
		},
		{
			desc: "We know from the grammar that imports are class names, so if they don't looks like class names we return them in full: we just don't know where to chop them to get the top-level part.",
			source: `import foo.bar;
		    		import foo.camelCase.Bar;`,
			want: []string{"foo.bar", "foo.camelCase.Bar"},
		},
		{
			desc: "Do not report the identifier part of on-demand imports if they're non-static (here, 'foo.bar'), because it's always a package name, not a class name. " +
				"However, static on-demand imports give us useful information, so we report them (here, foo.camelCase.Bar must be a class name)",
			source: `import foo.bar.*;
					import static foo.bla.*;
					import foo.baz.Baz.*;
		    		import static foo.camelCase.Bar.*;`,
			want: []string{"foo.bla", "foo.baz.Baz", "foo.camelCase.Bar"},
		},
		// TODO: Enable test.
		//     It's not the end of the world right now because parameters usually don't conform to the class name style so we don't report them.
		// {
		// 	desc: "Don't report formal parameters",
		// 	source: `class A{
		// 				void f(Object B) {
		// 					B.f();
		// 				}
		// 			}`,
		// 	want: nil,
		// },

		{
			desc: "Type parameter bounds are handled correctly.",
			source: `class A {
						<B extends BaseModel<B>> B m1() { }
						<B extends BaseModel<C>> B m2() { }
					}`,
			want: []string{"BaseModel", "C"},
		},
		{
			desc:   "Accept package names with digits",
			source: `import com.google.ads.proto.proto2api.Ads.LocalUniversalAdParams;`,
			want:   []string{"com.google.ads.proto.proto2api.Ads"},
		},
	}

	opt := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	ctx := context.Background()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			got, err := referencedClasses(ctx, testPath, tc.source, []string{"Class", "Object", "System", "Thread"})
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(got, tc.want, opt); diff != "" {
				t.Errorf("Result from referencedClasses() differs: (-got +want)\n%s", diff)
			}
		})
	}
}

func TestReferencedClassesIgnoresBuiltin(t *testing.T) {
	src := `class A{
				void f() {
					String s;
					Object o;
					System.out.println();
					Map m;
				}
			}`
	want := []string{"Map"}

	ctx := context.Background()
	got, err := referencedClasses(ctx, testPath, src, []string{"Object", "String", "System"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want, nil); diff != "" {
		t.Errorf("Result from referencedClasses() differs: (-got +want)\n%s", diff)
	}
}

func TestReferencedClassesSyntaxError(t *testing.T) {
	src := `class A{
				void f() {
			}`
	wantErr := parsers.SyntaxError{Description: "syntax error", Line: 3, Offset: 28, Length: 0}

	ctx := context.Background()
	_, err := referencedClasses(ctx, testPath, src, nil)
	if diff := cmp.Diff(err, wantErr, nil); diff != "" {
		t.Errorf("Error from referencedClasses() differs: (-got +want)\n%s", diff)
	}
}

func TestExtractClassNameFromQualifiedName(t *testing.T) {
	tests := []struct {
		parts   []string
		want    string
		wantIdx int
	}{
		{[]string{"com", "google", "Foo"}, "com.google.Foo", 2},
		{[]string{"com", "google", "foo", "Bar", "BAZ"}, "com.google.foo.Bar", 3},
		{[]string{"com", "google", "foo", "bar", "BAZ"}, "com.google.foo.bar.BAZ", 4},
		{[]string{"com", "google", "foo", "bar"}, "", -1},
		{[]string{"com", "google", "g_Foo"}, "", -1},
	}
	for _, tt := range tests {
		got, gotIdx := ExtractClassNameFromQualifiedName(tt.parts)
		if diff := cmp.Diff(gotIdx, tt.wantIdx); diff != "" {
			t.Errorf("Idx from ExtractClassNameFromQualifiedName(%v) differs: (-got +want)\n%s", tt.parts, diff)
		}
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Errorf("String from ExtractClassNameFromQualifiedName(%v) differs: (-got +want)\n%s", tt.parts, diff)
		}
	}
}
