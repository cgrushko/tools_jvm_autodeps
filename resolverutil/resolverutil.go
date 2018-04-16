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

// Package resolverutil provides functions that help implement resolvers.
package resolverutil

import (
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
)

// SatisfiedByExistingDeps finds class names for which a satisfying dependency appears in the consuming rules.
// In a sense, it returns the intersection of consumingRules and satisfyingRules.
// It is used as an optimization to return faster when we know the consuming rules already have a dependency for certain class names.
// For each class name, it checks whether every consuming rule has at least one dependency that this class name can be satisfied with.
// It returns class names for which this holds, with the list of satisfying dependencies that covers all consuming rules.
func SatisfiedByExistingDeps(consumingRules map[bazel.Label]map[bazel.Label]bool, satisfyingRules map[jadeplib.ClassName][]bazel.Label) map[jadeplib.ClassName][]bazel.Label {
	if len(consumingRules) == 0 {
		return nil
	}

	alreadySatisfied := make(map[jadeplib.ClassName][]bazel.Label)

classLoop:
	for cls, possibleDeps := range satisfyingRules {
		var existingSatisfyingDeps []bazel.Label
		for consumingRule, existingDeps := range consumingRules {
			satisfied := false
			for _, d := range possibleDeps {
				if consumingRule == d || existingDeps[d] {
					existingSatisfyingDeps = append(existingSatisfyingDeps, d)
					satisfied = true
					break
				}
			}
			if !satisfied {
				continue classLoop
			}
		}
		alreadySatisfied[cls] = existingSatisfyingDeps
	}

	return alreadySatisfied
}
