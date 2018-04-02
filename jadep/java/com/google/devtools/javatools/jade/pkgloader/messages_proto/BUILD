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
