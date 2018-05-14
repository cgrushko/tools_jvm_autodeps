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

// Package buildozer provides functions to work with Buildozer
package buildozer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/edit"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
)

// Ref returns a token that Buildozer can use to manipulate a function call.
// When a rule is instatiated directly (not through a Bazel macro), its reference equals its label.
// Otherwise, if the macro has a name attribute, the reference looks like a label whose name is the macro's name.
// Finally, if there's no name attribute, the reference is //<pkg>:<line> where line is where the macro starts in the BUILD file.
func Ref(rule *bazel.Rule) (string, error) {
	if _, ok := rule.Attrs["generator_function"]; !ok {
		// Not a macro
		return string(rule.Label()), nil
	}
	name, ok := rule.Attrs["generator_name"].(string)
	if !ok {
		loc, _ := rule.Attrs["generator_location"].(string)
		parts := strings.Split(loc, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("expected rule's generator_location (%q) to have exactly one colon", loc)
		}
		name = "%" + parts[1]
	}
	return "//" + rule.PkgName + ":" + name, nil
}

// NewRule uses Buildozer to create a new rule based on the attributes of 'rule'.
// Used attributes are Name, PkgName, Schema and srcs.
func NewRule(workspaceRoot string, rule *bazel.Rule) error {
	pkgName := rule.PkgName
	name := rule.Name()
	buildFile := filepath.Join(workspaceRoot, pkgName, "BUILD")
	if _, err := os.Stat(buildFile); os.IsNotExist(err) {
		if err := ioutil.WriteFile(buildFile, nil, 0666); err != nil {
			return fmt.Errorf("error writing %s:\n%v", buildFile, err)
		}
	}
	err := exec(workspaceRoot, []string{
		fmt.Sprintf("new %s %s", rule.Schema, name),
		fmt.Sprintf("//%s:__pkg__", pkgName),
	}, []int{0})
	if err != nil {
		return err
	}
	err = exec(workspaceRoot, []string{
		fmt.Sprintf("add srcs %s", strings.Join(rule.StringListAttr("srcs"), " ")),
		fmt.Sprintf("//%s:%s", pkgName, name),
	}, []int{0})
	if err != nil {
		return err
	}
	return nil
}

// AddDepsToRules on (rule -> labels) adds labels to rule.
func AddDepsToRules(workspaceRoot string, missingDeps map[*bazel.Rule][]bazel.Label) error {
	for rule, labels := range missingDeps {
		labelToModify, err := Ref(rule)
		if err != nil {
			return fmt.Errorf("error getting buildozer reference for %v:\n%v", rule, err)
		}
		var deps bytes.Buffer
		for _, l := range labels {
			deps.WriteString(string(l))
			deps.WriteString(" ")
		}
		err = exec(workspaceRoot, []string{fmt.Sprintf("add deps %s", deps.String()), labelToModify}, []int{0, 3})
		if err != nil {
			return err
		}
	}
	return nil
}

// exec calls Buildozer and returns an error if its exit code isn't one of allowedReturnedCodes.
// args is a single command line, e.g. ["add deps", "//foo:bar", "//target"].
func exec(workspaceRoot string, args []string, allowedReturnedCodes []int) error {
	opts := &edit.Options{
		NumIO:             200,
		KeepGoing:         true,
		PreferEOLComments: true,
		RootDir:           workspaceRoot,
		Quiet:             true,
	}
	retval := edit.Buildozer(opts, args)
	for _, allowed := range allowedReturnedCodes {
		if retval == allowed {
			return nil
		}
	}
	return fmt.Errorf("buildozer returned %d, want one of %v, while executing %v", retval, allowedReturnedCodes, args)
}
