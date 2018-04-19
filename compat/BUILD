load("@io_bazel_rules_go//go:def.bzl", "go_library")

# keep
go_library(
    name = "go_default_library",
    srcs = ["compat.go"],
    importpath = "github.com/bazelbuild/tools_jvm_autodeps/compat",
    visibility = ["//visibility:public"],
    deps = ["@io_bazel_rules_go//go/tools/bazel:go_default_library"],
)
