go_test(
    name = "go_default_test",
    srcs = ["grpcloader_test.go"],
    args = ["--pkg_loader_executable=$(location //java/com/google/devtools/javatools/jade/pkgloader:GrpcLocalServer)"],
    data = [
        "//java/com/google/devtools/javatools/jade/pkgloader:GrpcLocalServer",
    ],
    embed = [":go_default_library"],
    deps = [
        # Filled by Gazelle.
    ],
)
