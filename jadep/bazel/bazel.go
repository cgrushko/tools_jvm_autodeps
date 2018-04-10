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

// Package bazel contains types representing Bazel concepts, such as packages and rules.
package bazel

import (
	"fmt"
	"path"
	"strings"
)

// Label is a rule's label, e.g. //foo:bar.
type Label string

// Split splits a label to its package name and rule name parts.
// Example: //foo:bar --> "foo", "bar"
func (l Label) Split() (pkgName, ruleName string) {
	s := strings.TrimPrefix(string(l), "//")
	i := strings.IndexByte(s, ':')
	if i == -1 {
		return s, s[strings.LastIndexByte(s, '/')+1:]
	}
	return s[:i], s[i+1:]
}

// ParseRelativeLabel parses a label, not necessarily absolute,
// relative to some package.
//
// If the label is absolute, for instance "//a/b" or "//a/b:c", the
// result is the same as ParseAbsoluteLabel.  If the label is relative,
// for instance ":foo" or "bar.go", the result is the same as
// ParseAbsoluteLabel on "//pkg:foo" or "//pkg:bar.go", respectively.
func ParseRelativeLabel(pkg, s string) (Label, error) {
	if strings.HasPrefix(s, "//") || strings.HasPrefix(s, "@") {
		return ParseAbsoluteLabel(s)
	}
	if strings.Count(pkg, "//") > 1 {
		return Label(""), fmt.Errorf("package name %q contains '//' more than once", pkg)
	}
	if s == "" {
		return Label(""), fmt.Errorf("empty label")
	}
	colonIdx := strings.Index(s, ":")
	if colonIdx > 0 {
		return Label(""), fmt.Errorf("label %q doesn't start with // or @, but also contains a colon", s)
	}
	if s[0] == ':' {
		s = s[1:]
	}
	if strings.HasPrefix(pkg, "@") {
		return Label(pkg + ":" + s), nil
	}
	return Label("//" + pkg + ":" + s), nil
}

// ParseAbsoluteLabel parses a label string in absolute form, such as
// "//aaa/bbb:ccc/ddd" or "//aaa/bbb".
//
// See https://bazel.build/versions/master/docs/build-ref.html#labels.
func ParseAbsoluteLabel(s string) (Label, error) {
	if !strings.HasPrefix(s, "//") && !strings.HasPrefix(s, "@") {
		return Label(""), fmt.Errorf("absolute label must start with // or @, %q is neither", s)
	}
	i := strings.Index(s, "//")
	if i < 0 {
		return Label(""), fmt.Errorf("invalid label %q", s)
	}

	// Bazel accepts invalid labels starting with more than two slashes,
	// thus so must we for now.
	s = strings.TrimLeft(s, "/")

	var pkg, name string
	if i = strings.IndexByte(s, ':'); i < 0 {
		// "//foo/bar"
		pkg = s
		name = path.Base(s)
	} else {
		// "//foo/bar:wiz"
		pkg = s[:i]
		name = s[i+1:]
	}
	if strings.Count(pkg, "//") > 1 {
		return Label(""), fmt.Errorf("package name %q contains '//' more than once", pkg)
	}
	if strings.Index(name, "//") != -1 {
		return Label(""), fmt.Errorf("target name %q contains '//'", pkg)
	}
	if strings.Index(name, ":") != -1 {
		return Label(""), fmt.Errorf("target name %q contains ':'", pkg)
	}

	if strings.HasPrefix(pkg, "@") {
		return Label(pkg + ":" + name), nil
	}
	return Label("//" + pkg + ":" + name), nil
}

// UnknownAttributeValue is used as a value in Rule.Attrs to represent an attribute that is present,
// but whose value we can't represent.
// For example, 'deps = select(...)' will be represented this way.
// The value itself is meaningless.
type UnknownAttributeValue struct{}

func (v UnknownAttributeValue) String() string {
	return "UNKNOWN_ATTRIBUTE_VALUE"
}

