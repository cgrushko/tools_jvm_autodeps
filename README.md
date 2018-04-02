# Automatic Dependency Management Tools for JVM Languages

This repository contains tools to manage `deps` of Bazel rules for JVM
languages.

It is organized as a set of independent Bazel repositories. To build a tool,
`cd` into its subdirectory and consult its `README` file.

For example,

    cd jadep/
    bazel build //cmd/jadep  # See README in subdirectory for details

Tools:

* Jadep - adds `BUILD` dependencies that a Java file needs, aiming for <1s response times.
