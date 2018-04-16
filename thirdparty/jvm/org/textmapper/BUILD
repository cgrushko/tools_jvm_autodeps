java_library(
    name = "lapg",
    exports = [
        "//external:jar/org/textmapper/lapg"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "templates",
    exports = [
        "//external:jar/org/textmapper/templates"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "textmapper",
    exports = [
        "//external:jar/org/textmapper/textmapper"
    ],
    runtime_deps = [
        ":lapg",
        ":templates"
    ],
    visibility = [
        "//visibility:public"
    ]
)


