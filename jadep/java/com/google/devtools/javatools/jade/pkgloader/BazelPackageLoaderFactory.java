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

import com.google.common.eventbus.EventBus;
import com.google.devtools.build.lib.events.PrintingEventHandler;
import com.google.devtools.build.lib.events.Reporter;
import com.google.devtools.build.lib.skyframe.packages.BazelPackageLoader;
import com.google.devtools.build.lib.skyframe.packages.PackageLoader;
import com.google.devtools.build.lib.vfs.Path;

public class BazelPackageLoaderFactory implements PackageLoaderFactory {
  private static final Reporter REPORTER =
      new Reporter(new EventBus(), PrintingEventHandler.ERRORS_TO_STDERR);

  @Override
  public PackageLoader create(Path workspaceDir, Path installBase, Path outputBase) {
    return BazelPackageLoader.builder(workspaceDir, installBase, outputBase)
        .useDefaultSkylarkSemantics()
        .setReporter(REPORTER)
        .setLegacyGlobbingThreads(400)
        .setSkyframeThreads(300)
        .build();
  }
}
