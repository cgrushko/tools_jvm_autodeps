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

package bazel

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLabelSplit(t *testing.T) {
	var tests = []struct {
		label        string
		wantPkgName  string
		wantRuleName string
	}{
		{"//foo:bar", "foo", "bar"},
		{"//foo/bar", "foo/bar", "bar"},
		{"//foo/bar/zoo", "foo/bar/zoo", "zoo"},
		{"//foo", "foo", "foo"},
		{"@r//foo", "@r//foo", "foo"},
	}

	for _, tt := range tests {
		pkgName, ruleName := Label(tt.label).Split()
		if pkgName != tt.wantPkgName || ruleName != tt.wantRuleName {
			t.Errorf("Label(%s).Split() = (%s, %s), want (%s, %s)", tt.label, pkgName, ruleName, tt.wantPkgName, tt.wantRuleName)
		}
	}
}

func TestParseRelativeLabel(t *testing.T) {
	tests := []struct {
		pkgName, s string
		want       Label
	}{
		{"", "//foo", "//foo:foo"},
		{"", "//foo:foo", "//foo:foo"},
		{"", "//foo:bar", "//foo:bar"},
		{"foo", ":bar", "//foo:bar"},
		{"foo", "bar", "//foo:bar"},
		{"@r//foo", "bar", "@r//foo:bar"},
		{"dontcare", "//foo/bar:bar", "//foo/bar:bar"},
		{"dontcare", "//foo/bar:baz", "//foo/bar:baz"},
		{"dontcare", "@r//foo/bar:baz", "@r//foo/bar:baz"},
	}
	for _, tt := range tests {
		got, err := ParseRelativeLabel(tt.pkgName, tt.s)
		if err != nil {
			t.Errorf("ParseLabel(%s, %s) has error %v, want nil", tt.pkgName, tt.s, err)
		}
		if got != tt.want {
			t.Errorf("ParseLabel(%s, %s) = %v, want %v", tt.pkgName, tt.s, got, tt.want)
		}
	}
}

func TestParseRelativeLabelErrors(t *testing.T) {
	tests := []struct {
		pkgName, s string
		wantErr    string
	}{
		{"dontcare", "foo:bar", `label "foo:bar" doesn't start with // or @, but also contains a colon`},
		{"@r//foo//", "bar", `package name "@r//foo//" contains '//' more than once`},
	}
	for _, tt := range tests {
		_, err := ParseRelativeLabel(tt.pkgName, tt.s)
		if diff := cmp.Diff(err.Error(), tt.wantErr); diff != "" {
			t.Errorf("ParseLabel(%s, %s) returns the wrong error (-got +want): %v", tt.pkgName, tt.s, diff)
		}
	}
}

func TestParseAbsoluteLabel(t *testing.T) {
	tests := []struct {
		s       string
		wantErr string
		want    Label
	}{
		{"//foo", "", "//foo:foo"},
		{"//foo/bar", "", "//foo/bar:bar"},
		{"foo", `absolute label must start with // or @, "foo" is neither`, ""},
		{":foo", `absolute label must start with // or @, ":foo" is neither`, ""},
		{"//foo:fo:o", `target name "foo" contains ':'`, ""},
		{"@//:foo", "", "@//:foo"},
		{"@//asd//:foo", `package name "@//asd//" contains '//' more than once`, ""},
		{"//foo:foo", "", "//foo:foo"},
		{"//foo/bar:foo", "", "//foo/bar:foo"},
		{"@r//foo/bar:foo", "", "@r//foo/bar:foo"},
		// TODO(b/36533053): Remove once fixed.
		{"////foo/bar:foo", "", "//foo/bar:foo"},
	}
	for _, tt := range tests {
		got, err := ParseAbsoluteLabel(tt.s)
		if err == nil {
			if tt.wantErr != "" {
				t.Errorf("Got no error, want %q", tt.wantErr)
			}
		} else {
			if diff := cmp.Diff(err.Error(), tt.wantErr); diff != "" {
				t.Errorf("ParseAbsoluteLabel(%s) returns the wrong error (-got +want): %v", tt.s, diff)
			}
		}
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Errorf("ParseAbsoluteLabel(%s) returns the wrong label (-got +want): %v", tt.s, diff)
		}
	}
}

func TestLabelListAttr(t *testing.T) {
	tests := []struct {
		rule *Rule
		attr string
		want []Label
	}{
		{
			rule: &Rule{PkgName: "x", Attrs: map[string]interface{}{"deps": []string{"//foo:bar", "//foo", "bar"}}},
			attr: "deps",
			want: []Label{"//foo:bar", "//foo:foo", "//x:bar"},
		},
	}
	for _, tt := range tests {
		got := tt.rule.LabelListAttr(tt.attr)
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Errorf("rule.LabelListAttr(%s) has diff (-got +want):\nrule = %v\n\ndiff = %v", tt.attr, tt.rule, diff)
		}
	}
}

func TestLabelAttr(t *testing.T) {
	tests := []struct {
		rule    *Rule
		attr    string
		want    Label
		wantErr string
	}{
		{
			rule: &Rule{PkgName: "x", Attrs: map[string]interface{}{"attr": "//foo:bar"}},
			attr: "attr",
			want: Label("//foo:bar"),
		},
		{
			rule: &Rule{PkgName: "x", Attrs: map[string]interface{}{"attr": "//foo"}},
			attr: "attr",
			want: Label("//foo:foo"),
		},
		{
			rule: &Rule{PkgName: "x", Attrs: map[string]interface{}{"attr": ":bar"}},
			attr: "attr",
			want: Label("//x:bar"),
		},
		{
			rule:    &Rule{PkgName: "x", Attrs: map[string]interface{}{"attr": []string(nil)}},
			attr:    "attr",
			wantErr: "//x:'s attr is not a string, can't parse to label",
		},
		{
			rule:    &Rule{PkgName: "x", Attrs: map[string]interface{}{"attr": "a:"}},
			attr:    "attr",
			wantErr: `can't read //x:'s attr as label: label "a:" doesn't start with // or @, but also contains a colon`,
		},
	}
	for _, tt := range tests {
		got, err := tt.rule.LabelAttr(tt.attr)
		if err == nil {
			if tt.wantErr != "" {
				t.Errorf("Got no error, want %q", tt.wantErr)
			}
		} else {
			if diff := cmp.Diff(err.Error(), tt.wantErr); diff != "" {
				t.Errorf("rule.LabelListAttr(%s) returns the wrong error (-got +want): %v", tt.attr, diff)
			}
		}
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Errorf("rule.LabelListAttr(%s) has diff (-got +want):\nrule = %v\n\ndiff = %v", tt.attr, tt.rule, diff)
		}
	}
}
