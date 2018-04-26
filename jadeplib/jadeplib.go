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

// Package jadeplib finds a list of BUILD labels that provide the requested Java class names.
package jadeplib

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/compat"
	"github.com/bazelbuild/tools_jvm_autodeps/filter"
	"github.com/bazelbuild/tools_jvm_autodeps/future"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
	"github.com/bazelbuild/tools_jvm_autodeps/vlog"
)

// Config specifies the content roots and workspace root.
// The WorkspaceDir defines the users workspace.
type Config struct {
	// WorkspaceDir is a path to the root of a Bazel workspace.
	WorkspaceDir string

	// Loader loads BUILD files.
	Loader pkgloading.Loader

	Resolvers []Resolver

	DepsRanker DepsRanker
}

// Resolver defines methods to resolve class names to Bazel rules.
type Resolver interface {
	Name() string

	// consumingRules specifies the dependencies of each rule whose srcs include the file currently being processed.
	// Resolvers may use this information to short-circuit computations.
	Resolve(ctx context.Context, classNames []ClassName, consumingRules map[bazel.Label]map[bazel.Label]bool) (map[ClassName][]*bazel.Rule, error)
}

// DepsRanker defines methods to rank dependencies so it's easier for users to choose the right option.
type DepsRanker interface {
	// Less is used in a call to sort.Slice() to rank dependencies before asking a user to choose one.
	// Less should position the dependency a user is most likely to choose, first.
	// In other words, the label that should appear first should satisfy Less(ctx, label, x) == true for all x.
	Less(ctx context.Context, label1, label2 bazel.Label) bool
}

// ClassName is a class name, e.g. com.google.Foo.
type ClassName string

// MissingDeps returns Labels that can be used to satisfy missing dependencies. For example,
// let F.java be the Java file the user is processing, and {F1, F2, ..., Fn}
// be the rules that have F.java in their srcs. Then MissingDeps returns for each Fi,
// the set of missing dependencies. A missing dependency is reported as a map
// ClassName -> []bazel.Label, which details which classnames can be satisfied by which dependencies.
// It also returns a list of classnames that were unable to be resolved.
func MissingDeps(ctx context.Context, config Config, rulesToFix []*bazel.Rule, classNames []ClassName) (map[*bazel.Rule]map[ClassName][]bazel.Label, []ClassName, error) {
	depsOfRuleToFix := make(map[bazel.Label]map[bazel.Label]bool)
	for _, r := range rulesToFix {
		depsOfRuleToFix[r.Label()] = deps(r)
	}

	resolved, unresClassNames, _ := resolveAll(ctx, config.Resolvers, classNames, depsOfRuleToFix)

	// Initially filter 'resolved' according to tags, rule type, etc.
	// These do not require loading BUILD packages.
	ctx, endSpan := compat.NewLocalSpan(ctx, "Jade: MissingDeps construct result")
	filteredCandidates := make(map[*bazel.Rule]map[ClassName][]*bazel.Rule)
	visQuery := make(map[filter.VisQuery]bool)
	for _, consumingRule := range rulesToFix {
		lbl := consumingRule.Label()
		candidatesForConsRule := make(map[ClassName][]*bazel.Rule)
		for class, satisfyingRules := range resolved {
			if alreadySatisfied(lbl, depsOfRuleToFix[lbl], satisfyingRules) {
				continue
			}
			for _, satRule := range satisfyingRules {
				if filter.IsValidDependency(satRule) {
					candidatesForConsRule[class] = append(candidatesForConsRule[class], satRule)
					visQuery[filter.VisQuery{Rule: satRule, Pkg: consumingRule.PkgName}] = true
				}
			}
		}
		filteredCandidates[consumingRule] = candidatesForConsRule
	}

	// Further filter filteredCandidates according to visiblity and fill out missingRuleDeps for returning.
	visResult, err := filter.CheckVisibility(ctx, config.Loader, visQuery)
	if err != nil {
		return nil, nil, err
	}
	missingRuleDeps := make(map[*bazel.Rule]map[ClassName][]bazel.Label)
	for consRule, classToSatisfiers := range filteredCandidates {
		consPkgName := consRule.PkgName
		missingForConsRule := make(map[ClassName][]bazel.Label)
		for cls, satisfyingRules := range classToSatisfiers {
			var visible []bazel.Label
			for _, satRule := range satisfyingRules {
				if visResult[filter.VisQuery{Rule: satRule, Pkg: consPkgName}] {
					visible = append(visible, satRule.Label())
				} else {
					vlog.V(2).Printf("Filtered because of visibility: %q is not visible to %q for class %q", satRule.Label(), consRule.Label(), cls)
				}
			}
			if len(visible) == 0 {
				log.Printf("No rules left for class %q after visibility filtering; returning all results.", cls)
				for _, satRule := range satisfyingRules {
					visible = append(visible, satRule.Label())
				}
			}
			missingForConsRule[cls] = visible
		}
		if len(missingForConsRule) > 0 {
			missingRuleDeps[consRule] = missingForConsRule
		}
	}

	sortDependencies(ctx, config.DepsRanker, missingRuleDeps)
	endSpan()

	return missingRuleDeps, unresClassNames, nil
}

