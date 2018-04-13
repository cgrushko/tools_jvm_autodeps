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


import com.google.common.collect.ImmutableMap;
import com.google.common.collect.ImmutableSet;
import com.google.devtools.build.lib.cmdline.LabelSyntaxException;
import com.google.devtools.build.lib.cmdline.PackageIdentifier;
import com.google.devtools.build.lib.packages.NoSuchPackageException;
import com.google.devtools.build.lib.skyframe.packages.PackageLoader;
import com.google.devtools.build.lib.vfs.FileSystem;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.LoaderRequest;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.LoaderResponse;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Set;
import java.util.logging.Level;
import java.util.logging.Logger;

/** `pkgloader` allows clients to load Bazel packages without calling Bazel. */
public class Lib {

  private static final Logger logger = Logger.getLogger("Lib");

  /** load loads packages according to 'request', in the file system 'fileSystem'. */
  static LoaderResponse load(
      PackageLoaderFactory packageLoaderFactory, FileSystem fileSystem, LoaderRequest request) {
    logger.info("Start of 'load'");
    PackageLoader loader =
        packageLoaderFactory.create(
            fileSystem.getPath(request.getWorkspaceDir()),
            fileSystem.getPath(request.getInstallBase()),
            fileSystem.getPath(request.getOutputBase()));

    HashSet<PackageIdentifier> pkgIds = new HashSet<>();
    HashMap<PackageIdentifier, String> pkgNames = new HashMap<>();
    for (String pkgName : request.getPackagesList()) {
      try {
        PackageIdentifier pkgId = PackageIdentifier.parse(pkgName).makeAbsolute();
        pkgIds.add(pkgId);
        pkgNames.put(pkgId, pkgName);
      } catch (LabelSyntaxException e) {
        // TODO: return an error to load()'s caller for propagation to the user.
        logger.log(Level.WARNING, "Invalid package label", e);
      }
    }
    ImmutableMap<PackageIdentifier, PackageLoader.PackageOrException> pkgs;
    try {
      pkgs = loader.loadPackages(pkgIds);
    } catch (Exception e) {
      return LoaderResponse.getDefaultInstance();
    }

    Set<String> ruleKindsToSerialize = ImmutableSet.copyOf(request.getRuleKindsToSerializeList());

    LoaderResponse.Builder response = LoaderResponse.newBuilder();
    pkgs.forEach(
        (pkgId, pkg) -> {
          try {
            response.putPkgs(
                pkgNames.get(pkgId),
                Serializer.serialize(pkg.get(), ruleKindsToSerialize));
          } catch (NoSuchPackageException e) {
            logger.log(Level.FINE, String.format("No such package: %s", pkgId), e);
          }
        });
    logger.info("End of 'load'");
    return response.build();
  }
}
