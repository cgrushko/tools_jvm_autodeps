# Java Automatic Dependencies (Jadep)

Jadep is a [Bazel](http://bazel.build) `BUILD` file generator for Java projects. It adds `BUILD`
dependencies that a Java file needs, aiming for <1s response times.

Jadep is intended to manage `BUILD` files for your own code in the current Bazel workspace (as opposed to BUILD files for third-party libraries).

Jadep is not an official Google product.

[![Build status](https://badge.buildkite.com/38a87d1503f25d2cf22f75eed28b43318b91cb1a59f3d33aa3.svg)](https://buildkite.com/bazel/tools-jvm-autodeps)

[![demo](https://asciinema.org/a/hbS6FBgF91iobsgCL96m9Loxf.png)](https://asciinema.org/a/hbS6FBgF91iobsgCL96m9Loxf?autoplay=1)

**Contents**

-   [Java Automatic Dependencies (Jadep)](#java-automatic-dependencies-jadep)
    -   [Usage](#usage)
    -   [Building / Installation](#building-installation)
    -   [How does it Work?](#how-does-it-work)
        -   [Extracting Class Names](#extracting-class-names)
        -   [Resolver: File System](#resolver-file-system)
        -   [Resolver: JDK / Android SDK](#resolver-jdk-android-sdk)
        -   [Reading ("Loading") `BUILD` files](#reading-loading-build-files)
    -   [Extending / Hacking / Future Ideas](#extending-hacking-future-ideas)
    -   [Bugs](#bugs)
    -   [Contributing](#contributing)

## Usage

```
~/bin/jadep path/to/File.java
```

## Detailed Example: Migrating a Java project to Bazel

<https://github.com/cgrushko/text/blob/master/migrating-gjf-to-bazel.md>

## Building / Installation

The following will build Jadep and its persistent server, and will copy them to
`~/bin/` and `~/jadep/`.

```bash
# Jadep
mkdir -p ~/bin
mkdir -p ~/jadep

bazel build -c opt //cmd/jadep

jadep=( bazel-bin/cmd/jadep/*/jadep ) # work around https://github.com/bazelbuild/rules_go/issues/1239
cp "${jadep[0]}" ~/bin/

# PackageLoader server
bazel build -c opt --nocheck_visibility //java/com/google/devtools/javatools/jade/pkgloader:GrpcLocalServer_deploy.jar

cp bazel-bin/java/com/google/devtools/javatools/jade/pkgloader/GrpcLocalServer_deploy.jar ~/jadep/
cp scripts/pkgloader_server.sh ~/jadep/

# JDK symbols [Jadep can run without these]
bazel build //:jdk_android_builtin_class_names

cp bazel-genfiles/jdk_android_builtin_class_names.txt ~/jadep/
```

## How does it Work?

After parsing a Java file, Jadep extracts the class names it references.

It then tries to resolve each class name to `BUILD` rules that provide it, by
employing a set of strategies ("resolvers") in sequence.

Once a set of possible `BUILD` rules is found, it is filtered down according to
`visibility`, `tags` and so on.

The following subsections detail different parts of Jadep.

### Extracting Class Names

Jadep parses a Java file to obtain an AST, then partially resolves it: each
symbol is mapped to its place of definition. For example, a call to a method
maps to the method's definition.

Jadep then walks the AST and finds all

1.  symbols that must be class names based on the Java 8 grammar
2.  symbols that can be class names, and aren't defined anywhere in the same
    Java file

Unqualified class names are assumed to be in the same package as the Java file.

This technique gives pretty good results, but the semantics of Java make it
impossible to be 100% correct. For example, a subclass has access to all the
(visible) inner classes of its superclass, without having to explicitly import
them. Jadep doesn't follow inheritance chains because it means reading arbitrary
files, so it doesn't know which symbols are inherited.

### Resolver: File System

Java source files are typically organized in the file system according to their
package and class name, and this resolver utilizes this structure to find BUILD
rules.

It is based on the convention that a class named `com.foo.Bar` will be
defined in a file named `<content root>/com/foo/Bar.java`.

The `<content root>` is by default either one of `{src/main/java,
src/test/java}`.

The resolver derives a set of file names from the set of content roots and a
transformation of the class names it's looking for, and searches for BUILD rules
that have these files in their `srcs` attributes.

The resolver also handles `java_library.exports` attributes and `alias()` rules
so long as they're in the same Bazel package as the composed file name.

### Resolver: JDK / Android SDK

JDK class names (e.g. `java.util.List`) do not need any BUILD dependencies to
build, so this resolver simply maps these classes to nothing, ensuring that
Jadep won't add anything for them.

Bazel Android rules don't need dependencies for Android SDK classes, so this
resolver also handles these classes.

### Reading `BUILD` files

Since Jadep interacts with existing Bazel rules (e.g., when filtering by
`visibility`) it needs to read `BUILD` files.

We use Bazel's [Skylark
interpreter](https://github.com/bazelbuild/bazel/blob/0.10.0/src/main/java/com/google/devtools/build/lib/skyframe/packages/BazelPackageLoader.java) rather than [Buildozer](https://github.com/bazelbuild/buildtools/tree/c98ff0c6395f09b1942e6f7c42bf3ec15e3b9ca7/buildozer), because the latter is unable to interpret macros.

Since the Skylark interpreter is wrriten in Java, a persistent local [gRPC](https://grpc.io/) server is
used to avoid repeatedly paying startup costs.

## Extending / Hacking / Future Ideas

*   The [dictresolver.go](??) is a resolver that uses a plain-text class ->
    BUILD mapping encoded in CSV, and can be used as an example for how to write
    a performant resolver.
*   A Maven Central resolver would be useful - it would search class names in
    Maven Central and add their coordinates to a
    [bazel-deps](https://github.com/johnynek/bazel-deps) configuration.
*   [Kythe](http://kythe.io) could be used to generate an index that Jadep uses.

## Bugs

1.  Jadep doesn't yet handle external repositories. The `bazel.Label` data
    structure is unaware of them, as is `GrpcLocalServer`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
