load("@io_bazel_rules_go//go:def.bzl", "go_library")

java_binary(
    name = "textmapper_tool",
    main_class = "org.textmapper.tool.Tool",
    runtime_deps = ["//thirdparty/jvm/org/textmapper"],
)

genrule(
    name = "syntax",
    srcs = [
        "java.tm",
        "go_filter.ltp",
        "go_listener.ltp",
        "go_parser.ltp",
    ],
    outs = [
        "lexer.go",
        "lexer_tables.go",
        "listener.go",
        "parser.go",
        "parser_tables.go",
        "token.go",
    ],
    cmd =
        "$(location :textmapper_tool)" +
        "  $(location :java.tm) -o $(@D) -i thirdparty/golang/parsers/java",
    tools = [":textmapper_tool"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "gen.go",
        "init.go",
        "lexer.go",
        "lexer_tables.go",
        "listener.go",
        "parser.go",
        "parser_tables.go",
        "token.go",
    ],
    importpath = "github.com/bazelbuild/tools_java_autodeps/thirdparty/golang/parsers/java",
    visibility = ["//visibility:public"],
    deps = [
        "//thirdparty/golang/parsers/lang:go_default_library",
        "//thirdparty/golang/parsers/node:go_default_library",
        "//thirdparty/golang/parsers/parsers:go_default_library",
        "//thirdparty/golang/parsers/public:go_default_library",
    ],
)
