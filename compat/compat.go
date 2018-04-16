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

package compat

import (
	"context"
	"net"
	"os"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func FileStat(ctx context.Context, name string) (interface{}, error) {
	return os.Stat(name)
}

func NewLocalSpan(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func RunfilesPath(path string) string {
	r, _ := bazel.Runfile(path)
	return r
}

func PickUnusedPort() (int, func(), error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, func() {}, nil
}
