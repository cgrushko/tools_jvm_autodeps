load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "grpcloader_test.go",
        "mockworkspace_test.go",
    ],
    args = ["--pkg_loader_executable=$(location //java/com/google/devtools/javatools/jade/pkgloader:GrpcLocalServer)"],
    data = [
        "//java/com/google/devtools/javatools/jade/pkgloader:GrpcLocalServer",
    ],
    embed = [":go_default_library"],
    deps = [
        "//bazel:go_default_library",
        "//compat:go_default_library",
        "//java/com/google/devtools/javatools/jade/pkgloader/services_proto:go_default_library",
        "@com_github_google_go_cmp//cmp:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
    ],
)

go_library(
    name = "go_default_library",
    srcs = ["grpcloader.go"],
    importpath = "github.com/bazelbuild/tools_jvm_autodeps/grpcloader",
    visibility = ["//visibility:public"],
    deps = [
        "//bazel:go_default_library",
        "//java/com/google/devtools/javatools/jade/pkgloader/messages_proto:go_default_library",
        "//java/com/google/devtools/javatools/jade/pkgloader/services_proto:go_default_library",
        "//vlog:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_google_grpc//codes:go_default_library",
        "@org_golang_google_grpc//status:go_default_library",
    ],
)