// Rule represents a Bazel Rule.
type Rule struct {
	Schema  string                 // string representing the type of rule for example java_library
	PkgName string                 // name of the containing package, e.g. "src/main/java/"
	Attrs   map[string]interface{} // map from type of attr i.e. srcs or export to the actual content name of srcs and exports
}

// NewRule creates a new Rule.
func NewRule(schema, pkgName, ruleName string, attributes map[string]interface{}) *Rule {
	attrs := map[string]interface{}{"name": ruleName}
	for k, v := range attributes {
		attrs[k] = v
	}
	result := &Rule{
		Schema:  schema,
		PkgName: pkgName,
		Attrs:   attrs,
	}
	return result
}

// StringListAttr converts the attributes of a rule from
// an interface type to a list of strings.
// If the attribute value is not a strict list (e.g., a selector), StringListAttr returns the empty list.
func (r *Rule) StringListAttr(attrName string) []string {
	values, _ := r.Attrs[attrName].([]string)
	return values
}

// LabelListAttr converts the attributes of a rule from
// an interface type to a list of labels.
// If the attribute value is not a strict list (e.g., a selector), LabelListAttr returns the empty list.
func (r *Rule) LabelListAttr(attrName string) []Label {
	values, _ := r.Attrs[attrName].([]string)
	var ret []Label
	for _, s := range values {
		if l, err := ParseRelativeLabel(r.PkgName, s); err == nil {
			ret = append(ret, l)
		}
	}
	return ret
}

// BoolAttr converts the attributes of a rule from
// an interface type to a boolean.
// If the attribute value is not a boolean, BoolAttr returns defaultValue.
func (r *Rule) BoolAttr(attrName string, defaultValue bool) bool {
	val, ok := r.Attrs[attrName].(bool)
	if !ok {
		return defaultValue
	}
	return val
}

// IntAttr converts the attributes of a rule from
// an interface type to a boolean.
// If the attribute value is not a boolean, IntAttr returns defaultValue.
func (r *Rule) IntAttr(attrName string, defaultValue int) int {
	val, ok := r.Attrs[attrName].(int)
	if !ok {
		return defaultValue
	}
	return val
}

// StrAttr converts the attributes of a rule from
// an interface type to a string.
// If the attribute value is not a string, StrAttr returns defaultValue.
func (r *Rule) StrAttr(attrName string, defaultValue string) string {
	val, ok := r.Attrs[attrName].(string)
	if !ok {
		return defaultValue
	}
	return val
}

// LabelAttr converts the attributes of a rule from
// an interface type to a string.
// If the attribute value is not a label, LabelAttr returns an error.
func (r *Rule) LabelAttr(attrName string) (Label, error) {
	val, ok := r.Attrs[attrName].(string)
	if !ok {
		return Label(""), fmt.Errorf("%s's %s is not a string, can't parse to label", r.Label(), attrName)
	}
	l, err := ParseRelativeLabel(r.PkgName, val)
	if err != nil {
		return Label(""), fmt.Errorf("can't read %s's %s as label: %v", r.Label(), attrName, err)
	}
	return l, nil
}

// Name returns the name of the rule (e.g. "collect").
func (r *Rule) Name() string {
	s, _ := r.Attrs["name"].(string)
	return s
}

// Label returns the label of the rule (e.g. "//java:collect").
func (r *Rule) Label() Label {
	if strings.HasPrefix(r.PkgName, "@") {
		return Label(r.PkgName + ":" + r.Name())
	}
	return Label("//" + r.PkgName + ":" + r.Name())
}

// Package represents a Bazel Package.
type Package struct {
	Path              string
	DefaultVisibility []Label
	Files             map[string]string
	Rules             map[string]*Rule // maps rule name to Rule
	PackageGroups     map[string]*PackageGroup
}

// PackageGroup represents a package_group() function call in a BUILD file.
type PackageGroup struct {
	// Package specs, e.g. foo/bar, foo/... or //... in case the user wrote "//foo/bar", "//foo/..." and "//..." respectively.
	// (//foo:bar is illegal)
	Specs []string

	// Includes of package_group.
	Includes []Label
}
