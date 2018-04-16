# Do not edit. bazel-deps autogenerates this file from maven_deps.yaml.

def declare_maven(hash):
    native.maven_jar(
        name = hash["name"],
        artifact = hash["artifact"],
        sha1 = hash["sha1"],
        repository = hash["repository"]
    )
    native.bind(
        name = hash["bind"],
        actual = hash["actual"]
    )

def maven_dependencies(callback = declare_maven):
    callback({"artifact": "args4j:args4j:2.33", "lang": "java", "sha1": "bd87a75374a6d6523de82fef51fc3cfe9baf9fc9", "repository": "https://repo.maven.apache.org/maven2/", "name": "args4j_args4j", "actual": "@args4j_args4j//jar", "bind": "jar/args4j/args4j"})
    callback({"artifact": "com.google.api.grpc:proto-google-common-protos:1.0.0", "lang": "java", "sha1": "86f070507e28b930e50d218ee5b6788ef0dd05e6", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_api_grpc_proto_google_common_protos", "actual": "@com_google_api_grpc_proto_google_common_protos//jar", "bind": "jar/com/google/api/grpc/proto_google_common_protos"})
# duplicates in com.google.code.findbugs:jsr305 fixed to 3.0.2
# - com.google.guava:guava:23.0 wanted version 1.3.9
# - io.grpc:grpc-core:1.10.0 wanted version 3.0.0
# - com.google.instrumentation:instrumentation-api:0.4.3 wanted version 3.0.0
    callback({"artifact": "com.google.code.findbugs:jsr305:3.0.2", "lang": "java", "sha1": "25ea2e8b0c338a877313bd4672d3fe056ea78f0d", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_code_findbugs_jsr305", "actual": "@com_google_code_findbugs_jsr305//jar", "bind": "jar/com/google/code/findbugs/jsr305"})
    callback({"artifact": "com.google.code.gson:gson:2.8.2", "lang": "java", "sha1": "3edcfe49d2c6053a70a2a47e4e1c2f94998a49cf", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_code_gson_gson", "actual": "@com_google_code_gson_gson//jar", "bind": "jar/com/google/code/gson/gson"})
# duplicates in com.google.errorprone:error_prone_annotations promoted to 2.1.2
# - com.google.guava:guava:23.0 wanted version 2.0.18
# - io.grpc:grpc-core:1.10.0 wanted version 2.1.2
# - com.google.truth:truth:0.35 wanted version 2.0.19
    callback({"artifact": "com.google.errorprone:error_prone_annotations:2.1.2", "lang": "java", "sha1": "6dcc08f90f678ac33e5ef78c3c752b6f59e63e0c", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_errorprone_error_prone_annotations", "actual": "@com_google_errorprone_error_prone_annotations//jar", "bind": "jar/com/google/errorprone/error_prone_annotations"})
# duplicates in com.google.guava:guava fixed to 23.0
# - io.grpc:grpc-core:1.10.0 wanted version 19.0
# - io.grpc:grpc-protobuf:1.10.0 wanted version 19.0
# - com.google.truth:truth:0.35 wanted version 22.0-android
    callback({"artifact": "com.google.guava:guava:23.0", "lang": "java", "sha1": "c947004bb13d18182be60077ade044099e4f26f1", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_guava_guava", "actual": "@com_google_guava_guava//jar", "bind": "jar/com/google/guava/guava"})
    callback({"artifact": "com.google.instrumentation:instrumentation-api:0.4.3", "lang": "java", "sha1": "41614af3429573dc02645d541638929d877945a2", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_instrumentation_instrumentation_api", "actual": "@com_google_instrumentation_instrumentation_api//jar", "bind": "jar/com/google/instrumentation/instrumentation_api"})
    callback({"artifact": "com.google.j2objc:j2objc-annotations:1.1", "lang": "java", "sha1": "ed28ded51a8b1c6b112568def5f4b455e6809019", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_j2objc_j2objc_annotations", "actual": "@com_google_j2objc_j2objc_annotations//jar", "bind": "jar/com/google/j2objc/j2objc_annotations"})
    callback({"artifact": "com.google.protobuf:protobuf-java-util:3.5.1", "lang": "java", "sha1": "6e40a6a3f52455bd633aa2a0dba1a416e62b4575", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_protobuf_protobuf_java_util", "actual": "@com_google_protobuf_protobuf_java_util//jar", "bind": "jar/com/google/protobuf/protobuf_java_util"})
    callback({"artifact": "com.google.protobuf:protobuf-java:3.5.1", "lang": "java", "sha1": "8c3492f7662fa1cbf8ca76a0f5eb1146f7725acd", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_protobuf_protobuf_java", "actual": "@com_google_protobuf_protobuf_java//jar", "bind": "jar/com/google/protobuf/protobuf_java"})
    callback({"artifact": "com.google.truth:truth:0.35", "lang": "java", "sha1": "c08a7fde45e058323bcfa3f510d4fe1e2b028f37", "repository": "https://repo.maven.apache.org/maven2/", "name": "com_google_truth_truth", "actual": "@com_google_truth_truth//jar", "bind": "jar/com/google/truth/truth"})
    callback({"artifact": "io.grpc:grpc-context:1.10.0", "lang": "java", "sha1": "da0a701be6ba04aff0bd54ca3db8248d8f2eaafc", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_context", "actual": "@io_grpc_grpc_context//jar", "bind": "jar/io/grpc/grpc_context"})
    callback({"artifact": "io.grpc:grpc-core:1.10.0", "lang": "java", "sha1": "8976afebf2a6530574a71bc1260920ce910c2292", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_core", "actual": "@io_grpc_grpc_core//jar", "bind": "jar/io/grpc/grpc_core"})
    callback({"artifact": "io.grpc:grpc-netty:1.10.0", "lang": "java", "sha1": "a1056d69003c9b46d1c4aa4a9167c6e8a714d152", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_netty", "actual": "@io_grpc_grpc_netty//jar", "bind": "jar/io/grpc/grpc_netty"})
    callback({"artifact": "io.grpc:grpc-protobuf-lite:1.10.0", "lang": "java", "sha1": "b8e40dd308dc370e64bd2c337bb2761a03299a7f", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_protobuf_lite", "actual": "@io_grpc_grpc_protobuf_lite//jar", "bind": "jar/io/grpc/grpc_protobuf_lite"})
    callback({"artifact": "io.grpc:grpc-protobuf:1.10.0", "lang": "java", "sha1": "64098f046f227b47238bc747e3cee6c7fc087bb8", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_protobuf", "actual": "@io_grpc_grpc_protobuf//jar", "bind": "jar/io/grpc/grpc_protobuf"})
    callback({"artifact": "io.grpc:grpc-services:1.10.0", "lang": "java", "sha1": "ae898f12418429c9d1396aaf5ac2377bf8ecb25b", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_services", "actual": "@io_grpc_grpc_services//jar", "bind": "jar/io/grpc/grpc_services"})
    callback({"artifact": "io.grpc:grpc-stub:1.10.0", "lang": "java", "sha1": "d022706796b0820d388f83571da160fb8d280ded", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_grpc_grpc_stub", "actual": "@io_grpc_grpc_stub//jar", "bind": "jar/io/grpc/grpc_stub"})
    callback({"artifact": "io.netty:netty-all:4.1.22.Final", "lang": "java", "sha1": "c1cea5d30025e4d584d2b287d177c31aea4ae629", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_all", "actual": "@io_netty_netty_all//jar", "bind": "jar/io/netty/netty_all"})
    callback({"artifact": "io.netty:netty-buffer:4.1.17.Final", "lang": "java", "sha1": "fdd68fb3defd7059a7392b9395ee941ef9bacc25", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_buffer", "actual": "@io_netty_netty_buffer//jar", "bind": "jar/io/netty/netty_buffer"})
    callback({"artifact": "io.netty:netty-codec-http2:4.1.17.Final", "lang": "java", "sha1": "f9844005869c6d9049f4b677228a89fee4c6eab3", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_codec_http2", "actual": "@io_netty_netty_codec_http2//jar", "bind": "jar/io/netty/netty_codec_http2"})
    callback({"artifact": "io.netty:netty-codec-http:4.1.17.Final", "lang": "java", "sha1": "251d7edcb897122b9b23f24ff793cd0739056b9e", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_codec_http", "actual": "@io_netty_netty_codec_http//jar", "bind": "jar/io/netty/netty_codec_http"})
    callback({"artifact": "io.netty:netty-codec-socks:4.1.17.Final", "lang": "java", "sha1": "a159bf1f3d5019e0d561c92fbbec8400967471fa", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_codec_socks", "actual": "@io_netty_netty_codec_socks//jar", "bind": "jar/io/netty/netty_codec_socks"})
    callback({"artifact": "io.netty:netty-codec:4.1.17.Final", "lang": "java", "sha1": "1d00f56dc9e55203a4bde5aae3d0828fdeb818e7", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_codec", "actual": "@io_netty_netty_codec//jar", "bind": "jar/io/netty/netty_codec"})
    callback({"artifact": "io.netty:netty-common:4.1.17.Final", "lang": "java", "sha1": "581c8ee239e4dc0976c2405d155f475538325098", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_common", "actual": "@io_netty_netty_common//jar", "bind": "jar/io/netty/netty_common"})
    callback({"artifact": "io.netty:netty-handler-proxy:4.1.17.Final", "lang": "java", "sha1": "9330ee60c4e48ca60aac89b7bc5ec2567e84f28e", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_handler_proxy", "actual": "@io_netty_netty_handler_proxy//jar", "bind": "jar/io/netty/netty_handler_proxy"})
    callback({"artifact": "io.netty:netty-handler:4.1.17.Final", "lang": "java", "sha1": "18c40ffb61a1d1979eca024087070762fdc4664a", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_handler", "actual": "@io_netty_netty_handler//jar", "bind": "jar/io/netty/netty_handler"})
    callback({"artifact": "io.netty:netty-resolver:4.1.17.Final", "lang": "java", "sha1": "8f386c80821e200f542da282ae1d3cde5cad8368", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_resolver", "actual": "@io_netty_netty_resolver//jar", "bind": "jar/io/netty/netty_resolver"})
    callback({"artifact": "io.netty:netty-transport:4.1.17.Final", "lang": "java", "sha1": "9585776b0a8153182412b5d5366061ff486914c1", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_netty_netty_transport", "actual": "@io_netty_netty_transport//jar", "bind": "jar/io/netty/netty_transport"})
    callback({"artifact": "io.opencensus:opencensus-api:0.11.0", "lang": "java", "sha1": "c1ff1f0d737a689d900a3e2113ddc29847188c64", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_opencensus_opencensus_api", "actual": "@io_opencensus_opencensus_api//jar", "bind": "jar/io/opencensus/opencensus_api"})
    callback({"artifact": "io.opencensus:opencensus-contrib-grpc-metrics:0.11.0", "lang": "java", "sha1": "d57b877f1a28a613452d45e35c7faae5af585258", "repository": "https://repo.maven.apache.org/maven2/", "name": "io_opencensus_opencensus_contrib_grpc_metrics", "actual": "@io_opencensus_opencensus_contrib_grpc_metrics//jar", "bind": "jar/io/opencensus/opencensus_contrib_grpc_metrics"})
    callback({"artifact": "junit:junit:4.12", "lang": "java", "sha1": "2973d150c0dc1fefe998f834810d68f278ea58ec", "repository": "https://repo.maven.apache.org/maven2/", "name": "junit_junit", "actual": "@junit_junit//jar", "bind": "jar/junit/junit"})
    callback({"artifact": "org.codehaus.mojo:animal-sniffer-annotations:1.14", "lang": "java", "sha1": "775b7e22fb10026eed3f86e8dc556dfafe35f2d5", "repository": "https://repo.maven.apache.org/maven2/", "name": "org_codehaus_mojo_animal_sniffer_annotations", "actual": "@org_codehaus_mojo_animal_sniffer_annotations//jar", "bind": "jar/org/codehaus/mojo/animal_sniffer_annotations"})
    callback({"artifact": "org.hamcrest:hamcrest-core:1.3", "lang": "java", "sha1": "42a25dc3219429f0e5d060061f71acb49bf010a0", "repository": "https://repo.maven.apache.org/maven2/", "name": "org_hamcrest_hamcrest_core", "actual": "@org_hamcrest_hamcrest_core//jar", "bind": "jar/org/hamcrest/hamcrest_core"})
    callback({"artifact": "org.textmapper:lapg:0.9.18", "lang": "java", "sha1": "9d480589d5770d75c4401f38c3cfd22a7139a397", "repository": "https://repo.maven.apache.org/maven2/", "name": "org_textmapper_lapg", "actual": "@org_textmapper_lapg//jar", "bind": "jar/org/textmapper/lapg"})
    callback({"artifact": "org.textmapper:templates:0.9.18", "lang": "java", "sha1": "1979db4fe5d0581639d3ace891a7abeaf95f8220", "repository": "https://repo.maven.apache.org/maven2/", "name": "org_textmapper_templates", "actual": "@org_textmapper_templates//jar", "bind": "jar/org/textmapper/templates"})
    callback({"artifact": "org.textmapper:textmapper:0.9.18", "lang": "java", "sha1": "80ffa6ce9f7f3250fcc62419c0898ffeedbd5902", "repository": "https://repo.maven.apache.org/maven2/", "name": "org_textmapper_textmapper", "actual": "@org_textmapper_textmapper//jar", "bind": "jar/org/textmapper/textmapper"})
