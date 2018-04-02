java_library(
    name = "grpc_context",
    exports = [
        "//external:jar/io/grpc/grpc_context"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "grpc_core",
    exports = [
        "//external:jar/io/grpc/grpc_core"
    ],
    runtime_deps = [
        "//thirdparty/jvm/com/google/code/findbugs:jsr305",
        "//thirdparty/jvm/com/google/code/gson:gson",
        "//thirdparty/jvm/com/google/errorprone:error_prone_annotations",
        "//thirdparty/jvm/com/google/guava:guava",
        "//thirdparty/jvm/io/opencensus:opencensus_api",
        "//thirdparty/jvm/io/opencensus:opencensus_contrib_grpc_metrics",
        ":grpc_context"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "grpc_netty",
    exports = [
        "//external:jar/io/grpc/grpc_netty"
    ],
    runtime_deps = [
        "//thirdparty/jvm/io/netty:netty_codec_http2",
        "//thirdparty/jvm/io/netty:netty_handler_proxy",
        ":grpc_core"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "grpc_protobuf",
    exports = [
        "//external:jar/io/grpc/grpc_protobuf"
    ],
    runtime_deps = [
        "//thirdparty/jvm/com/google/api/grpc:proto_google_common_protos",
        "//thirdparty/jvm/com/google/guava:guava",
        "//thirdparty/jvm/com/google/protobuf:protobuf_java",
        "//thirdparty/jvm/com/google/protobuf:protobuf_java_util",
        ":grpc_core",
        ":grpc_protobuf_lite"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "grpc_protobuf_lite",
    exports = [
        "//external:jar/io/grpc/grpc_protobuf_lite"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "grpc_services",
    exports = [
        "//external:jar/io/grpc/grpc_services"
    ],
    runtime_deps = [
        "//thirdparty/jvm/com/google/instrumentation:instrumentation_api",
        ":grpc_protobuf",
        ":grpc_stub"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "grpc_stub",
    exports = [
        "//external:jar/io/grpc/grpc_stub"
    ],
    runtime_deps = [
        ":grpc_core"
    ],
    visibility = [
        "//visibility:public"
    ]
)


