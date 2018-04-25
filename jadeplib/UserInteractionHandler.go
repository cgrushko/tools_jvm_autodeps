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
	"io"
	"strconv"

	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
)

// ask takes in a list of printable interfaces. It returns the
// input from the user indicating which interfaces is wanted.
// ask keeps asking the user for input until a valid input is given.
// If reading from stdin fails, returns an error.
func ask(in io.Reader, description string, options []bazel.Label) (int, error) {
	if len(options) == 1 {
		return 1, nil
	}
	fmt.Println()
	for i := len(options) - 1; i >= 0; i-- {
		fmt.Printf("[%v] %v\n", i+1, options[i])
	}
	fmt.Println("[0] None")

	fmt.Print(description)
	fmt.Printf("Enter a number to choose, or just Enter to select the default [%s]\n", options[0])
	for {
		var i string
		if _, err := fmt.Fscanln(in, &i); err != nil {
			if err == io.EOF {
				return -1, fmt.Errorf("Error reading stdin: %v", err)
			}
			return 1, nil
		}
		idx, err := strconv.Atoi(i)
		if err != nil {
			fmt.Println("Error occurred when converting input to integer. Please try again.")
			continue
		}
		if idx <= len(options) && idx >= 0 {
			return idx, nil
		}
		fmt.Println("Invalid index inputted. Please try again.")
	}
}

// SelectDepsToAdd asks the user to choose which deps to add to their rules to satisfy missing dependencies.
func SelectDepsToAdd(in io.Reader, missingDepsMap map[*bazel.Rule]map[ClassName][]bazel.Label) (map[*bazel.Rule][]bazel.Label, error) {
	depsToAdd := make(map[*bazel.Rule][]bazel.Label)
	for rule, classToRules := range missingDepsMap {
		addedDeps := make(map[bazel.Label]bool)
		for class, rules := range classToRules {
			if depAlreadySatisfied(addedDeps, rules) {
				continue
			}
			idx, err := ask(in, fmt.Sprintf("Choose a BUILD rule for %s to add to %s.\n", class, rule.Label()), rules)
			if err != nil {
				return nil, err
			}
			if idx != 0 {
				addedDeps[rules[idx-1]] = true
				depsToAdd[rule] = append(depsToAdd[rule], rules[(idx-1)])
			}
		}
	}
	return depsToAdd, nil
}

func depAlreadySatisfied(addedDeps map[bazel.Label]bool, rules []bazel.Label) bool {
	for _, rule := range rules {
		if _, ok := addedDeps[rule]; ok {
			return true
		}
	}
	return false
}
