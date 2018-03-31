java_test(
    name = "SerializerTest",
    srcs = ["SerializerTest.java"],
    deps = [
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
