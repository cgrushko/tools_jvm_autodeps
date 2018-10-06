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
import static com.google.devtools.javatools.jade.pkgloader.Serializer.serialize;
import static java.nio.charset.StandardCharsets.UTF_8;

import com.google.common.collect.ImmutableSet;
import com.google.devtools.build.lib.cmdline.PackageIdentifier;
import com.google.devtools.build.lib.packages.Package;
import com.google.devtools.build.lib.skyframe.packages.PackageLoader;
import com.google.devtools.build.lib.vfs.DigestHashFunction;
import com.google.devtools.build.lib.vfs.FileSystem;
import com.google.devtools.build.lib.vfs.FileSystemUtils;
import com.google.devtools.build.lib.vfs.Path;
import com.google.devtools.build.lib.vfs.Root;
import com.google.devtools.build.lib.vfs.inmemoryfs.InMemoryFileSystem;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.messages.Messages;
import java.io.IOException;
import java.util.Map;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class SerializerTest {

  private static final PackageLoaderFactory PACKAGE_LOADER_FACTORY =
      new BazelPackageLoaderFactory();

  private PackageLoader packageLoader;
  private Path workspaceRoot;

  @Before
  public void setUp() throws IOException {
    FileSystem fs = new InMemoryFileSystem(DigestHashFunction.MD5);
    workspaceRoot = fs.getPath("/workspace/");
    Path installBase = fs.getPath("/install_base/");
    Path outputBase = fs.getPath("/output_base/");
    MockWorkspace.create(workspaceRoot, installBase, outputBase);

    packageLoader =
        PACKAGE_LOADER_FACTORY.create(Root.fromPath(workspaceRoot), installBase, outputBase);
  }

  @Test
  public void basic() throws Exception {
    FileSystemUtils.writeContent(workspaceRoot.getRelative("Foo.java"), new byte[0]);
    FileSystemUtils.writeContent(workspaceRoot.getRelative("Bar.java"), new byte[0]);
    FileSystemUtils.writeContent(workspaceRoot.getRelative("Zoo.java"), new byte[0]);
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("BUILD"),
        UTF_8,
        "java_test(name = 'Foo', srcs = ['Foo.java'], deps = [':Bar'])",
        "java_library(name = 'Bar', srcs = ['Bar.java'], exports = [':Zoo'])",
        "java_library(name = 'Zoo', srcs = ['Zoo.java'], deps = ['//other:Bla'])");

    Messages.Pkg pkg = loadAndSerialize("");
    assertThat(pkg.getPath()).isEqualTo("/workspace");

    assertThat(pkg.getFilesMap()).containsEntry("Foo.java", "");
    assertThat(pkg.getFilesMap()).containsEntry("Bar.java", "");
    assertThat(pkg.getFilesMap()).containsEntry("Zoo.java", "");

    assertThat(pkg.getRulesMap()).containsKey("Foo");
    assertThat(pkg.getRulesMap()).containsKey("Bar");
    assertThat(pkg.getRulesMap()).containsKey("Zoo");

    assertThat(pkg.getRulesMap().get("Foo").getKind()).isEqualTo("java_test");
    assertThat(pkg.getRulesMap().get("Bar").getKind()).isEqualTo("java_library");
    assertThat(pkg.getRulesMap().get("Zoo").getKind()).isEqualTo("java_library");

    assertThat(
            pkg.getRulesMap()
                .get("Foo")
                .getAttributesMap()
                .get("srcs")
                .getListOfStrings()
                .getStrList())
        .containsExactly("Foo.java");
    assertThat(
            pkg.getRulesMap()
                .get("Foo")
                .getAttributesMap()
                .get("deps")
                .getListOfStrings()
                .getStrList())
        .containsExactly("Bar");
    assertThat(
            pkg.getRulesMap()
                .get("Bar")
                .getAttributesMap()
                .get("exports")
                .getListOfStrings()
                .getStrList())
        .containsExactly("Zoo");
    assertThat(
            pkg.getRulesMap()
                .get("Zoo")
                .getAttributesMap()
                .get("deps")
                .getListOfStrings()
                .getStrList())
        .containsExactly("//other:Bla");
  }

  /**
   * Two things to note about the output:
   * <li>package_group's '//a/...' is returned as 'a/...', but '//...' is returned as '//...'.
   * <li>package_group's includes are absolute labels, even if they can be written in relative form.
   * <li>visibility lists are absolute labels, even if the label is in the same package.
   */
  @Test
  public void visibilityAndPackageGroups() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "package(default_visibility = ['//bar:__subpackages__', '__pkg__'])",
        "package_group(name = 'group1', packages = ['//a/...', '//b', '//...'],",
        "     includes = ['//bla:bla', ':baz'])",
        "package_group(name = 'group2', packages = ['//c/...'])",
        "java_library(name = 'Foo', visibility = ['//foo:__pkg__', '//x:group1', 'group2',",
        "    '//other:group3'])");

    Messages.Pkg pkg = loadAndSerialize("x");

    assertThat(pkg.getDefaultVisibilityList())
        .containsExactly("//bar:__subpackages__", "//x:__pkg__");

    Map<String, Messages.PackageGroup> groups = pkg.getPackageGroupsMap();
    assertThat(groups.keySet()).containsExactly("group1", "group2");
    assertThat(groups.get("group1").getPackageSpecsList()).containsExactly("a/...", "b", "//...");
    assertThat(groups.get("group1").getIncludesList()).containsExactly("//bla:bla", "//x:baz");

    Messages.Attribute fooVis = pkg.getRulesMap().get("Foo").getAttributesMap().get("visibility");
    assertThat(fooVis.getListOfStrings().getStrList())
        .containsExactly("//foo:__pkg__", "//x:group1", "//x:group2", "//other:group3");
  }

  /** By default, rules have visibility = private. */
  @Test
  public void visibilityAlwaysPresent() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"), UTF_8, "java_library(name = 'Foo')");
    Messages.Pkg pkg = loadAndSerialize("x");
    Messages.Attribute fooVis = pkg.getRulesMap().get("Foo").getAttributesMap().get("visibility");
    assertThat(fooVis.getListOfStrings().getStrList()).containsExactly("//visibility:private");
  }

  /**
   * A package's default_visibility is applied to all rules in the package. In other words, it's
   * enough for a caller to look at the visibility attribute of a rule, no need to look at the
   * package declaration.
   */
  @Test
  public void defaultVisibility() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "package(default_visibility = ['//bar:__subpackages__', '__pkg__'])",
        "java_library(name = 'Foo')");
    Messages.Pkg pkg = loadAndSerialize("x");
    Messages.Attribute fooVis = pkg.getRulesMap().get("Foo").getAttributesMap().get("visibility");
    assertThat(fooVis.getListOfStrings().getStrList())
        .containsExactly("//bar:__subpackages__", "//x:__pkg__");
  }

  /**
   * By default, java_library has testonly=0 and java_test has testonly=1. Check that we get these
   * values.
   */
  @Test
  public void testonlyAlwaysPresent() throws Exception {
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("BUILD"),
        UTF_8,
        "java_library(name = 'Foo')",
        "java_test(name = 'FooTest')");
    Messages.Pkg pkg = loadAndSerialize("");
    assertThat(pkg.getRulesMap().get("Foo").getAttributesMap().get("testonly").getB()).isFalse();
    assertThat(pkg.getRulesMap().get("FooTest").getAttributesMap().get("testonly").getB()).isTrue();
  }

  /**
   * A package's default_testonly is applied to all rules in the package. In other words, it's
   * enough for a caller to look at the testonly attribute of a rule, no need to look at the package
   * declaration.
   */
  @Test
  public void defaultTestonly() throws Exception {
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("BUILD"),
        UTF_8,
        "package(default_testonly = 1)",
        "java_library(name = 'Foo')");
    Messages.Pkg pkg = loadAndSerialize("");
    assertThat(pkg.getRulesMap().get("Foo").getAttributesMap().get("testonly").getB()).isTrue();
  }

  /** Google-specific: everything under javatests/ is considered testonly=1. */
  @Test
  public void testonlyTrueInJavatests() throws Exception {
    workspaceRoot.getRelative("javatests/x").createDirectoryAndParents();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("javatests/x/BUILD"), UTF_8, "java_library(name = 'Foo')");
    Messages.Pkg pkg = loadAndSerialize("javatests/x");
    assertThat(pkg.getRulesMap().get("Foo").getAttributesMap().get("testonly").getB()).isTrue();
  }

  /**
   * When there's no deprecation and no default_deprecation, we know for certain there's no
   * deprecation notice, so we don't serialize it.
   */
  @Test
  public void noDeprecation() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"), UTF_8, "java_library(name = 'Foo')");
    Messages.Pkg pkg = loadAndSerialize("x");
    assertThat(pkg.getRulesMap().get("Foo").getAttributesMap()).doesNotContainKey("deprecation");
  }

  /**
   * A package's default_deprecation is applied to all rules in the package. In other words, it's
   * enough for a caller to look at the deprecation attribute of a rule, no need to look at the
   * package declaration.
   */
  @Test
  public void defaultDeprecation() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "package(default_deprecation = 'meow')",
        "java_library(name = 'Foo')");
    Messages.Pkg pkg = loadAndSerialize("x");
    Messages.Attribute deprecation =
        pkg.getRulesMap().get("Foo").getAttributesMap().get("deprecation");
    assertThat(deprecation.getS()).isEqualTo("meow");
  }

  @Test
  public void ruleKindsToSerialize() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "java_library(name = 'Foo')",
        "filegroup(name = 'Bar')");
    Package p = packageLoader.loadPackage(PackageIdentifier.createInMainRepo("x"));

    {
      Messages.Pkg s = serialize(p, ImmutableSet.of("java_library"));
      assertThat(s.getRulesMap().keySet()).containsExactly("Foo");
    }

    {
      Messages.Pkg s = serialize(p, ImmutableSet.of());
      assertThat(s.getRulesMap().keySet()).containsExactly("Foo", "Bar");
    }
  }

  Messages.Pkg loadAndSerialize(String pkgId) {
    try {
      return serialize(
          packageLoader.loadPackage(PackageIdentifier.createInMainRepo(pkgId)), ImmutableSet.of());
    } catch (Exception e) {
      return null;
    }
  }

  @Test
  public void flattenListSelects() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "java_library(name = 'Foo', srcs = ['Foo.java'] + ",
        "    select({':cond1': ['Bar1.java'], ':cond2': ['Bar2.java']}))");
    Messages.Pkg pkg = loadAndSerialize("x");
    Messages.Attribute fooVis = pkg.getRulesMap().get("Foo").getAttributesMap().get("srcs");
    assertThat(fooVis.getListOfStrings().getStrList())
        .containsExactly("Foo.java", "Bar1.java", "Bar2.java")
        .inOrder();
  }

  /** Assert that we serialize the default value of a scalar attribute set to a select(). */
  @Test
  public void scalarSelectsGetDefault() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "java_binary(name = 'Foo', stamp = ",
        "    select({'//conditions:default': -1, ':cond1': 1}))");
    Messages.Pkg pkg = loadAndSerialize("x");
    Messages.Attribute fooVis = pkg.getRulesMap().get("Foo").getAttributesMap().get("stamp");
    assertThat(fooVis.getI()).isEqualTo(-1);
  }

  /** A scalar select() without a default value is serialized as "unknown". */
  @Test
  public void scalarSelectsNoDefault() throws Exception {
    workspaceRoot.getRelative("x").createDirectory();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("x/BUILD"),
        UTF_8,
        "java_library(name = 'Foo', neverlink = ",
        "    select({':cond1': False, ':cond2': True}))");
    Messages.Pkg pkg = loadAndSerialize("x");
    Messages.Attribute fooVis = pkg.getRulesMap().get("Foo").getAttributesMap().get("neverlink");
    assertThat(fooVis.hasUnknown()).isTrue();
  }

  @Test
  public void packagePaths() throws Exception {
    workspaceRoot.getRelative("foo/bar").createDirectoryAndParents();
    FileSystemUtils.writeLinesAs(
        workspaceRoot.getRelative("foo/bar/BUILD"), UTF_8, "java_test(name = 'Foo')");

    Messages.Pkg pkg = loadAndSerialize("foo/bar");
    assertThat(pkg.getPath()).isEqualTo("/workspace/foo/bar");
  }
}
