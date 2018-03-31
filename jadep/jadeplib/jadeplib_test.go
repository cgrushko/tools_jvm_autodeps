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

package jadeplib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/future"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/pkgloaderfakes"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/sortingdepsranker"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	equateErrorMessage = cmp.Comparer(func(x, y error) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Error() == y.Error()
	})

	// sortRuleKeys allows cmp.Diff to compare maps whose keys are *bazel.Rule.
	sortRuleKeys = cmpopts.SortMaps(func(x, y *bazel.Rule) bool { return x.Label() < y.Label() })

	publicAttr = map[string]interface{}{"visibility": []string{"//visibility:public"}}
)

func createBuildFileDir(t *testing.T, pkgNames []string, workDir string) func() {
	for _, p := range pkgNames {
		err := os.MkdirAll(filepath.Join(workDir, filepath.FromSlash(p)), 0700)
		if err != nil {
			t.Fatal(err)
		}
		err = ioutil.WriteFile(filepath.Join(workDir, filepath.FromSlash(p+"/BUILD")), []byte("#"), 0500)
		if err != nil {
			t.Fatal(err)
		}
	}
	return func() {
		for _, p := range pkgNames {
			os.RemoveAll(filepath.Join(workDir, p))
		}
	}
}

func TestResolveAll(t *testing.T) {
	var tests = []struct {
		desc                string
		resolvers           []Resolver
		classnamesToResolve []ClassName
		wantResolved        map[ClassName][]*bazel.Rule
		wantUnresolved      []ClassName
		wantErrors          map[Resolver]error
	}{
		{
			"basic",
			[]Resolver{
				&testResolver{[]ClassName{"a", "b", "c"}, map[ClassName][]*bazel.Rule{"a": {{PkgName: "p1"}}}},
				&testResolver{[]ClassName{"b", "c"}, map[ClassName][]*bazel.Rule{"b": {{PkgName: "p2"}}}},
			},
			[]ClassName{"a", "b", "c"},
			map[ClassName][]*bazel.Rule{"a": {{PkgName: "p1"}}, "b": {{PkgName: "p2"}}},
			[]ClassName{"c"},
			map[Resolver]error{},
		},
	}

	for _, tt := range tests {
		resolved, unresolved, errors := resolveAll(context.Background(), tt.resolvers, tt.classnamesToResolve, nil)
		if diff := cmp.Diff(resolved, tt.wantResolved); diff != "" {
			t.Errorf("%s: Diff in resolved (-got +want).\n%s", tt.desc, diff)
		}
		if diff := cmp.Diff(unresolved, tt.wantUnresolved); diff != "" {
			t.Errorf("%s: Diff in unresolved (-got +want).\n%s", tt.desc, diff)
		}
		if diff := cmp.Diff(errors, tt.wantErrors); diff != "" {
			t.Errorf("%s: Diff in errors (-got +want).\n%s", tt.desc, diff)
		}
	}
}

type testResolver struct {
	// List of classnames we expect this resolver to be called on.
	expectedRequested []ClassName

	// The response we'll return if 'expectedRequested' == actual requested.
	cannedResponse map[ClassName][]*bazel.Rule
}

func (r *testResolver) Name() string {
	return "TestResolver"
}

func (r *testResolver) Resolve(ctx context.Context, classNames []ClassName, consumingRules map[bazel.Label]map[bazel.Label]bool) (map[ClassName][]*bazel.Rule, error) {
	sort.Slice(classNames, func(i, j int) bool { return string(classNames[i]) < string(classNames[j]) })
	if !reflect.DeepEqual(classNames, r.expectedRequested) {
		return nil, fmt.Errorf("Unexpected class names to resolve. Requested: %v, Expected: %#v", classNames, r.expectedRequested)
	}
	return r.cannedResponse, nil
}

