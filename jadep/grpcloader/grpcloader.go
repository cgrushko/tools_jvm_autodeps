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

// Package grpcloader implements a Loader that connects to a gRPC server.
package grpcloader

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"context"
	"github.com/golang/protobuf/proto"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/jadep/vlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	mpb "github.com/bazelbuild/tools_jvm_autodeps/jadep/java/com/google/devtools/javatools/jade/pkgloader/messages_proto"
	sgrpc "github.com/bazelbuild/tools_jvm_autodeps/jadep/java/com/google/devtools/javatools/jade/pkgloader/services_proto"
	spb "github.com/bazelbuild/tools_jvm_autodeps/jadep/java/com/google/devtools/javatools/jade/pkgloader/services_proto"
)

// udsDialerOpt instructs gRPC to dial to a Unix domain socket.
var udsDialerOpt = grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
})

// Loader calls a gRPC PkgLoader service to interpret BUILD files.
type Loader struct {
	stub                 sgrpc.PackageLoaderClient
	timeout              time.Duration
	workspaceRoot        string
	bazelInstallBase     string
	bazelOutputBase      string
	ruleKindsToSerialize []string
}

// NewLoader creates a new Loader that sends RPCs on 'conn' with 'timeout'.
// 'workspaceRoot' is a root Bazel directory, i.e. contains a WORKSPACE file.
// bazelInstallBase and bazelOutputBase are Bazel's current install base and output base. They can be acquired by calling `bazel info`.
// 'ruleKindsToSerialize' are the rule kinds to send back from the server; leave empty to get all.
func NewLoader(stub sgrpc.PackageLoaderClient, timeout time.Duration, workspaceRoot, bazelInstallBase, bazelOutputBase string, ruleKindsToSerialize []string) *Loader {
	return &Loader{
		stub:                 stub,
		timeout:              timeout,
		workspaceRoot:        workspaceRoot,
		bazelInstallBase:     bazelInstallBase,
		bazelOutputBase:      bazelOutputBase,
		ruleKindsToSerialize: ruleKindsToSerialize,
	}
}

// Connect connects to a PackageLoader gRPC server at 'addr', starting a server it if there isn't one, and restarting an existing one if it's out-of-date.
//
// If 'addr' is of the form "unix://<file name>", it tries to connect to a service listening on the Unix domain socket <file name>.
// If 'addr' is of the form "localhost:<port>", it tries to connect to that address.
// Otherwise, addr is used as is in a call to grpc.Dial.
// In the first two cases, if the call attempt times out, StartAndConnext starts 'executable',
// passing it --bind=unix://<filename> in the first case, or --bind=<port> in the second case.
//
// Server freshness is implemented as following.
// 1. On server startup, Jade sets the server's --version flag to the server's executable mtime.
// 2. On each connection, Jade uses the VersionManagement service to query the server's version.
// 3. If the response doesn't match the server-executable's current mtime, it means that the server's executable has been updated, and Jade should restart the server.
// 4. VersionManagement.Shutdown is used to tell the server to shut itself down.
// 5. Jade starts a new server (setting its --version, etc.).
//
// Connect returns a Loader and a cleanup function.
//
// 'workspaceRoot' is a root Bazel directory, i.e. contains a WORKSPACE file.
// 'ruleKindsToSerialize' are the rule kinds to send back from the server; leave empty to get all.
func Connect(ctx context.Context, executable, addr string, timeout time.Duration, workspaceRoot, bazelInstallBase, bazelOutputBase string, ruleKindsToSerialize []string) (*Loader, func(), error) {
	conn, proc, err := dialAndStart(ctx, executable, addr, timeout)
	if err != nil {
		return nil, nil, err
	}
	if proc != nil {
		proc.Release()
	}
	return NewLoader(sgrpc.NewPackageLoaderClient(conn), timeout, workspaceRoot, bazelInstallBase, bazelOutputBase, ruleKindsToSerialize), func() { conn.Close() }, nil
}

