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

// Package graphs provides functions related to graphs.
package graphs

// DFS runs a DFS on the provided graph.
// f is called when a node is visited.
func DFS(graph map[string][]string, startingNode string, f func(node string)) {
	visited := make(map[string]bool)
	var visit func(u string)
	visit = func(u string) {
		if _, ok := visited[u]; ok {
			return
		}
		visited[u] = true
		f(u)

		for _, v := range graph[u] {
			visit(v)
		}
	}

	visit(startingNode)
}
