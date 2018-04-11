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

// Package ruleconsts defines constants related to names and kinds of Java rules.
package ruleconsts

import (
	"regexp"

	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
)

var (
	// NewRuleNamingRules specifies how to name, and which kind, should new rules have, based on the file names they srcs.
	// See jadeplib.NamingRule for requirements for the regexps.
	NewRuleNamingRules = []jadeplib.NamingRule{
		{FileNameMatcher: androidTestRegexp, RuleKind: "android_test"},
		{FileNameMatcher: androidLibraryRegexp, RuleKind: "android_library"},
		{FileNameMatcher: javaTestRegexp, RuleKind: "java_test"},
	}

	androidTestRegexp    = regexp.MustCompile(`^javatests/com/google/android/(.*/)?.*Test\.java$`)
	androidLibraryRegexp = regexp.MustCompile(`^java(tests)?/com/google/android/(.*/)?.+\.java$`)
	javaTestRegexp       = regexp.MustCompile(`^javatests/(.*/)?.*Test\.java$`)
)

// DefaultNewRuleKind is used to create new rules when NewRuleNamingRules can't be used.
const DefaultNewRuleKind = "java_library"