// dialAndStart attempts to connect to 'bindLocation'.
// If it fails, it starts 'executable' and attempts to connect to it for 'connectionTimeout' duration.
func dialAndStart(ctx context.Context, executable, bindLocation string, connectionTimeout time.Duration) (*grpc.ClientConn, *os.Process, error) {
	log.Printf("Connecting to gRPC server at %s", bindLocation)

	callOpts := []grpc.CallOption{grpc.MaxCallRecvMsgSize(100 * 1 << 20)}
	dialOpts := []grpc.DialOption{grpc.WithTimeout(time.Second), grpc.WithBlock(), grpc.WithInsecure(), grpc.WithDefaultCallOptions(callOpts...)}
	dialAddr, bindParam, typ := dialAddr(bindLocation)
	if typ == uds {
		dialOpts = append(dialOpts, udsDialerOpt)
	}

	conn, err := grpc.Dial(dialAddr, dialOpts...)

	if err == context.DeadlineExceeded && typ != unknown {
		// Couldn't connect to a local service, start it.
		conn, proc, err := startServer(ctx, executable, bindParam, dialAddr, dialOpts, connectionTimeout)
		if err == nil {
			return conn, proc, nil
		}
		return nil, nil, fmt.Errorf("error starting gRPC server:\n%v", err)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("error connecting to gRPC server:\n%v", err)
	}

	if typ == unknown {
		// Succesfully connected to a remote service.
		return conn, nil, nil
	}
	// Succesfully connected to a local service, check if it has the right version.
	killed, err := killServerIfOld(ctx, executable, conn, connectionTimeout)
	if err != nil {
		return nil, nil, fmt.Errorf("error killing stale gRPC server:\n%v", err)
	}
	var proc *os.Process
	if killed {
		conn, proc, err = startServer(ctx, executable, bindParam, dialAddr, dialOpts, connectionTimeout)
		if err != nil {
			return nil, nil, fmt.Errorf("error starting gRPC server:\n%v", err)
		}
	}
	return conn, proc, nil
}

// Return values for dialAddr.
const (
	uds = iota
	localhost
	unknown
)

// dialAddr returns the dial address corresponding to bindLocation, the --param that should be passed to a gRPC server start, and the type of the location.
// It returns (<filename>, unix://<filename>, uds) when bindLocation is of the form unix://<filename>.
// It returns (<localhost:<number>, number, localhost) when bindLocation is of the form localhost:<number>.
// Otherwise, it returns (bindLocation, bindLocation, unknown)
func dialAddr(bindLocation string) (addr string, bindParam string, typ int) {
	if strings.HasPrefix(bindLocation, "unix://") {
		return bindLocation[len("unix://"):], bindLocation, uds
	} else if strings.HasPrefix(bindLocation, "localhost:") {
		return bindLocation, bindLocation[len("localhost:"):], localhost
	} else {
		return bindLocation, bindLocation, unknown
	}
}

// startServer starts 'executable' and connects to it.
// executable is assumed to point at a GrpcLocalServer_deploy.jar.
func startServer(ctx context.Context, executable, bindParam string, dialAddr string, dialOpts []grpc.DialOption, connectionTimeout time.Duration) (*grpc.ClientConn, *os.Process, error) {
	log.Printf("No gRPC server found, starting one.")
	mtime, err := modTime(executable)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, executable, "--bind="+bindParam, fmt.Sprintf("--version=%d", mtime))
	vlog.V(2).Printf("Starting gRPC server: %s --bind=%s --version=%d", executable, bindParam, mtime)
	buffer := &closeableBuffer{}
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	defer buffer.close()
	conn, err := attemptDial(dialAddr, dialOpts, connectionTimeout)
	if err != nil {
		return nil, nil, fmt.Errorf("can't connect to the started PackageLoader server: %v\nServer's stdout+stderr:\n%s", err, buffer.buf.String())
	}

	return conn, cmd.Process, nil
}

// attemptDial attempts to connect to 'dialAddr' for 'connectionTimeout' duration.
func attemptDial(dialAddr string, dialOpts []grpc.DialOption, connectionTimeout time.Duration) (*grpc.ClientConn, error) {
	stopwatch := time.Now()
	for time.Now().Sub(stopwatch) < connectionTimeout {
		conn, err := grpc.Dial(dialAddr, dialOpts...)
		if err == nil {
			return conn, nil
		}
		if err != context.DeadlineExceeded {
			return nil, fmt.Errorf("error connecting to gRPC server:\n%v", err)
		}
	}
	return nil, fmt.Errorf("timeout (%v) while connecting to %s", connectionTimeout, dialAddr)
}

// modTime returns the modification time (mtime) of 'fileName', in Unix time (seconds since epoch).
func modTime(fileName string) (int64, error) {
	info, err := os.Stat(fileName)
	if err != nil {
		return 0, fmt.Errorf("error getting the modification time of %s:\n%v", fileName, err)
	}
	return info.ModTime().Unix(), nil
}

