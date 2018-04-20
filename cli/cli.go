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

// Package cli provides utilities for writing languages-specific main packages, such as Jade.
package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/buildozer"
	"github.com/bazelbuild/tools_jvm_autodeps/future"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/lang/java/parser"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
	"github.com/bazelbuild/tools_jvm_autodeps/vlog"
)

// FilesToParse returns the list of files to parse based on 'arg'.
// If arg is a label, FilesToParse loads the rule and returns the files referenced in its "srcs" attribute.
// Otherwise, 'arg' is assumed to be a file name which is returned in absolute form.
// 'arg' is treated relative to 'workingDir', which is not necessarily $pwd in case the user provided an explicit -workapce flag.
func FilesToParse(arg, workingDir string, loader pkgloading.Loader) ([]string, error) {
	label, err := bazel.ParseAbsoluteLabel(arg)
	if err != nil {
		// arg is not a label, must be a file.
		if filepath.IsAbs(arg) {
			return []string{arg}, nil
		}
		return []string{filepath.Join(workingDir, arg)}, nil
	}

	rules, _, err := pkgloading.LoadRules(context.Background(), loader, []bazel.Label{label})
	if err != nil {
		return nil, fmt.Errorf("Error while loading %v:\n%v", label, err)
	}
	rule := rules[label]
	if rule == nil {
		return nil, fmt.Errorf("Rule not found: %v", label)
	}
	var ret []string
	for _, s := range rule.StringListAttr("srcs") {
		lbl, err := bazel.ParseRelativeLabel(rule.PkgName, s)
		if err != nil {
			log.Printf("Illegal label %q in srcs attribute, skipping.", lbl)
		}
		p, n := lbl.Split()
		ret = append(ret, filepath.Join(p, n))
	}
	return ret, nil
}

// RulesToFix returns the set of rules whose 'deps' Jade should manipulate, based on 'arg'.
// If 'arg' is a label, it will be loaded and returned.
// Otherwise, 'arg' is assumed to be a file name, and RulesToFix will load its containig package and return any Java rule that 'srcs' it.
// In this case, 'arg' is treated relative to 'relWorkingDir', which is the working directory relative to the workspace root.
// For a description of namingRules and defaultRuleKind, see jadeplib.CreateRule.
func RulesToFix(ctx context.Context, config jadeplib.Config, relWorkingDir, arg string, namingRules []jadeplib.NamingRule, defaultRuleKind string) ([]*bazel.Rule, error) {
	label, err := bazel.ParseAbsoluteLabel(arg)
	if err == nil {
		rules, _, err := pkgloading.LoadRules(ctx, config.Loader, []bazel.Label{label})
		if err != nil {
			return nil, fmt.Errorf("Error loading %q:\n%v", label, err)
		}

		r := rules[label]
		if r == nil {
			return nil, fmt.Errorf("Rule not found: %v", label)
		}
		return []*bazel.Rule{r}, nil
	}

	var fileName string
	// Make arg relative to the workspace root.
	if filepath.IsAbs(arg) {
		relarg, err := filepath.Rel(config.WorkspaceDir, arg)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relarg, "..") {
			return nil, fmt.Errorf("%q is not a relative path nor in a subdirectory of %q", arg, config.WorkspaceDir)
		}
		fileName = relarg
	} else {
		fileName = filepath.Join(relWorkingDir, arg)
	}

	ret, err := jadeplib.RulesConsumingFile(ctx, config, fileName)
	if err != nil {
		return nil, fmt.Errorf("Error from finding rules to fix from %q:\n%v", arg, err)
	}
	if len(ret) > 0 {
		return ret, nil
	}

	// No rules consumes file name - create one,
	newRule := jadeplib.CreateRule(fileName, namingRules, defaultRuleKind)
	err = buildozer.NewRule(config.WorkspaceDir, newRule)
	if err != nil {
		return nil, err
	}
	return []*bazel.Rule{newRule}, nil
}

// LogRulesToFix prints 'rules'.
// It is used to announce which rules we're about to fix.
func LogRulesToFix(rules []*bazel.Rule) {
	if len(rules) == 0 {
		return
	}
	var strs []string
	for _, r := range rules {
		strs = append(strs, string(r.Label()))
	}
	log.Printf("Fixing: %s", strings.Join(strs, ", "))
}