func TestMissingDeps(t *testing.T) {
	type Attrs = map[string]interface{}

	workDir := createWorkspace(t)
	var tests = []struct {
		desc         string
		fileName     string
		classNames   []ClassName
		existingPkgs map[string]*bazel.Package
		resolvers    []Resolver
		// Outputs:
		wantMissing    map[*bazel.Rule]map[ClassName][]bazel.Label
		wantUnresolved []ClassName
	}{
		{
			"Scenario: Java file references 3 classes: one that's already satisfied, one we can satisfy using //p2:dep2, and one that we can't resolve at all." +
				"We expect that MissingDeps reports that com.CanBeSatisfied can be satisfied by //p2:dep2.",
			"java/Foo.java",
			[]ClassName{"com.AlreadySatisfied", "com.CanBeSatisfied", "com.Unresolved"},
			map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg([]*bazel.Rule{pkgloaderfakes.JavaLibrary("java", "Foo", []string{"Foo.java"}, []string{"//p1:dep1"}, nil)}),
			},
			[]Resolver{
				&testResolver{
					[]ClassName{"com.AlreadySatisfied", "com.CanBeSatisfied", "com.Unresolved"},
					map[ClassName][]*bazel.Rule{
						"com.AlreadySatisfied": {bazel.NewRule("java_library", "p1", "dep1", publicAttr)},
						"com.CanBeSatisfied":   {bazel.NewRule("java_library", "p2", "dep2", publicAttr)},
					},
				},
			},
			// Outputs:
			map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}, "deps": []string{"//p1:dep1"}}): {"com.CanBeSatisfied": {"//p2:dep2"}},
			},
			[]ClassName{"com.Unresolved"},
		},
		{
			"Scenario: the package that contains 'filename' has two rules, but only one srcs 'filename', " +
				"so we shouldn't mention the other rule in MissingDeps' results.",
			"java/Foo.java",
			[]ClassName{"com.Foo"},
			map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaLibrary("java", "Foo", []string{"Foo.java"}, nil, nil),
					pkgloaderfakes.JavaLibrary("java", "Bar", []string{"Bar.java"}, nil, nil),
				}),
			},
			[]Resolver{
				&testResolver{
					[]ClassName{"com.Foo"},
					map[ClassName][]*bazel.Rule{"com.Foo": {bazel.NewRule("java_library", "p1", "dep1", publicAttr)}},
				},
			},
			// Outputs:
			map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}}): {"com.Foo": {"//p1:dep1"}},
			},
			nil,
		},
		{
			"Scenario: //java:Lib srcs two files, and we run on one of them, Foo.java. " +
				"The file references classes in the other file, com.Bar. " +
				"We want to assert that Lib is considered to provide all classes from Java files it srcs. " +
				"In other words, assert that Lib isn't added to itself",
			"java/Foo.java",
			[]ClassName{"com.Bar"},
			map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg([]*bazel.Rule{
					bazel.NewRule("java_library", "java", "Lib", publicAttr),
				}),
			},
			[]Resolver{
				&testResolver{
					[]ClassName{"com.Bar"},
					map[ClassName][]*bazel.Rule{"com.Bar": {pkgloaderfakes.JavaLibrary("java", "Lib", nil, nil, nil)}},
				},
			},
			// Outputs:
			map[*bazel.Rule]map[ClassName][]bazel.Label{},
			nil,
		},
		{
			"Don't report missing deps on filegroup() rules",
			"java/Foo.java",
			[]ClassName{"com.Bar"},
			map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg([]*bazel.Rule{
					pkgloaderfakes.JavaBinary("java", "Foo1", []string{"Foo.java"}, nil, nil),
					{
						Schema:  "filegroup",
						PkgName: "java",
						Attrs: map[string]interface{}{
							"name": "Foo2",
							"srcs": []string{"Foo.java"},
						},
					},
				}),
			},
			[]Resolver{
				&testResolver{
					[]ClassName{"com.Bar"},
					map[ClassName][]*bazel.Rule{"com.Bar": {bazel.NewRule("java_library", "java", "Bar", publicAttr)}},
				},
			},
			// Outputs:
			map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("java_binary", "java", "Foo1", Attrs{"srcs": []string{"Foo.java"}}): {"com.Bar": {"//java:Bar"}},
			},
			nil,
		},
		{
			"Filters applied to candidate dependencies. E.g., we don't return filegroup(), avoid_dep or non-visible rules.",
			"java/Foo.java",
			[]ClassName{"com.Bar"},
			map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg([]*bazel.Rule{bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}})}),
			},
			[]Resolver{
				&testResolver{
					[]ClassName{"com.Bar"},
					map[ClassName][]*bazel.Rule{
						"com.Bar": {
							bazel.NewRule("java_library", "x", "Option1", publicAttr),
							bazel.NewRule("filegroup", "x", "Option2", publicAttr),
							bazel.NewRule("java_library", "x", "Option3", Attrs{"visibility": []string{"//visibility:public"}, "tags": []string{"bla", "avoid_dep"}}),
							bazel.NewRule("java_library", "x", "Option4", Attrs{"visibility": []string{"//visibility:private"}}),
						},
					},
				},
			},
			// Outputs:
			map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}}): {"com.Bar": {"//x:Option1"}},
			},
			nil,
		},
		{
			desc: "MissingDeps ranks the dependencies before returning them. " +
				"In this test, the ranker is a simple lexicographic sort." +
				"Therefore, //p2:dep2 appears after //p1:dep2 in the result, even though it was returned first from the resolver.",
			fileName:   "java/Foo.java",
			classNames: []ClassName{"com.Bar"},
			existingPkgs: map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg(
					[]*bazel.Rule{
						bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}}),
					},
				),
			},
			resolvers: []Resolver{
				&testResolver{
					[]ClassName{"com.Bar"},
					map[ClassName][]*bazel.Rule{
						"com.Bar": {
							bazel.NewRule("java_library", "p2", "dep2", publicAttr),
							bazel.NewRule("java_library", "p1", "dep1", publicAttr),
						},
					},
				},
			},
			// Outputs:
			wantMissing: map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}}): {"com.Bar": {"//p1:dep1", "//p2:dep2"}},
			},
		},
		{
			desc: `Jadep first filters by intrinsic attributes such as 'tags', and then filters by visibility.
If no candidate is left after the visibility filtering, we return _all_ candidates that were left after the initial filtering.
The reason is that visibility often needs to be updated as part of adding a dependency, while filtering due to intrinsic attributes
usually means the wrong dep was used.`,
			fileName:   "java/Foo.java",
			classNames: []ClassName{"com.Bar"},
			existingPkgs: map[string]*bazel.Package{
				"java": pkgloaderfakes.Pkg(
					[]*bazel.Rule{
						bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}}),
					},
				),
			},
			resolvers: []Resolver{
				&testResolver{
					[]ClassName{"com.Bar"},
					map[ClassName][]*bazel.Rule{
						"com.Bar": {
							bazel.NewRule("java_library", "p3", "dep2", Attrs{"tags": []string{"bla", "avoid_dep"}, "visibility": []string{"//visibility:public"}}),
							bazel.NewRule("java_library", "p2", "dep2", nil),
							bazel.NewRule("java_library", "p1", "dep1", nil),
						},
					},
				},
			},
			// Outputs:
			wantMissing: map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("java_library", "java", "Foo", Attrs{"srcs": []string{"Foo.java"}}): {"com.Bar": {"//p1:dep1", "//p2:dep2"}},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			var pkgNames []string
			for p := range test.existingPkgs {
				pkgNames = append(pkgNames, p)
			}
			cleanup := createBuildFileDir(t, pkgNames, workDir)
			defer cleanup()
			config := Config{
				WorkspaceDir: workDir,
				Loader:       &testLoader{test.existingPkgs},
				Resolvers:    test.resolvers,
				DepsRanker:   &sortingdepsranker.Ranker{},
			}
			rulesToFix, err := RulesConsumingFile(context.Background(), config, test.fileName)
			if err != nil {
				t.Errorf("Error getting rules to fix:\n%v", err)
			}
			missingDepsMap, unresClasses, err := MissingDeps(context.Background(), config, rulesToFix, test.classNames)
			if err != nil {
				t.Errorf("MissingDeps failed: %v.", err)
				return
			}
			if diff := cmp.Diff(missingDepsMap, test.wantMissing, sortRuleKeys); diff != "" {
				t.Errorf("MissingDeps returned diff in missing dependencies:\n%s", diff)
			}
			if !reflect.DeepEqual(unresClasses, test.wantUnresolved) {
				t.Errorf("MissingDeps returned unresolved classnames %s, want %s", unresClasses, test.wantUnresolved)
			}
		})
	}
}

