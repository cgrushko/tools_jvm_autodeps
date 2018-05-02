java_test(
    name = "SerializerTest",
    srcs = ["SerializerTest.java"],
    deps = [
        ":MockWorkspace",
        "//java/com/google/devtools/javatools/jade/pkgloader:BazelPackageLoaderFactory",
        "//java/com/google/devtools/javatools/jade/pkgloader:PackageLoaderFactory",
        "//java/com/google/devtools/javatools/jade/pkgloader:Serializer",
        "//java/com/google/devtools/javatools/jade/pkgloader/messages_proto:messages_java_proto",
        "//thirdparty/jvm/com/google/guava",
        "//thirdparty/jvm/com/google/truth",
        "//thirdparty/jvm/junit",
        "@io_bazel//src/main/java/com/google/devtools/build/lib:packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/skyframe/packages",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs/inmemoryfs",
    ],
)

java_test(
    name = "BazelPackageLoaderFactoryTest",
    srcs = ["BazelPackageLoaderFactoryTest.java"],
    deps = [
        "//java/com/google/devtools/javatools/jade/pkgloader:BazelPackageLoaderFactory",
        "//thirdparty/jvm/junit",
    ],
)

java_test(
    name = "LibTest",
    srcs = ["LibTest.java"],
    deps = [
        ":MockWorkspace",
        "//java/com/google/devtools/javatools/jade/pkgloader:BazelPackageLoaderFactory",
        "//java/com/google/devtools/javatools/jade/pkgloader:Lib",
        "//java/com/google/devtools/javatools/jade/pkgloader:PackageLoaderFactory",
        "//java/com/google/devtools/javatools/jade/pkgloader/services_proto:services_java_proto",
        "//thirdparty/jvm/com/google/truth",
        "//thirdparty/jvm/junit",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs",
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs/inmemoryfs",
    ],
)

java_library(
    name = "MockWorkspace",
    srcs = ["MockWorkspace.java"],
    deps = [
        "@io_bazel//src/main/java/com/google/devtools/build/lib/vfs",
    ],
)
