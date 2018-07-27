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

import static com.google.common.truth.Truth.assertThat;
import static java.nio.charset.StandardCharsets.UTF_8;

import com.google.devtools.build.lib.vfs.DigestHashFunction;
import com.google.devtools.build.lib.vfs.FileSystem;
import com.google.devtools.build.lib.vfs.FileSystemUtils;
import com.google.devtools.build.lib.vfs.Path;
import com.google.devtools.build.lib.vfs.inmemoryfs.InMemoryFileSystem;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.LoaderRequest;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.LoaderResponse;
import java.io.IOException;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class LibTest {
  private static final PackageLoaderFactory PACKAGE_LOADER_FACTORY =
      new BazelPackageLoaderFactory();

  private static final FileSystem FILESYSTEM = new InMemoryFileSystem(DigestHashFunction.MD5);
  private Path workspaceRoot;
  private Path installBase;
  private Path outputBase;

  @Before
  public void setUp() throws IOException {
    workspaceRoot = FILESYSTEM.getPath("/workspace/");
    installBase = FILESYSTEM.getPath("/install_base/");
    outputBase = FILESYSTEM.getPath("/output_base/");
    MockWorkspace.create(workspaceRoot, installBase, outputBase);
  }

  @Test
  public void basic() throws Exception {
    workspaceRoot.getRelative("foo/bar").createDirectoryAndParents();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("foo/bar/BUILD"), UTF_8, "sh_library(name = 'Foo')");

    LoaderRequest request =
        LoaderRequest.newBuilder()
            .setWorkspaceDir(workspaceRoot.getPathString())
            .setInstallBase(installBase.getPathString())
            .setOutputBase(outputBase.getPathString())
            .addPackages("foo/bar")
            .build();
    LoaderResponse response = Lib.load(PACKAGE_LOADER_FACTORY, FILESYSTEM, request);

    assertThat(response.getPkgsMap()).hasSize(1);
    assertThat(response.getPkgsMap()).containsKey("foo/bar");
  }
}