func TestUnfilteredMissingDeps(t *testing.T) {
	type Attrs = map[string]interface{}

	var tests = []struct {
		desc             string
		classNames       []ClassName
		resolvers        []Resolver
		wantClassToRules map[ClassName][]bazel.Label
		wantUnresolved   []ClassName
	}{
		{
			desc: "UnfilteredMissingDeps ranks the dependencies before returning them. " +
				"In this test, the ranker is a simple lexicographic sort." +
				"Therefore, //p2:dep2 appears after //p1:dep2 in the result, even though it was returned first from the resolver.",
			classNames: []ClassName{"com.Bar"},
			resolvers: []Resolver{
				&testResolver{
					[]ClassName{"com.Bar"},
					map[ClassName][]*bazel.Rule{
						"com.Bar": {
							bazel.NewRule("java_library", "p2", "dep2", publicAttr),
							bazel.NewRule("java_library", "p1", "dep1", publicAttr),
						},
					},
				},
			},
			wantClassToRules: map[ClassName][]bazel.Label{
				"com.Bar": {"//p1:dep1", "//p2:dep2"},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			config := Config{
				Resolvers:  test.resolvers,
				DepsRanker: &sortingdepsranker.Ranker{},
			}
			classToRules, unresClasses := UnfilteredMissingDeps(context.Background(), config, test.classNames)
			if !reflect.DeepEqual(classToRules, test.wantClassToRules) {
				t.Errorf("UnfilteredMissingDeps returned missing dependencies %s, want %s", classToRules, test.wantClassToRules)
			}
			if !reflect.DeepEqual(unresClasses, test.wantUnresolved) {
				t.Errorf("UnfilteredMissingDeps returned unresolved classnames %s, want %s", unresClasses, test.wantUnresolved)
			}
		})
	}
}

