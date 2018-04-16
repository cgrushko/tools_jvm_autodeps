java_library(
    name = "netty_all",
    exports = [
        "//external:jar/io/netty/netty_all"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_buffer",
    exports = [
        "//external:jar/io/netty/netty_buffer"
    ],
    runtime_deps = [
        ":netty_common"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_codec",
    exports = [
        "//external:jar/io/netty/netty_codec"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_codec_http",
    exports = [
        "//external:jar/io/netty/netty_codec_http"
    ],
    runtime_deps = [
        ":netty_codec"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_codec_http2",
    exports = [
        "//external:jar/io/netty/netty_codec_http2"
    ],
    runtime_deps = [
        ":netty_codec_http",
        ":netty_handler"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_codec_socks",
    exports = [
        "//external:jar/io/netty/netty_codec_socks"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_common",
    exports = [
        "//external:jar/io/netty/netty_common"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_handler",
    exports = [
        "//external:jar/io/netty/netty_handler"
    ],
    runtime_deps = [
        ":netty_buffer"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_handler_proxy",
    exports = [
        "//external:jar/io/netty/netty_handler_proxy"
    ],
    runtime_deps = [
        ":netty_codec_socks",
        ":netty_transport"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_resolver",
    exports = [
        "//external:jar/io/netty/netty_resolver"
    ],
    visibility = [
        "//visibility:public"
    ]
)



java_library(
    name = "netty_transport",
    exports = [
        "//external:jar/io/netty/netty_transport"
    ],
    runtime_deps = [
        ":netty_resolver"
    ],
    visibility = [
        "//visibility:public"
    ]
)