// Workspace returns the directory path of the workspace in which Jade should operate (workspaceDir) and the working dir relative to it (relWorkingDir).
// workspaceFlag is what the user specified on the command-line.
// If workspaceFlag is empty, Workspace() searches for a directory that contains a WORKSPACE file starting at the working directory and moving upwards.
// relWorkingDir will be empty if workspaceFlag isn't empty (= -workspace was specifief explicitly)
func Workspace(workspaceFlag string) (workspaceDir, relWorkingDir string, err error) {
	if workspaceFlag != "" {
		result, err := filepath.Abs(workspaceFlag)
		if err != nil {
			return "", "", fmt.Errorf("couldn't make directory absolute: %v", err)
		}
		if !hasWORKSPACE(result) {
			return "", "", fmt.Errorf("directory %v has no file named WORKSPACE", result)
		}
		return result, "", nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("couldn't get working directory: %v", err)
	}
	for result := wd; result != string(filepath.Separator); result = filepath.Dir(result) {
		if hasWORKSPACE(result) {
			relWorkingDir, err = filepath.Rel(result, wd)
			if err != nil {
				return "", "", fmt.Errorf("couldn't relative %s to %s", result, wd)
			}
			if relWorkingDir == "." {
				relWorkingDir = ""
			}
			return result, relWorkingDir, nil
		}
	}
	return "", "", fmt.Errorf("couldn't find a parent of %v that has a WORKSPACE file", wd)
}

func hasWORKSPACE(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "WORKSPACE"))
	return !os.IsNotExist(err)
}

// StartProfiler starts CPU profiling and writes the output to outFile.
func StartProfiler(outFile string) (stopProfiler func()) {
	if outFile == "" {
		return func() {}
	}
	f, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	return pprof.StopCPUProfile
}

// ReportMissingDeps logs the dependencies that Jadep detected as missing.
func ReportMissingDeps(missingDeps map[*bazel.Rule]map[jadeplib.ClassName][]bazel.Label) {
	anythingMissing := false
	for editedRule, classToRule := range missingDeps {
		log.Printf("Missing dependencies in %s", editedRule.Label())
		for cls, lbls := range classToRule {
			var lblsStr []string
			for _, l := range lbls {
				lblsStr = append(lblsStr, string(l))
			}
			log.Printf("%-50s can be satisfied using:", cls)
			log.Printf("             %s", strings.Join(lblsStr, ", "))
			anythingMissing = true
		}
	}
	if !anythingMissing {
		log.Println("Nothing to do.")
	}
}

// ReportUnresolvedClassnames logs the class names that Jadep couldn't find any BUILD dependencies for.
func ReportUnresolvedClassnames(unresolvedClassNames []jadeplib.ClassName) {
	if len(unresolvedClassNames) == 0 {
		return
	}
	log.Printf("Class names we don't know how to satisfy:")
	for _, cls := range unresolvedClassNames {
		log.Println(cls)
	}
}

// ClassNamesToResolve returns the list of class names which should be satisfied with BUILD dependencies.
// If the user provided a list in --classnames (which is passed in classNamesArg), that list is returned.
// Otherwise, it parses Java files as described in FilesToParse().
// blacklist is a list of regular expressions matching names of classes for which we will not look for BUILD rules.
// See FilesToParse for explanation about 'workingDir' and 'arg'.
func ClassNamesToResolve(ctx context.Context, workingDir string, loader pkgloading.Loader, arg string, classNamesArg []string, implicitImports *future.Value, blacklist []string) []jadeplib.ClassName {
	if len(classNamesArg) > 0 {
		var ret []jadeplib.ClassName
		for _, c := range classNamesArg {
			parts := strings.Split(c, ".")
			if topLevel, idx := parser.ExtractClassNameFromQualifiedName(parts); idx != -1 {
				ret = append(ret, jadeplib.ClassName(topLevel))
			} else {
				ret = append(ret, jadeplib.ClassName(c))
			}
		}
		return ret
	}

	filesToParse, err := FilesToParse(arg, workingDir, loader)
	if err != nil {
		log.Fatal(err)
	}
	stopwatch := time.Now()
	ret := jadeplib.ExcludeClassNames(blacklist, parser.ReferencedClasses(ctx, filesToParse, implicitImports.Get().([]string)))
	vlog.V(2).Printf("Class names to resolve:\n%v", ret)

	log.Printf("Found %d classes in %d Java file(s) (%dms)", len(ret), len(filesToParse), int64(time.Now().Sub(stopwatch)/time.Millisecond))
	return ret
}
