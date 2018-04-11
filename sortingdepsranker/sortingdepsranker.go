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

// Package sortingdepsranker ranks deps by simply sorting labels lexicographically.
package sortingdepsranker

import (
	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
)

// Ranker is a jadeplib.DepsRanker that ranks labels by their lexicographic order.
type Ranker struct{}

// Less returns true iff label1 < label2.
func (r *Ranker) Less(ctx context.Context, label1, label2 bazel.Label) bool {
	return label1 < label2
}
