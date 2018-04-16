// Copyright 2018 The Jadep Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.google.devtools.javatools.jade.pkgloader;

import com.google.common.util.concurrent.ThreadFactoryBuilder;
import com.google.devtools.build.lib.vfs.JavaIoFileSystem;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.PackageLoaderGrpc.PackageLoaderImplBase;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.Empty;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.LoaderRequest;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.LoaderResponse;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.Services.VersionResponse;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.services.VersionManagementGrpc.VersionManagementImplBase;
import io.grpc.MethodDescriptor;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerMethodDefinition;
import io.grpc.ServerServiceDefinition;
import io.grpc.netty.NettyServerBuilder;
import io.grpc.protobuf.services.ProtoReflectionService;
import io.grpc.stub.StreamObserver;
import io.netty.channel.EventLoopGroup;
import io.netty.channel.epoll.EpollEventLoopGroup;
import io.netty.channel.epoll.EpollServerDomainSocketChannel;
import io.netty.channel.kqueue.KQueueEventLoopGroup;
import io.netty.channel.kqueue.KQueueServerDomainSocketChannel;
import io.netty.channel.unix.DomainSocketAddress;
import io.netty.util.NetUtil;
import io.netty.util.internal.MacAddressUtil;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.concurrent.ThreadFactory;
import java.util.logging.Level;
import java.util.logging.Logger;
import org.kohsuke.args4j.CmdLineException;
import org.kohsuke.args4j.CmdLineParser;
import org.kohsuke.args4j.Option;

/**
 * GrpcLocalServer allows clients to load Bazel packages without calling Bazel.
 *
 * <p>It is intended to be used locally, i.e. on the same machine that Jade runs on. It only
 * supports local paths.
 */
public class GrpcLocalServer {

  private static final Logger logger = Logger.getLogger("GrpcLocalServer");

  private static final com.google.devtools.build.lib.vfs.FileSystem FILESYSTEM =
      new JavaIoFileSystem();

  private static final String UNIX_DOMAIN_SOCKET_PREFIX = "unix://";

  private static final PackageLoaderFactory PACKAGE_LOADER_FACTORY =
      new BazelPackageLoaderFactory();

  @Option(
    name = "--bind",
    usage =
        "what to bind (listen) the server to. "
            + "If of the form 'unix://<filename>', will bind to a Unix domain socket at <filename>."
            + " Otherwise, must be a number (will bind to localhost:<number>)."
  )
  private static String bind = "";

  @Option(
    name = "--version",
    usage =
        "a user specified string to be returned from VersionManagement.Version. "
            + "Jade uses this flag to detect when a new server release has happened "
            + "and it should restart the server"
  )
  private static String version = "";

  private static class PackageLoaderImpl extends PackageLoaderImplBase {
    @Override
    public void load(LoaderRequest request, StreamObserver<LoaderResponse> responseObserver) {
      responseObserver.onNext(Lib.load(PACKAGE_LOADER_FACTORY, FILESYSTEM, request));
      responseObserver.onCompleted();
    }
  }

  private static class VersionManagementImpl extends VersionManagementImplBase {
    @Override
    public void version(Empty request, StreamObserver<VersionResponse> responseObserver) {
      responseObserver.onNext(VersionResponse.newBuilder().setVersion(version).build());
      responseObserver.onCompleted();
    }

    @Override
    public void shutdown(Empty request, StreamObserver<Empty> responseObserver) {
      logger.info("Shutting down in response to user request");
      System.exit(0);
    }
  }

  public static void main(String[] args) throws Exception {
    GrpcLocalServer server = new GrpcLocalServer();
    CmdLineParser parser = new CmdLineParser(server);
    try {
      parser.parseArgument(args);
      server.run();
    } catch (CmdLineException e) {
      System.err.println(e.getMessage());
      parser.printUsage(System.err);
    }
  }

