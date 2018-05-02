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

package grpcloader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// create() creates a Bazel workspace for use with PackageLoader.
// Its behavior is tied to a specific version of Bazel, and might stop working if Bazel changes
// significantly. Still, I think it's easier to update this once in a while than to interface with
// Bazel.
//
// This is a Go version of javatests/com/google/devtools/javatools/jade/pkgloader/MockWorkspace.java
func createWorkspace() (workspaceRoot, installBase, outputBase string, err error) {
	tmpRoot, err := ioutil.TempDir("", "jadep")
	if err != nil {
		return "", "", "", fmt.Errorf("error calling ioutil.TempDir: %v", err)
	}
	workspaceRoot = filepath.Join(tmpRoot, "workspace")
	installBase = filepath.Join(tmpRoot, "install_base")
	outputBase = filepath.Join(tmpRoot, "output_base")

	embeddedBinaries := filepath.Join(installBase, "_embedded_binaries/")
	embeddedTools, err := mockEmbeddedTools(embeddedBinaries)
	if err != nil {
		return "", "", "", err
	}

	bazelToolsRepo := filepath.Join(outputBase, "external/")
	if err := os.MkdirAll(bazelToolsRepo, 0700); err != nil {
		return "", "", "", fmt.Errorf("error calling MkdirAll: %v", err)
	}
	if err := os.Symlink(embeddedTools, filepath.Join(bazelToolsRepo, "bazel_tools")); err != nil {
		return "", "", "", fmt.Errorf("error calling Symlink: %v", err)
	}

	return workspaceRoot, installBase, outputBase, nil
}

func mockEmbeddedTools(embeddedBinaries string) (string, error) {
	tools := filepath.Join(embeddedBinaries, "embedded_tools")

	if err := os.MkdirAll(filepath.Join(tools, "tools/cpp"), 0700); err != nil {
		return "", fmt.Errorf("error calling MkdirAll: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tools, "tools/osx"), 0700); err != nil {
		return "", fmt.Errorf("error calling MkdirAll: %v", err)
	}

	if err := ioutil.WriteFile(filepath.Join(tools, "WORKSPACE"), nil, 0666); err != nil {
		return "", fmt.Errorf("error calling WriteFile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(tools, "tools/cpp/BUILD"), nil, 0666); err != nil {
		return "", fmt.Errorf("error calling WriteFile: %v", err)
	}

	contents := `
def cc_configure(*args, **kwargs):
    pass`
	if err := ioutil.WriteFile(filepath.Join(tools, "tools/cpp/cc_configure.bzl"), []byte(contents), 0666); err != nil {
		return "", fmt.Errorf("error calling WriteFile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(tools, "tools/osx/BUILD"), nil, 0666); err != nil {
		return "", fmt.Errorf("error calling WriteFile: %v", err)
	}

	contents = `
def xcode_configure(*args, **kwargs):
    pass`
	if err := ioutil.WriteFile(filepath.Join(tools, "tools/osx/xcode_configure.bzl"), []byte(contents), 0666); err != nil {
		return "", fmt.Errorf("error calling WriteFile: %v", err)
	}

	return tools, nil
}
