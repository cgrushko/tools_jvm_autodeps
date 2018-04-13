load("@bazel_gazelle//:def.bzl", "gazelle")

gazelle(
    name = "gazelle",
    prefix = "github.com/bazelbuild/tools_jvm_autodeps/jadep",
)

genrule(
    name = "jdk_android_builtin_class_names",
    outs = ["jdk_android_builtin_class_names.txt"],
    cmd = "$(location //cmd/list_classes_in_jar) $(JAVABASE)/jre/lib/rt.jar > $@",
    toolchains = ["@bazel_tools//tools/jdk:current_java_runtime"],
    tools = ["//cmd/list_classes_in_jar"],
)