  private void run() throws Exception {
    final ServerBuilder<?> serverBuilder;
    if (bind.startsWith(UNIX_DOMAIN_SOCKET_PREFIX)) {
      OperatingSystem os = OperatingSystem.detect();
      if (os != OperatingSystem.LINUX && os != OperatingSystem.MACOS) {
        logger.severe("binding to unix:// addresses is only supported on Linux and macOS");
        return;
      }
      serverBuilder = bindUDS(bind, os);
    } else {
      final int port;
      try {
        port = Integer.parseInt(bind);
      } catch (NumberFormatException e) {
        logger.severe("--bind parameter is neither prefixed with unix:// nor a number");
        return;
      }
      serverBuilder = bindTCP(port);
    }
    ServerServiceDefinition versionServiceDef = new VersionManagementImpl().bindService();
    ServerServiceDefinition loaderServiceDef = new PackageLoaderImpl().bindService();
    Server server =
        serverBuilder
            .addService(loaderServiceDef)
            .addService(versionServiceDef)
            .addService(renameService(loaderServiceDef))
            .addService(renameService(versionServiceDef))
            .addService(ProtoReflectionService.newInstance())
            .build();
    try {
      server.start();
    } catch (IOException e) {
      logger.log(Level.SEVERE, "Error starting server", e);
      System.exit(1);
    }
    logger.info("PackageLoader service started and listening");
    server.awaitTermination();
    logger.info("Reached end of main()");
    // If we reached here, server is no longer active. However, other threads might still be running
    // which prevents the program from exiting. We don't care about them, so explicitly exit.
    System.exit(0);
  }

  private ServerBuilder<?> bindUDS(String bind, OperatingSystem os) throws Exception {
    // Try to eagerly initialize these classes.
    // TODO: Remove once fix is available.
    try {
      Class.forName(NetUtil.class.getName());
      Class.forName(MacAddressUtil.class.getName());
    } catch (ClassNotFoundException e) {
      throw new IOException(e);
    }
    String uds = bind.substring(UNIX_DOMAIN_SOCKET_PREFIX.length());
    logger.info("Binding to UDS: " + uds);
    Path udsPath = Paths.get(uds);
    Files.deleteIfExists(udsPath);
    deleteOnExit(udsPath);
    ThreadFactory threadFactory = new ThreadFactoryBuilder().setDaemon(true).build();

    NettyServerBuilder result = NettyServerBuilder.forAddress(new DomainSocketAddress(uds));

    EventLoopGroup group;
    switch (os) {
      case LINUX:
        group = new EpollEventLoopGroup(1, threadFactory);
        result.channelType(EpollServerDomainSocketChannel.class);
        break;
      case MACOS:
        group = new KQueueEventLoopGroup(1, threadFactory);
        result.channelType(KQueueServerDomainSocketChannel.class);
        break;
      default:
        throw new IllegalStateException(
            "binding to unix:// addresses is only supported on Linux and macOS");
    }
    result.workerEventLoopGroup(group).bossEventLoopGroup(group);

    return result;
  }

  private ServerBuilder<?> bindTCP(int port) throws Exception {
    logger.info("Binding to port: " + port);
    return NettyServerBuilder.forAddress(new InetSocketAddress("localhost", port));
  }

  /**
   * renameService returns a copy of 'serviceDef', with all names renamed from
   * "java.com.google.devtools.javatools.jade.pkgloader.services." to
   * "java.com.google.devtools.javatools.jade.pkgloader."
   *
   * <p>The purpose is to allow clients to migrate to the new proto names (with "services.").
   */
  // TODO: Delete in a week.
  private ServerServiceDefinition renameService(ServerServiceDefinition serviceDef) {
    ServerServiceDefinition.Builder renamedServiceDef =
        ServerServiceDefinition.builder(
            serviceDef
                .getServiceDescriptor()
                .getName()
                .replace(
                    "java.com.google.devtools.javatools.jade.pkgloader.services.",
                    "java.com.google.devtools.javatools.jade.pkgloader."));
    for (ServerMethodDefinition<?, ?> method : serviceDef.getMethods()) {
      renamedServiceDef.addMethod(renameGrpcMethod(method));
    }
    return renamedServiceDef.build();
  }

  /** See renameService(). */
  private <ReqT, ResT> ServerMethodDefinition<ReqT, ResT> renameGrpcMethod(
      ServerMethodDefinition<ReqT, ResT> method) {
    MethodDescriptor<ReqT, ResT> descriptor = method.getMethodDescriptor();
    String newname =
        descriptor
            .getFullMethodName()
            .replace(
                "java.com.google.devtools.javatools.jade.pkgloader.services.",
                "java.com.google.devtools.javatools.jade.pkgloader.");
    return ServerMethodDefinition.create(
        descriptor.toBuilder().setFullMethodName(newname).build(), method.getServerCallHandler());
  }

  private static void deleteOnExit(Path path) {
    Runtime.getRuntime()
        .addShutdownHook(
            new Thread() {
              @Override
              public void run() {
                try {
                  Files.deleteIfExists(path);
                } catch (IOException e) {
                  // Do nothing - we're already in a shutdown hook.
                }
              }
            });
  }
}