// killServerIfOld kills the server (through sgrpc.VersionManagement) if the server at 'conn' has a different mtime than that of 'executable'.
// It returns true if it killed the server.
func killServerIfOld(ctx context.Context, executable string, conn *grpc.ClientConn, timeout time.Duration) (bool, error) {
	mtime, err := modTime(executable)
	if err != nil {
		return false, err
	}
	client := sgrpc.NewVersionManagementClient(conn)
	version, err := client.Version(ctx, &spb.Empty{})
	if err != nil {
		return false, err
	}
	if *version.Version == strconv.FormatInt(mtime, 10) {
		return false, nil
	}
	log.Printf("Currently running gRPC server is stale, restarting it")

	_, err = client.Shutdown(ctx, &spb.Empty{})
	code := status.Code(err)
	// client.Shutdown forcibly shuts down the server before answering the shutdown request, so codes.Unavailable is the expected result.
	if code == codes.Unavailable {
		return true, nil
	}
	return false, fmt.Errorf("sending 'shutdown' command to server failed:\n%v", err)
}

// closeableBuffer is a bytes.Buffer that can be closed. Once closed, its Write method is a no-op.
type closeableBuffer struct {
	buf bytes.Buffer

	mu       sync.RWMutex
	disabled bool
}

func (b *closeableBuffer) close() {
	b.mu.Lock()
	b.disabled = true
	b.mu.Unlock()
}

func (b *closeableBuffer) Write(p []byte) (n int, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.disabled {
		return len(p), nil
	}
	return b.buf.Write(p)
}

// Load sends an RPC to a PkgLoader service, requesting it to interpret 'packages' (e.g., "foo/bar" to interpret <root>/foo/bar/BUILD)
func (r *Loader) Load(ctx context.Context, packages []string) (map[string]*bazel.Package, error) {
	req := spb.LoaderRequest{
		WorkspaceDir:         &r.workspaceRoot,
		InstallBase:          &r.bazelInstallBase,
		OutputBase:           &r.bazelOutputBase,
		Packages:             packages,
		RuleKindsToSerialize: r.ruleKindsToSerialize,
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	stopwatch := time.Now()
	reply, err := r.stub.Load(ctx, &req)
	if vlog.V(2) {
		log.Printf("Loading packages took %dms. Request:\n%q", int64(time.Now().Sub(stopwatch)/time.Millisecond), proto.CompactTextString(&req))
	}
	if err != nil {
		return nil, err
	}

	return DeserializeProto(reply), nil
}

// DeserializeProto deserializes a response from a PackageLoader gRPC service.
func DeserializeProto(proto *spb.LoaderResponse) map[string]*bazel.Package {
	result := make(map[string]*bazel.Package)
	for pkgName, protoPkg := range proto.Pkgs {
		var defaultVisibility []bazel.Label
		for _, l := range protoPkg.DefaultVisibility {
			defaultVisibility = append(defaultVisibility, bazel.Label(l))
		}

		rules := make(map[string]*bazel.Rule)
		for ruleName, protoRule := range protoPkg.Rules {
			attrs := make(map[string]interface{})
			for attrName, protoAttr := range protoRule.Attributes {
				switch x := protoAttr.Value.(type) {
				case *mpb.Attribute_S:
					attrs[attrName] = x.S
				case *mpb.Attribute_I:
					attrs[attrName] = x.I
				case *mpb.Attribute_B:
					attrs[attrName] = x.B
				case *mpb.Attribute_ListOfStrings:
					attrs[attrName] = x.ListOfStrings.Str
				case *mpb.Attribute_Unknown:
					attrs[attrName] = bazel.UnknownAttributeValue{}
				}
			}
			rules[ruleName] = &bazel.Rule{
				Schema:  *protoRule.Kind,
				PkgName: pkgName,
				Attrs:   attrs,
			}
		}

		var packageGroups map[string]*bazel.PackageGroup
		if len(protoPkg.PackageGroups) > 0 {
			packageGroups = make(map[string]*bazel.PackageGroup)
			for grpName, grpProto := range protoPkg.PackageGroups {
				var includes []bazel.Label
				for _, inc := range grpProto.Includes {
					includes = append(includes, bazel.Label(inc))
				}
				packageGroups[grpName] = &bazel.PackageGroup{grpProto.PackageSpecs, includes}
			}
		}

		result[pkgName] = &bazel.Package{
			DefaultVisibility: defaultVisibility,
			Files:             protoPkg.Files,
			PackageGroups:     packageGroups,
			Rules:             rules,
		}
	}
	return result
}
