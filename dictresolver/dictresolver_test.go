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

package dictresolver

import (
	"strings"
	"testing"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/future"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/loadertest"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloaderfakes"
	"github.com/google/go-cmp/cmp"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		desc                string
		existingPkgs        map[string]*bazel.Package
		dict                map[jadeplib.ClassName][]bazel.Label
		classNamesToResolve []jadeplib.ClassName
		consumingRules      map[bazel.Label]map[bazel.Label]bool
		expectedLoads       [][]string
		want                map[jadeplib.ClassName][]*bazel.Rule
	}{
		{
			desc: "when dict maps to nil, return a mapping to nil. " +
				"This is used by the JDK resolver, when we want some class namers to not require any Bazel deps at all.",
			dict:                map[jadeplib.ClassName][]bazel.Label{"java.lang.Thread": nil, "java.util.HashSet": nil, "java.util.concurrent.Executor": nil},
			classNamesToResolve: []jadeplib.ClassName{"java.lang.Thread", "com.Foo", "java.util.concurrent.Executor", "javax.inject.Inject"},
			want: map[jadeplib.ClassName][]*bazel.Rule{
				"java.lang.Thread":              nil,
				"java.util.concurrent.Executor": nil,
			},
		},
		{
			desc: "dict points to a label, see that we load it and return it.",
			existingPkgs: map[string]*bazel.Package{
				"foo": pkgloaderfakes.Pkg([]*bazel.Rule{
					bazel.NewRule("dontcare", "foo", "Foo", nil),
					bazel.NewRule("dontcare", "foo", "Foo2", nil),
				}),
			},
			dict:                map[jadeplib.ClassName][]bazel.Label{"com.Foo": {"//foo:Foo", "//foo:Foo2"}},
			classNamesToResolve: []jadeplib.ClassName{"com.Foo"},
			expectedLoads:       [][]string{{"foo"}},
			want: map[jadeplib.ClassName][]*bazel.Rule{
				"com.Foo": {
					bazel.NewRule("dontcare", "foo", "Foo", nil),
					bazel.NewRule("dontcare", "foo", "Foo2", nil),
				},
			},
		},
		{
			desc:                "Don't load packages for classes that are already satisfied, but return labels for them",
			existingPkgs:        nil,
			dict:                map[jadeplib.ClassName][]bazel.Label{"com.ImmutableList": {"//third_party/java_src:collect"}},
			classNamesToResolve: []jadeplib.ClassName{"com.ImmutableList"},
			consumingRules: map[bazel.Label]map[bazel.Label]bool{
				"//:consumer": {"//third_party/java_src:collect": true},
			},
			expectedLoads: nil,
			want: map[jadeplib.ClassName][]*bazel.Rule{
				"com.ImmutableList": {bazel.NewRule("", "third_party/java_src", "collect", nil)},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			loader := &loadertest.StubLoader{Pkgs: tt.existingPkgs}
			resolver := NewResolver("dict resolver", future.Immediate(tt.dict), loader)
			got, err := resolver.Resolve(context.Background(), tt.classNamesToResolve, tt.consumingRules)
			if err != nil {
				t.Fatalf("Unexpected from Resolve(%v) error:%v", tt.classNamesToResolve, err)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Resolve(%v) diff: (-got +want)\n%s", tt.classNamesToResolve, diff)
			}
			if diff := cmp.Diff(loader.RecordedCalls, tt.expectedLoads); diff != "" {
				t.Errorf("%s: Diffs in Load() calls to loader (-got +want):\n%s", tt.desc, diff)
			}
		})
	}
}

func TestReadDictFromCSV(t *testing.T) {
	tests := []struct {
		desc string
		csv  string
		want map[jadeplib.ClassName][]bazel.Label
	}{
		{
			desc: "records with only one column result in mappings to nil",
			csv:  `com.Foo`,
			want: map[jadeplib.ClassName][]bazel.Label{"com.Foo": nil},
		},
		{
			desc: "records with variable number of columns are ok",
			csv: `com.Foo,//:Foo1,//:Foo2
com.Bar,//:Bar
com.Zee`,
			want: map[jadeplib.ClassName][]bazel.Label{"com.Foo": {"//:Foo1", "//:Foo2"}, "com.Bar": {"//:Bar"}, "com.Zee": nil},
		},
		{
			desc: "invalid labels are silently ignored",
			csv:  `com.Foo,blabla`,
			want: map[jadeplib.ClassName][]bazel.Label{"com.Foo": nil},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ReadDictFromCSV(strings.NewReader(tt.csv))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("ReadDictFromCSV diff: (-got +want)\n%s", diff)
			}
		})
	}
}
