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
	"bytes"
	"testing"

	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/google/go-cmp/cmp"
)

func TestUserInteractionHandler(t *testing.T) {
	var tests = []struct {
		input string
		rules []bazel.Label
		want  int
	}{
		{"5 \n", []bazel.Label{"", "", "", "", ""}, 5},
		{"\n", []bazel.Label{"", "", "", "", ""}, 1},
		{"1 \n", []bazel.Label{"", ""}, 1},
		{"1 1\n 1\n", []bazel.Label{""}, 1},
		{"e\n 2\n 1\n", []bazel.Label{""}, 1},
		{"", []bazel.Label{""}, 1},
	}
	for idx, test := range tests {
		in := bytes.NewReader([]byte(test.input))
		i, err := UserInteractionHandler(in, test.rules)
		if err != nil {
			t.Errorf("Test case %d returned unexpected error:\n%v", idx, err)
		}
		if i != test.want {
			t.Errorf("UserInteractionHandler returned index %v, want %v", i, test.want)
		}
	}
}

func TestUserInteractionHandlerNoStdin(t *testing.T) {
	in := bytes.NewReader(nil)
	_, err := UserInteractionHandler(in, []bazel.Label{"", ""})
	wantErr := "Error reading stdin: EOF"
	if err.Error() != wantErr {
		t.Errorf("Want error %q, got: %v", wantErr, err)
	}
}

func TestSelectDepsToAdd(t *testing.T) {
	var tests = []struct {
		desc           string
		missingDepsMap map[*bazel.Rule]map[ClassName][]bazel.Label
		input          string
		want           map[*bazel.Rule][]bazel.Label
	}{
		{
			desc: "Basic test to check if deps are added to the Map",
			missingDepsMap: map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("", "java/y", "Jade", nil): {"x.Foo": {"//java/x:Missing"}},
			},
			input: "1\n",
			want: map[*bazel.Rule][]bazel.Label{
				bazel.NewRule("", "java/y", "Jade", nil): {"//java/x:Missing"},
			},
		},
		{
			desc: "Test to check if satisfying rule is reused",
			missingDepsMap: map[*bazel.Rule]map[ClassName][]bazel.Label{
				bazel.NewRule("", "java/a", "Jade", nil): {"b.Foo": {"//java/b:Foo"}, "c.Bar": {"//java/b:Foo", "//java/c:Bar"}},
			},
			input: "1\n",
			want: map[*bazel.Rule][]bazel.Label{
				bazel.NewRule("", "java/a", "Jade", nil): {"//java/b:Foo"},
			},
		},
	}
	for _, test := range tests {
		in := bytes.NewReader([]byte(test.input))
		actual, err := SelectDepsToAdd(in, test.missingDepsMap)
		if err != nil {
			t.Errorf("%s: SelectDepsToAdd(%s, %s) returned unexpected error:\n%v", test.desc, test.input, test.missingDepsMap, err)
		}
		if diff := cmp.Diff(actual, test.want, sortRuleKeys); diff != "" {
			t.Errorf("Diff in SelectDepsToAdd (-got +want):\n%s", diff)
		}
	}
}
