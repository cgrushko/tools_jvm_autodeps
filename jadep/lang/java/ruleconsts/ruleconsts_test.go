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

package ruleconsts

import (
	"regexp"
	"testing"
)

func TestNewRuleNamingRules(t *testing.T) {
	tests := []struct {
		re   *regexp.Regexp
		s    string
		want bool
	}{
		// androidTestRegexp
		{
			re:   androidTestRegexp,
			s:    "javatests/com/google/android/FooTest.java",
			want: true,
		},
		{
			re:   androidTestRegexp,
			s:    "javatests/com/google/android/Test.java",
			want: true,
		},
		{
			re:   androidTestRegexp,
			s:    "javatests/com/google/android/blabla1/blabla2/FooTest.java",
			want: true,
		},
		{
			re:   androidTestRegexp,
			s:    "javatests/com/google/android/Foo.java",
			want: false,
		},
		{
			re:   androidTestRegexp,
			s:    "javatests/com/google/FooTest.java",
			want: false,
		},
		// androidLibraryRegexp
		{
			re:   androidLibraryRegexp,
			s:    "javatests/com/google/android/FooTest.java",
			want: true,
		},
		{
			re:   androidLibraryRegexp,
			s:    "javatests/com/google/android/blabla1/blabla2/FooTest.java",
			want: true,
		},
		{
			re:   androidLibraryRegexp,
			s:    "javatests/com/google/android/Foo.java",
			want: true,
		},
		{
			re:   androidLibraryRegexp,
			s:    "java/com/google/android/Foo.java",
			want: true,
		},
		{
			re:   androidLibraryRegexp,
			s:    "javatests/com/google/FooTest.java",
			want: false,
		},
		// javaTestRegexp
		{
			re:   javaTestRegexp,
			s:    "javatests/com/google/android/FooTest.java",
			want: true,
		},
		{
			re:   javaTestRegexp,
			s:    "javatests/com/google/Test.java",
			want: true,
		},
		{
			re:   javaTestRegexp,
			s:    "javatests/com/google/FooTest.java",
			want: true,
		},
	}

	for _, tt := range tests {
		m := tt.re.FindStringSubmatch(tt.s)
		if m != nil != tt.want {
			t.Errorf("Matching %v on %q = %v, want %v", tt.re, tt.s, m != nil, tt.want)
		}
	}
}
