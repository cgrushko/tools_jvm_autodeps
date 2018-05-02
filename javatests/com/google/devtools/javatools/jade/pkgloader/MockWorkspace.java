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

package com.google.devtools.javatools.jade.pkgloader;

import com.google.devtools.build.lib.vfs.FileSystemUtils;
import com.google.devtools.build.lib.vfs.Path;
import java.io.IOException;

/**
 * MockWorkspace creates Bazel workspace directories with all required files, for testing purposes.
 */
public class MockWorkspace {

  /**
   * create() creates a Bazel workspace for use with PackageLoader.
   *
   * Its behavior is tied to a specific version of Bazel, and might stop working if Bazel changes
   * significantly. Still, I think it's easier to update this once in a while than to interface with
   * Bazel.
   *
   * This is a Java version of grpcloader/mockworkspace_test.go.
   */
  static void create(Path workspaceRoot, Path installBase, Path outputBase) throws IOException {
    workspaceRoot.createDirectoryAndParents();
    FileSystemUtils.writeContent(workspaceRoot.getRelative("WORKSPACE"), new byte[0]);

    Path embeddedBinaries = installBase.getRelative("_embedded_binaries/");
    Path embeddedTools = mockEmbeddedTools(embeddedBinaries);

    Path bazelToolsRepo = outputBase.getRelative("external/");
    bazelToolsRepo.createDirectoryAndParents();
    bazelToolsRepo.getChild("bazel_tools").createSymbolicLink(embeddedTools);
  }

  private static Path mockEmbeddedTools(Path embeddedBinaries) throws IOException {
    Path tools = embeddedBinaries.getRelative("embedded_tools");
    tools.getRelative("tools/cpp").createDirectoryAndParents();
    tools.getRelative("tools/osx").createDirectoryAndParents();
    FileSystemUtils.writeIsoLatin1(tools.getRelative("WORKSPACE"), "");
    FileSystemUtils.writeIsoLatin1(tools.getRelative("tools/cpp/BUILD"), "");
    FileSystemUtils.writeIsoLatin1(
        tools.getRelative("tools/cpp/cc_configure.bzl"),
        "def cc_configure(*args, **kwargs):",
        "    pass");
    FileSystemUtils.writeIsoLatin1(tools.getRelative("tools/osx/BUILD"), "");
    FileSystemUtils.writeIsoLatin1(
        tools.getRelative("tools/osx/xcode_configure.bzl"),
        "def xcode_configure(*args, **kwargs):",
        "    pass");

    return tools;
  }
}