func createWorkspace(t *testing.T) string {
	root, err := ioutil.TempDir("", "jadep")
	if err != nil {
		t.Fatalf("Error called ioutil.TempDir: %v", err)
	}
	workDir := filepath.Join(root, "google3")
	if err := os.MkdirAll(filepath.Join(workDir, "tools/build_rules"), 0700); err != nil {
		t.Errorf("Error called MkdirAll: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(workDir, "tools/build_rules/BUILD"), nil, 0666); err != nil {
		t.Errorf("Error called WriteFile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(workDir, "tools/build_rules/prelude_-redacted-"), []byte("# must be non-empty"), 0666); err != nil {
		t.Errorf("Error called WriteFile: %v", err)
	}
	return workDir
}

func TestExcludeClassNames(t *testing.T) {
	var tests = []struct {
		desc       string
		blackList  []string
		classNames []ClassName
		want       []ClassName
	}{
		{
			desc:       "The class name util.R should be excluded leaving an empty string at output.",
			blackList:  []string{`.*\.R$`},
			classNames: []ClassName{"util.R"},
			want:       nil,
		},
	}
	for _, test := range tests {
		actual := ExcludeClassNames(test.blackList, test.classNames)
		if !reflect.DeepEqual(actual, test.want) {
			t.Errorf("%s: ExcludedClassNames(%s, %s) = %s, want %s", test.desc, test.blackList, test.classNames, actual, test.want)
		}
	}
}

func TestGetKindForNewRule(t *testing.T) {
	var tests = []struct {
		desc         string
		filename     string
		classnames   []ClassName
		wantRuleType string
	}{
		{
			"When a file ends with Test.java but doesn't reference JUnit, we return java_library",
			"java/com/FooTest.java",
			[]ClassName{"com.Foo"},
			"java_library",
		},
		{
			"When a file does not end with Test.java but references JUnit, we return java_library",
			"java/com/Foo.java",
			[]ClassName{"org.junit.Test"},
			"java_library",
		},
		{
			"When a file does not end with Test.java and doesn't reference JUnit, we return java_library",
			"java/com/Foo.java",
			[]ClassName{},
			"java_library",
		},
		{
			"When a file ends with Test.java and references JUnit, we return java_test",
			"java/com/FooTest.java",
			[]ClassName{"org.junit.Test"},
			"java_test",
		},
	}
	for _, test := range tests {
		returnedRuleType := GetKindForNewRule(test.filename, test.classnames)
		if returnedRuleType != test.wantRuleType {
			t.Errorf("%s: FindPackageName(%s, %s) = %s, want %s", test.desc, test.filename, test.classnames, returnedRuleType, test.wantRuleType)
		}
	}
}

type testLoader struct {
	pkgs map[string]*bazel.Package
}

func (l *testLoader) Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error) {
	result := make(map[string]*bazel.Package)
	for _, pkgName := range packages {
		if p, ok := l.pkgs[pkgName]; ok {
			result[pkgName] = p
		}
	}
	return result, nil
}

func TestImplicitImports(t *testing.T) {
	in := map[ClassName][]bazel.Label{
		"java.lang.reflect.Method":   nil,
		"java.lang.Object":           nil,
		"java.lang.String":           nil,
		"java.util.Map":              nil,
		"javax.annotation.Generated": nil,
	}
	want := []string{"Object", "String"}

	lines := future.NewValue(func() interface{} {
		return in
	})
	got := ImplicitImports(lines).Get()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Diff in ImplicitImports(%v) (-got +want):\n%s", in, diff)
	}
}

