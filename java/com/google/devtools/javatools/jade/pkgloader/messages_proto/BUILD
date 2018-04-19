load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

package(default_visibility = [
    "//grpcloader:__subpackages__",
    "//java/com/google/devtools/javatools/jade/pkgloader:__subpackages__",
])

proto_library(
    name = "java_com_google_devtools_javatools_jade_pkgloader_messages_proto",
    srcs = ["messages.proto"],
)

java_proto_library(
    name = "messages_java_proto",
    deps = [":java_com_google_devtools_javatools_jade_pkgloader_messages_proto"],
)

go_proto_library(
    name = "java_com_google_devtools_javatools_jade_pkgloader_messages_go_proto",
    importpath = "github.com/bazelbuild/tools_jvm_autodeps/java/com/google/devtools/javatools/jade/pkgloader/messages_proto",
    proto = ":java_com_google_devtools_javatools_jade_pkgloader_messages_proto",
)

go_library(
    name = "go_default_library",
    embed = [":java_com_google_devtools_javatools_jade_pkgloader_messages_go_proto"],
    importpath = "github.com/bazelbuild/tools_jvm_autodeps/java/com/google/devtools/javatools/jade/pkgloader/messages_proto",
)
