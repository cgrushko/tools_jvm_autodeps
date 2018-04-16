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

// Package dictresolver resolves according to an in-memory map.
package dictresolver

import (
	"encoding/csv"
	"fmt"
	"io"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/future"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/pkgloading"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/resolverutil"
)

// Resolver resolves class names according to an in-memory map.
type Resolver struct {
	name string

	// dict is a map[jadeplib.ClassName][]bazel.Label. Resolver resolves class name C to the Bazel rules dict[c].
	dict *future.Value

	loader pkgloading.Loader
}

// NewResolver returns a new Resolver.
func NewResolver(name string, dict *future.Value, loader pkgloading.Loader) *Resolver {
	return &Resolver{name, dict, loader}
}

// Name returns a description of the resolver.
func (r *Resolver) Name() string {
	return r.name
}

// Resolve resolves class names according to an in-memory map.
func (r *Resolver) Resolve(ctx context.Context, classNames []jadeplib.ClassName, consumingRules map[bazel.Label]map[bazel.Label]bool) (map[jadeplib.ClassName][]*bazel.Rule, error) {
	dict := r.dict.Get().(map[jadeplib.ClassName][]bazel.Label)

	candidates := make(map[jadeplib.ClassName][]bazel.Label)
	for _, cls := range classNames {
		if labels, ok := dict[cls]; ok {
			candidates[cls] = append(candidates[cls], labels...)
		}
	}

	// Skip LoadRules call for class names that are already satisfied by a consuming rule.
	alreadySat := resolverutil.SatisfiedByExistingDeps(consumingRules, candidates)
	for cls := range alreadySat {
		delete(candidates, cls)
	}

	// Load rules mentioned in 'candidates'.
	var labels []bazel.Label
	for _, c := range candidates {
		labels = append(labels, c...)
	}
	rules, _, err := pkgloading.LoadRules(ctx, r.loader, labels)
	if err != nil {
		return nil, err
	}

	// Convert 'candidates' to a map class name --> rule.
	result := make(map[jadeplib.ClassName][]*bazel.Rule)
	for classname, labels := range alreadySat {
		for _, label := range labels {
			p, r := label.Split()
			result[classname] = append(result[classname], bazel.NewRule("", p, r, nil))
		}
	}
	for classname, labels := range candidates {
		result[classname] = nil
		for _, label := range labels {
			if r, ok := rules[label]; ok {
				result[classname] = append(result[classname], r)
			}
		}
	}

	return result, nil
}

// ReadDictFromCSV reads a className --> []bazel.Label map from a CSV file.
// The format is:
// className,label1,label2,...
//
// If there are no labels, a mapping to nil is returned.
// Labels must be in absolute form. Invalid labels are silently ignored.
func ReadDictFromCSV(reader io.Reader) (map[jadeplib.ClassName][]bazel.Label, error) {
	r := csv.NewReader(reader)
	r.ReuseRecord = true
	r.FieldsPerRecord = -1 // allow each record to have different number of columns.
	result := make(map[jadeplib.ClassName][]bazel.Label)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV file: %v", err)
		}
		cls := jadeplib.ClassName(record[0])
		result[cls] = nil
		for i := 1; i < len(record); i++ {
			lbl, err := bazel.ParseAbsoluteLabel(record[i])
			if err == nil {
				result[cls] = append(result[cls], lbl)
			}
		}
	}
	return result, nil
}