// UnfilteredMissingDeps returns Labels that can be used to satisfy missing dependencies.
// Unlike MissingDeps, this function doesn't filter the results according to rule kind, visiblity, tag, etc.
// The results are ranked according to config.DepsRanker.
// It also returns a list of classnames that were unable to be resolved.
func UnfilteredMissingDeps(ctx context.Context, config Config, classNames []ClassName) (resolved map[ClassName][]bazel.Label, unresolved []ClassName) {
	resolvedAsRules, unresolved, _ := resolveAll(ctx, config.Resolvers, classNames, nil)
	resolved = make(map[ClassName][]bazel.Label)
	for cls, rules := range resolvedAsRules {
		var labels []bazel.Label
		for _, r := range rules {
			labels = append(labels, r.Label())
		}
		sort.Slice(labels, func(i, j int) bool { return config.DepsRanker.Less(ctx, labels[i], labels[j]) })
		resolved[cls] = labels
	}
	return resolved, unresolved
}

// RulesConsumingFile returns the set of Java rules whose 'srcs' attribute contains 'fileName'.
// fileName must be a path relative to config.WorkspaceDir.
func RulesConsumingFile(ctx context.Context, config Config, fileName string) ([]*bazel.Rule, error) {
	pkgs, _, err := pkgloading.Siblings(ctx, config.Loader, config.WorkspaceDir, []string{fileName})
	if err != nil {
		return nil, err
	}

	var ret []*bazel.Rule
	for consPkgName, consPkg := range pkgs {
		relativeFileName, err := filepath.Rel(consPkgName, fileName)
		if err != nil {
			return nil, err
		}
		for _, consRule := range consPkg.Rules {
			if filter.JavaEditableRuleKinds[consRule.Schema] && srcsFile(consRule, relativeFileName) {
				ret = append(ret, consRule)
			}
		}
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].Label() < ret[j].Label() })
	return ret, nil
}

// deps returns a set containing the 'deps' attribute of 'rule' in Label form.
func deps(rule *bazel.Rule) map[bazel.Label]bool {
	ret := make(map[bazel.Label]bool)
	for _, d := range rule.StringListAttr("deps") {
		if l, err := bazel.ParseRelativeLabel(rule.PkgName, d); err == nil {
			ret[l] = true
		}
	}
	return ret
}

// resolveAll calls all resolvers sequentially, feeding the unresolved classes from resolver[i-1] into resolver[i].
// Returns a map of resolved classnames -> rules, and a list of unresolved classes.
func resolveAll(ctx context.Context, resolvers []Resolver, classNames []ClassName, depsOfRuleToFix map[bazel.Label]map[bazel.Label]bool) (map[ClassName][]*bazel.Rule, []ClassName, map[Resolver]error) {
	resultResolved := make(map[ClassName][]*bazel.Rule)
	resultUnresolved := make(map[ClassName]bool)
	resultErrors := make(map[Resolver]error)
	for _, c := range classNames {
		resultUnresolved[c] = true
	}
	for _, res := range resolvers {
		if len(resultUnresolved) == 0 {
			break
		}
		var classNames []ClassName
		for cls := range resultUnresolved {
			classNames = append(classNames, cls)
		}

		tctx, endSpan := compat.NewLocalSpan(ctx, "Jade: Resolve ("+res.Name())
		stopwatch := time.Now()
		resolved, err := res.Resolve(tctx, classNames, depsOfRuleToFix)
		log.Printf("Resolved %4d/%-4d classes using %20s (%dms)", len(resolved), len(classNames), res.Name(), int64(time.Now().Sub(stopwatch)/time.Millisecond))
		endSpan()

		if err != nil {
			log.Printf("Error when resolving using %s: %v", res.Name(), err)
			resultErrors[res] = err
		}
		for cls, rules := range resolved {
			for _, r := range rules {
				resultResolved[cls] = append(resultResolved[cls], r)
			}
			delete(resultUnresolved, cls)
		}
	}

	var unresolvedSlice []ClassName
	for cls := range resultUnresolved {
		unresolvedSlice = append(unresolvedSlice, cls)
	}
	sort.Slice(unresolvedSlice, func(i, j int) bool { return string(unresolvedSlice[i]) < string(unresolvedSlice[j]) })
	return resultResolved, unresolvedSlice, resultErrors
}

