# Bazel code, required for PackageLoader

http_archive(
    name = "io_bazel",
    sha256 = "09c66b94356c82c52f212af52a81ac28eb06de1313755a2f23eeef84d167b36c",
    urls = ["https://releases.bazel.build/0.16.1/release/bazel-0.16.1-dist.zip"],
)

# Buildozer, to manipulate BUILD files

http_archive(
    name = "com_github_bazelbuild_buildtools",
    strip_prefix = "buildtools-b8569631d3e67c7b83279157dc659903f92e919b",
    type = "zip",
    urls = ["https://github.com/bazelbuild/buildtools/archive/b8569631d3e67c7b83279157dc659903f92e919b.zip"],
)

# Protobuf

http_archive(
    name = "com_google_protobuf",
    sha256 = "1f8b9b202e9a4e467ff0b0f25facb1642727cdf5e69092038f15b37c75b99e45",
    strip_prefix = "protobuf-3.5.1",
    urls = ["https://github.com/google/protobuf/archive/v3.5.1.zip"],
)

# Go - bind external repos

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "4d8d6244320dd751590f9100cf39fd7a4b75cd901e1f3ffdfd6f048328883695",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.9.0/rules_go-0.9.0.tar.gz",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "0103991d994db55b3b5d7b06336f8ae355739635e0c2379dea16b8213ea5a223",
    url = "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.9/bazel-gazelle-0.9.tar.gz",
)

# gRPC for Go

load("@io_bazel_rules_go//go:def.bzl", "go_repository", "go_rules_dependencies", "go_register_toolchains")

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",  # use existing generated code
    commit = "8e4536a86ab602859c20df5ebfd0bd4228d08655",  # v1.10.0
    importpath = "google.golang.org/grpc",
)

# Go - load additional repos.

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

go_repository(
    name = "com_github_google_go_cmp",
    commit = "v0.2.0",
    importpath = "github.com/google/go-cmp",
)

# Maven jars

load("//thirdparty:maven.bzl", "maven_dependencies")

maven_dependencies()

# gRPC for Java

http_archive(
    name = "grpc_java",
    sha256 = "b4c9839b672213686ee9633a6089a913e50b1040d9f7486bd09d17032a2a90d3",
    strip_prefix = "grpc-java-1.10.0",
    urls = [
        "https://github.com/grpc/grpc-java/archive/v1.10.0.zip",
    ],
)

# Bind Guava and GSON for @com_google_protobuf//:protobuf_java_util

bind(
    name = "guava",
    actual = "//thirdparty/jvm/com/google/guava",
)

bind(
    name = "gson",
    actual = "//thirdparty/jvm/com/google/code/gson",
)
