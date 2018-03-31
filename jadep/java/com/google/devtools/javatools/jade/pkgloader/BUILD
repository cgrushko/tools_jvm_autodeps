package(default_visibility = [
    "//grpcloader:__subpackages__",
])

java_binary(
    name = "GrpcLocalServer",
    srcs = ["GrpcLocalServer.java"],
    deps = [
        ":BazelPackageLoaderFactory",
        ":Lib",
        ":PackageLoaderFactory",
        ":Serializer",
        "//java/com/google/devtools/javatools/jade/pkgloader/services_proto:services_java_grpc",
        "//java/com/google/devtools/javatools/jade/pkgloader/services_proto:services_java_proto",
        "//thirdparty/jvm/args4j",
        "//thirdparty/jvm/com/google/guava",
        "//thirdparty/jvm/io/grpc:grpc_core",
        "//thirdparty/jvm/io/grpc:grpc_netty",
        "//thirdparty/jvm/io/grpc:grpc_services",
        "//thirdparty/jvm/io/grpc:grpc_stub",
        "//thirdparty/jvm/io/netty:netty_all",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:events",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:unix",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/skyframe/packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs",
    ],
)

java_library(
    name = "Lib",
    srcs = ["Lib.java"],
    deps = [
        ":PackageLoaderFactory",
        ":Serializer",
        "//java/com/google/devtools/javatools/jade/pkgloader/services_proto:services_java_proto",
        "//thirdparty/jvm/com/google/guava",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:events",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/skyframe/packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs",
    ],
)

java_library(
    name = "Serializer",
    srcs = ["Serializer.java"],
    deps = [
        "//java/com/google/devtools/javatools/jade/pkgloader/messages_proto:messages_java_proto",
        "//thirdparty/jvm/com/google/guava",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:packages",
    ],
)

java_library(
    name = "PackageLoaderFactory",
    srcs = ["PackageLoaderFactory.java"],
    deps = [
        "@io_bazel//src/main/java/com/google/devtools/build/lib/skyframe/packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs",
    ],
)

java_library(
    name = "BazelPackageLoaderFactory",
    srcs = ["BazelPackageLoaderFactory.java"],
    deps = [
        ":PackageLoaderFactory",
        "//thirdparty/jvm/com/google/guava",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:events",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/skyframe/packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs",
    ],
)