// sortDependencies sorts the options in missingRuleDeps according to 'ranker'.
// It mutates missingRulesDeps.
func sortDependencies(ctx context.Context, ranker DepsRanker, missingRuleDeps map[*bazel.Rule]map[ClassName][]bazel.Label) {
	stopwatch := time.Now()
	for _, classToLabels := range missingRuleDeps {
		for _, labels := range classToLabels {
			sort.Slice(labels, func(i, j int) bool { return ranker.Less(ctx, labels[i], labels[j]) })
		}
	}
	log.Printf("Ranking dependencies (%dms)", int64(time.Now().Sub(stopwatch)/time.Millisecond))
}

// ExcludeClassNames filters class names based on blacklisted regular expressions from the user.
func ExcludeClassNames(blacklistRegexps []string, classNames []ClassName) []ClassName {
	var newClassNames []ClassName
	for _, classname := range classNames {
		found := false
		for _, regexs := range blacklistRegexps {
			match, err := regexp.MatchString(regexs, string(classname))
			if err != nil {
				fmt.Printf("Error %v occurred during matching", err)
				continue
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			newClassNames = append(newClassNames, classname)
		}
	}
	return newClassNames
}

// GetKindForNewRule determines if a rule that srcs a  filename is a java_library rule
// or a java_test rule.
func GetKindForNewRule(filename string, classNames []ClassName) string {
	if !strings.HasSuffix(filename, "Test.java") {
		return "java_library"
	}
	for _, name := range classNames {
		if name == "org.junit.Test" {
			return "java_test"
		}
	}
	return "java_library"
}

// alreadySatisfied decides whether a class is already satisfied by the existing 'deps' of the consuming rule.
func alreadySatisfied(consumingRuleLabel bazel.Label, existingDeps map[bazel.Label]bool, satisfyingRules []*bazel.Rule) bool {
	for _, r := range satisfyingRules {
		if r.Label() == consumingRuleLabel {
			return true
		}
		if _, ok := existingDeps[r.Label()]; ok {
			return true
		}
	}
	return false
}

// srcsFile returns true if a rule has relativeFileName in its 'srcs' attribute.
// For example, only rules that source the file the user asked about should be edited.
func srcsFile(rule *bazel.Rule, relativeFileName string) bool {
	for _, src := range rule.StringListAttr("srcs") {
		if relativeFileName == src {
			return true
		}
	}
	return false
}

// ImplicitImports returns the set of simple names that Java programs can use without importing, e.g. String, Object, Integer, etc.
// 'dict' is a future to a map[ClassName][]bazel.Label whose keys are built-in fully-qualified class names.
// Returns a sorted slice if the input is a sorted slice.
func ImplicitImports(dict *future.Value) *future.Value {
	return future.NewValue(func() interface{} {
		var ret []string
		for cls := range dict.Get().(map[ClassName][]bazel.Label) {
			s := string(cls)
			if strings.HasPrefix(s, "java.lang.") {
				simple := s[len("java.lang."):]
				if !strings.ContainsRune(simple, '.') {
					ret = append(ret, simple)
				}
			}
		}
		sort.Strings(ret)
		return ret
	})
}

// NamingRule is used by NewRule to create new Bazel rules.
type NamingRule struct {
	// FileNameMatcher matches file names for which we should create a new rule of kind RuleKind.
	FileNameMatcher *regexp.Regexp

	// RuleKind is the kind of new rule that NewRule creates, e.g. "java_library".
	RuleKind string
}

// CreateRule creates a new rule with srcs = [filename].
// The kind of the new rule is determined by matching fileName against namingRules's FileNameMatcher, in sequence.
// If no naming rule matches, CreateRule creates a new rule of kind 'defaultRuleKind'.
// The name of the new rule is the file name (without extension).
// fileName is a file name relative to the workspace root (e.g., should be 'java/com/Foo.java', not 'Foo.java').
func CreateRule(fileName string, namingRules []NamingRule, defaultRuleKind string) *bazel.Rule {
	kind := defaultRuleKind
	for _, r := range namingRules {
		m := r.FileNameMatcher.FindStringSubmatch(fileName)
		if m != nil {
			kind = r.RuleKind
			break
		}
	}
	pkgName := filepath.Dir(fileName)
	if pkgName == "." {
		pkgName = ""
	}
	src := filepath.Base(fileName)
	name := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	return bazel.NewRule(kind, pkgName, name, map[string]interface{}{"srcs": []string{src}})
}