func TestRulesConsumingFile(t *testing.T) {
	tests := []struct {
		desc         string
		fileName     string
		existingPkgs map[string]*bazel.Package
		want         []*bazel.Rule
		wantErr      error
	}{
		{
			desc:     "basic: only one rule consumes the file",
			fileName: "x/Foo.java",
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"x": bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
			},
		},
		{
			desc:     "multiple rules consume the file",
			fileName: "x/Foo.java",
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"x":     bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
						"x-gwt": bazel.NewRule("java_library", "x", "x-gwt", map[string]interface{}{"srcs": []string{"Foo.java"}}),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"Foo.java"}}),
				bazel.NewRule("java_library", "x", "x-gwt", map[string]interface{}{"srcs": []string{"Foo.java"}}),
			},
		},
		{
			desc:     "Rule consumes a file in a subdirectory",
			fileName: "x/subdir/Foo.java",
			existingPkgs: map[string]*bazel.Package{
				"x": {
					Rules: map[string]*bazel.Rule{
						"x": bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"subdir/Foo.java"}}),
					},
				},
			},
			want: []*bazel.Rule{
				bazel.NewRule("java_library", "x", "x", map[string]interface{}{"srcs": []string{"subdir/Foo.java"}}),
			},
		},
	}

	workDir := createWorkspace(t)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			var pkgNames []string
			for p := range tt.existingPkgs {
				pkgNames = append(pkgNames, p)
			}
			cleanup := createBuildFileDir(t, pkgNames, workDir)
			defer cleanup()
			config := Config{WorkspaceDir: workDir, Loader: &testLoader{tt.existingPkgs}}
			got, err := RulesConsumingFile(context.Background(), config, tt.fileName)
			if diff := cmp.Diff(tt.wantErr, err, equateErrorMessage); diff != "" {
				t.Errorf("RulesConsumingFile returned diff in error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("RulesConsumingFile returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateRule(t *testing.T) {
	type Attrs = map[string]interface{}
	tests := []struct {
		desc            string
		fileName        string
		namingRules     []NamingRule
		defaultRuleKind string
		want            *bazel.Rule
	}{
		{
			desc:     "File name matches a name matcher.",
			fileName: "javatests/com/FooTest.java",
			namingRules: []NamingRule{
				{FileNameMatcher: regexp.MustCompile("javatests/.*?/(.*Test).java"), RuleKind: "java_test"},
			},
			defaultRuleKind: "java_library",
			want:            bazel.NewRule("java_test", "javatests/com", "FooTest", Attrs{"srcs": []string{"FooTest.java"}}),
		},
		{
			desc:     "File name doesn't match any matchers - use the default kind.",
			fileName: "java/com/Foo.java",
			namingRules: []NamingRule{
				{FileNameMatcher: regexp.MustCompile("javatests/.*?/(.*Test).java"), RuleKind: "java_test"},
			},
			defaultRuleKind: "java_library",
			want:            bazel.NewRule("java_library", "java/com", "Foo", Attrs{"srcs": []string{"Foo.java"}}),
		},
		{
			desc:            "Handles files in the workspace root correctly.",
			fileName:        "Foo.java",
			defaultRuleKind: "java_library",
			want:            bazel.NewRule("java_library", "", "Foo", Attrs{"srcs": []string{"Foo.java"}}),
		},
		{
			desc:     "Several rules match - pick the first one.",
			fileName: "javatests/android/FooTest.java",
			namingRules: []NamingRule{
				{FileNameMatcher: regexp.MustCompile("javatests/android/(.+).java"), RuleKind: "android_test"},
				{FileNameMatcher: regexp.MustCompile("javatests/(.+).java"), RuleKind: "java_test"},
			},
			defaultRuleKind: "java_library",
			want:            bazel.NewRule("android_test", "javatests/android", "FooTest", Attrs{"srcs": []string{"FooTest.java"}}),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			got := CreateRule(tt.fileName, tt.namingRules, tt.defaultRuleKind)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("CreateRule returned wrong rule: (-got +want).\n%s", diff)
			}
		})
	}
}
