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

package resolverutil

import (
	"testing"

	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSatisfiedByExistingDeps(t *testing.T) {
	tests := []struct {
		desc            string
		consumingRules  map[bazel.Label]map[bazel.Label]bool
		satisfyingRules map[jadeplib.ClassName][]bazel.Label
		want            map[jadeplib.ClassName][]bazel.Label
	}{
		{
			desc:            "No consuming rules at all",
			consumingRules:  map[bazel.Label]map[bazel.Label]bool{},
			satisfyingRules: map[jadeplib.ClassName][]bazel.Label{},
			want:            nil,
		},
		{
			desc: "Single consuming rule that doesn't contain any satisfying dep",
			consumingRules: map[bazel.Label]map[bazel.Label]bool{
				"//:foo": {"//:bar": true},
			},
			satisfyingRules: map[jadeplib.ClassName][]bazel.Label{
				"com.Zed": {"//:Zed"},
			},
			want: map[jadeplib.ClassName][]bazel.Label{},
		},
		{
			desc: "Single consuming rule that contains a satisfying dep",
			consumingRules: map[bazel.Label]map[bazel.Label]bool{
				"//:foo": {"//:bar1": true, "//:bar2": true},
			},
			satisfyingRules: map[jadeplib.ClassName][]bazel.Label{
				"com.Bar": {"//:bar1", "//:other"},
			},
			want: map[jadeplib.ClassName][]bazel.Label{
				"com.Bar": {"//:bar1"},
			},
		},
		{
			desc: "Multiple consuming rule that all contain some satisfying dep for each class name, though not the same. We return a single satisfying dep for each consuming rule",
			consumingRules: map[bazel.Label]map[bazel.Label]bool{
				"//:foo1": {"//:bar1": true, "//:bar2": true, "//:unrelated1": true},
				"//:foo2": {"//:bar3": true, "//:bar4": true, "//:unrelated2": true},
			},
			satisfyingRules: map[jadeplib.ClassName][]bazel.Label{
				"com.Bar1": {"//:bar1", "//:bar4", "//:emmo1"},
				"com.Bar2": {"//:bar2", "//:bar3", "//:emmo2"},
			},
			want: map[jadeplib.ClassName][]bazel.Label{
				"com.Bar1": {"//:bar1", "//:bar4"},
				"com.Bar2": {"//:bar2", "//:bar3"},
			},
		},
	}

	opt := cmpopts.SortSlices(func(a, b bazel.Label) bool { return a < b })

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := SatisfiedByExistingDeps(tt.consumingRules, tt.satisfyingRules)
			if diff := cmp.Diff(got, tt.want, opt); diff != "" {
				t.Errorf("SatisfiedByExistingDeps returned diff (-want +got):\n%s", diff)
			}
		})
	}
}
