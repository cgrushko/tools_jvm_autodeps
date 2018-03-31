load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@grpc_java//:java_grpc_library.bzl", "java_grpc_library")

package(default_visibility = [
    "//grpcloader:__subpackages__",
    "//java/com/google/devtools/javatools/jade/pkgloader:__subpackages__",
])

proto_library(
    name = "java_com_google_devtools_javatools_jade_pkgloader_services_proto",
    srcs = ["services.proto"],
    deps = ["//java/com/google/devtools/javatools/jade/pkgloader/messages_proto:java_com_google_devtools_javatools_jade_pkgloader_messages_proto"],
)

java_proto_library(
    name = "services_java_proto",
    deps = [":java_com_google_devtools_javatools_jade_pkgloader_services_proto"],
)

java_grpc_library(
    name = "services_java_grpc",
    srcs = [":java_com_google_devtools_javatools_jade_pkgloader_services_proto"],
    deps = [":services_java_proto"],
)

go_proto_library(
    name = "java_com_google_devtools_javatools_jade_pkgloader_services_go_proto",
    compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "github.com/bazelbuild/tools_jvm_autodeps/jadep/java/com/google/devtools/javatools/jade/pkgloader/services_proto",
    proto = ":java_com_google_devtools_javatools_jade_pkgloader_services_proto",
    deps = ["//java/com/google/devtools/javatools/jade/pkgloader/messages_proto:go_default_library"],
)

go_library(
    name = "go_default_library",
    embed = [":java_com_google_devtools_javatools_jade_pkgloader_services_go_proto"],
    importpath = "github.com/bazelbuild/tools_jvm_autodeps/jadep/java/com/google/devtools/javatools/jade/pkgloader/services_proto",
)
