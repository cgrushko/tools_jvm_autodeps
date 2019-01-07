# Drop-in replacement for http_archive
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Bazel code, required for PackageLoader

http_archive(
    name = "io_bazel",
    sha256 = "23e4281c3628cbd746da3f51330109bbf69780bd64461b63b386efae37203f20",
    urls = ["https://releases.bazel.build/0.17.1/release/bazel-0.17.1-dist.zip"],
)

# Buildozer, to manipulate BUILD files

http_archive(
    name = "com_github_bazelbuild_buildtools",
    strip_prefix = "buildtools-0b76442a60b61abbff02239620b493f25d6d9867",
    type = "zip",
    urls = ["https://github.com/bazelbuild/buildtools/archive/0b76442a60b61abbff02239620b493f25d6d9867.zip"],
)

# Protobuf

http_archive(
    name = "com_google_protobuf",
    sha256 = "d6618d117698132dadf0f830b762315807dc424ba36ab9183f1f436008a2fdb6",
    strip_prefix = "protobuf-3.6.1.2",
    urls = ["https://github.com/google/protobuf/archive/v3.6.1.2.zip"],
)

# Go - bind external repos

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.16.5/rules_go-0.16.5.tar.gz",
    sha256 = "7be7dc01f1e0afdba6c8eb2b43d2fa01c743be1b9273ab1eaf6c233df078d705",
)
http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
    sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

# gRPC for Go

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",  # use existing generated code
    commit = "8e4536a86ab602859c20df5ebfd0bd4228d08655",  # v1.10.0
    importpath = "google.golang.org/grpc",
)

# Go - load additional repos.

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
