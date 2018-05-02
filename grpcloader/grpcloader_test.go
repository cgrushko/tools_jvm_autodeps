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

package grpcloader

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"context"

	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/compat"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"

	sgrpc "github.com/bazelbuild/tools_jvm_autodeps/java/com/google/devtools/javatools/jade/pkgloader/services_proto"
	spb "github.com/bazelbuild/tools_jvm_autodeps/java/com/google/devtools/javatools/jade/pkgloader/services_proto"
)

var (
	pkgLoaderExecutable = flag.String("pkg_loader_executable", "", "path to a package loader server executable. Started when Jade fails to connect to --pkg_loader_bind_location")
	rpcTimeout          = flag.Duration("rpc_timeout", 15*time.Second, "RPC timeout (-record_rpc only)")
)

var pkgLoaderClient sgrpc.PackageLoaderClient

func TestMain(m *testing.M) {
	flag.Parse()
	bindLocation := "unix://" + tempFileName("grpc_binding_location")
	executable := compat.RunfilesPath(*pkgLoaderExecutable)
	conn, process, err := dialAndStart(context.Background(), executable, bindLocation, 30*time.Second)
	if err != nil {
		log.Fatalf("Error starting and dialing to gRPC PackageLoader server:\n%v", err)
	}
	defer conn.Close()
	defer process.Kill()
	pkgLoaderClient = sgrpc.NewPackageLoaderClient(conn)
	os.Exit(m.Run())
}

func TestLoad(t *testing.T) {
	type Attrs = map[string]interface{}

	workspaceRoot, installBase, outputBase, err := createWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(pkgLoaderClient, *rpcTimeout, workspaceRoot, installBase, outputBase, nil)

	// Run tests
	var tests = []struct {
		desc       string
		buildFiles []buildFile
		pkgsToLoad []string
		want       map[string]*bazel.Package
	}{
		{
			"basic e2e",
			[]buildFile{
				{"java/c", `java_library(name = 'Foo', srcs = [':Foo.java',':Bazel.java'], deps = [":G"])
java_library(name = 'G', srcs = [':Hello.java',':Jadep.java'])`},
				{"java/d", `java_library(name = 'Foo', srcs = [':Foo.java',':Bazel.java'], exports = ["//java/c:Foo"])
java_library(name = 'G', srcs = [':Hello.java',':Jadep.java'])`},
			},
			[]string{"java/c", "java/d"},
			map[string]*bazel.Package{
				"java/c": {
					Path:              filepath.Join(workspaceRoot, "java/c"),
					DefaultVisibility: []bazel.Label{"//visibility:private"},
					Files:             map[string]string{"BUILD": "", "Bazel.java": "", "Foo.java": "", "Hello.java": "", "Jadep.java": ""},
					Rules: map[string]*bazel.Rule{
						"Foo": {"java_library", "java/c", Attrs{"name": "Foo", "srcs": []string{"Bazel.java", "Foo.java"}, "deps": []string{"G"}, "testonly": false, "visibility": []string{"//visibility:private"}}},
						"G":   {"java_library", "java/c", Attrs{"name": "G", "srcs": []string{"Hello.java", "Jadep.java"}, "testonly": false, "visibility": []string{"//visibility:private"}}},
					},
				},
				"java/d": {
					Path:              filepath.Join(workspaceRoot, "java/d"),
					DefaultVisibility: []bazel.Label{"//visibility:private"},
					Files:             map[string]string{"BUILD": "", "Bazel.java": "", "Foo.java": "", "Hello.java": "", "Jadep.java": ""},
					Rules: map[string]*bazel.Rule{
						"Foo": {"java_library", "java/d", Attrs{"name": "Foo", "srcs": []string{"Bazel.java", "Foo.java"}, "exports": []string{"//java/c:Foo"}, "testonly": false, "visibility": []string{"//visibility:private"}}},
						"G":   {"java_library", "java/d", Attrs{"name": "G", "srcs": []string{"Hello.java", "Jadep.java"}, "testonly": false, "visibility": []string{"//visibility:private"}}},
					},
				},
			},
		},
		{
			"label-list selectors are flattened",
			[]buildFile{
				{"x", "proto_library(name = 'Foo', srcs = ['Bar'] + select({':cond1': ['a.proto'], ':cond2': ['b.proto']}))"},
			},
			[]string{"x"},
			map[string]*bazel.Package{
				"x": {
					Path:              filepath.Join(workspaceRoot, "x"),
					DefaultVisibility: []bazel.Label{"//visibility:private"},
					Files:             map[string]string{"Bar": "", "cond1": "", "cond2": "", "a.proto": "", "b.proto": "", "BUILD": ""},
					Rules:             map[string]*bazel.Rule{"Foo": {"proto_library", "x", Attrs{"name": "Foo", "srcs": []string{"Bar", "a.proto", "b.proto"}, "testonly": false, "visibility": []string{"//visibility:private"}}}},
				},
			},
		},
		{
			"the default clause of a scalar selectors is returned",
			[]buildFile{
				{"x", "java_binary(name = 'Foo', stamp = select({'//conditions:default': -1, ':cond1': 1}))"},
			},
			[]string{"x"},
			map[string]*bazel.Package{
				"x": {
					Path:              filepath.Join(workspaceRoot, "x"),
					DefaultVisibility: []bazel.Label{"//visibility:private"},
					Files:             map[string]string{"cond1": "", "BUILD": ""},
					Rules:             map[string]*bazel.Rule{"Foo": {"java_binary", "x", Attrs{"name": "Foo", "stamp": int32(-1), "testonly": false, "visibility": []string{"//visibility:private"}}}},
				},
			},
		},
		{
			"empty label lists show up in Rule.Attrs as nil.",
			[]buildFile{
				{"x", "proto_library(name = 'Foo', srcs = [])"},
			},
			[]string{"x"},
			map[string]*bazel.Package{
				"x": {
					Path:              filepath.Join(workspaceRoot, "x"),
					DefaultVisibility: []bazel.Label{"//visibility:private"},
					Files:             map[string]string{"BUILD": ""},
					Rules:             map[string]*bazel.Rule{"Foo": {"proto_library", "x", Attrs{"name": "Foo", "srcs": []string(nil), "testonly": false, "visibility": []string{"//visibility:private"}}}},
				},
			},
		},
	}
	for _, tt := range tests {
		func() {
			cleanup, err := createBuildFileDir(tt.buildFiles, workspaceRoot)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()

			got, err := loader.Load(context.Background(), tt.pkgsToLoad)
			if err != nil {
				t.Fatal(err)
			}
			filterOutGeneratedFiles(got)

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("%s: Load: (-got +want).\n%s", tt.desc, diff)
			}
		}()
	}
}

// filterOutGeneratedFiles removes any entry in the files of a package that has a non-empty owner.
func filterOutGeneratedFiles(pkgs map[string]*bazel.Package) {
	for _, pkg := range pkgs {
		for name, owner := range pkg.Files {
			if owner != "" {
				delete(pkg.Files, name)
			}
		}
	}
}

// TestDialAndStartUds asserts that calling dialAndStart() first starts a server and connects to it, and subsequently immediately connects.
// In this test, the server binds to a Unix domain socket.
func TestDialAndStartUds(t *testing.T) {
	bindLocation := "unix://" + tempFileName("grpc_binding_location")
	executable := compat.RunfilesPath(*pkgLoaderExecutable)

	// Running for the first time: process should be non-nil.
	{
		conn, process, err := dialAndStart(context.Background(), executable, bindLocation, 30*time.Second)
		defer conn.Close()
		defer process.Kill()
		if conn == nil || process == nil || err != nil {
			t.Fatalf("dialAndStart = (%v, %v, %v), want (non-nil, non-nil, nil)", conn, process, err)
		}
	}

	// Running a second time: process should be nil (because we didn't start a server)
	{
		conn, process, err := dialAndStart(context.Background(), executable, bindLocation, 30*time.Second)
		defer conn.Close()
		if conn == nil || process != nil || err != nil {
			t.Fatalf("dialAndStart = (%v, %v, %v), want (non-nil, nil, nil)", conn, process, err)
		}
	}
}

// TestDialAndStartLocalhost asserts that calling dialAndStart() first starts a server and connects to it, and subsequently immediately connects.
// In this test, the server binds to localhost:<port>.
func TestDialAndStartLocalhost(t *testing.T) {
	port, cleanup, err := compat.PickUnusedPort()
	if err != nil {
		t.Fatalf("Error creating temporary file:\n%v", err)
	}
	defer cleanup()
	bindLocation := fmt.Sprintf("localhost:%d", port)
	executable := compat.RunfilesPath(*pkgLoaderExecutable)

	// Running for the first time: process should be non-nil.
	{
		conn, process, err := dialAndStart(context.Background(), executable, bindLocation, 30*time.Second)
		defer conn.Close()
		defer process.Kill()
		if conn == nil || process == nil || err != nil {
			t.Fatalf("dialAndStart = (%v, %v, %v), want (non-nil, non-nil, nil)", conn, process, err)
		}
	}

	// Running a second time: process should be nil (because we didn't start a server)
	{
		conn, process, err := dialAndStart(context.Background(), executable, bindLocation, 30*time.Second)
		defer conn.Close()
		if conn == nil || process != nil || err != nil {
			t.Fatalf("dialAndStart = (%v, %v, %v), want (non-nil, nil, nil)", conn, process, err)
		}
	}
}

// TestRestartServer starts a server, then changes the executable's mtime, then tries to connect again.
// It asserts that the second connection causes a server restart.
func TestRestartServer(t *testing.T) {
	bindLocation := "unix://" + tempFileName("grpc_binding_location")
	timeout := 10 * time.Second

	// Create a copy of *pkgLoaderExecutable, to avoid changing the atime/mtime of the original file, later in the test.
	src, err := os.Open(compat.RunfilesPath(*pkgLoaderExecutable))
	if err != nil {
		t.Fatalf("Error opening %s for read:\n%v", compat.RunfilesPath(*pkgLoaderExecutable), err)
	}
	executableFile, err := ioutil.TempFile("", "GrpcLocalServer_deploy.jar")
	executable := executableFile.Name()
	if err != nil {
		t.Fatalf("Error creating temporary file:\n%v", err)
	}
	defer func() {
		os.Remove(executable)
	}()
	if _, err := io.Copy(executableFile, src); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(executable, 0700); err != nil {
		t.Fatal(err)
	}
	executableFile.Close()

	// Running for the first time.
	// Check that the version is 17, which is the mtime of the executable. We later see that the version changes.
	{
		mtime := time.Unix(17, 0)
		if err := os.Chtimes(executable, mtime, mtime); err != nil {
			t.Fatalf("Error changing mtime on %s to %v:\n%v", executable, mtime, err)
		}
		conn, process, err := dialAndStart(context.Background(), executable, bindLocation, timeout)
		if conn == nil || process == nil || err != nil {
			t.Fatalf("dialAndStart = (%v, %v, %v), want (non-nil, non-nil, nil)", conn, process, err)
		}
		defer conn.Close()
		defer process.Kill()
		v := version(conn)
		if v != "17" {
			t.Fatalf("Server has version %s, want 17", v)
		}
	}

	// Changing mtime of the executable, and running the second time.
	// We expect a new process to be returned again (i.e., non-nil) and the version of the server to change.
	{
		mtime := time.Unix(43, 0)
		if err := os.Chtimes(executable, mtime, mtime); err != nil {
			t.Fatalf("Error changing mtime on %s to %v:\n%v", executable, mtime, err)
		}
		conn, process, err := dialAndStart(context.Background(), executable, bindLocation, timeout)
		defer conn.Close()
		defer process.Kill()
		if conn == nil || process == nil || err != nil {
			t.Fatalf("dialAndStart = (%v, %v, %v), want (non-nil, non-nil, nil)", conn, process, err)
		}
		v := version(conn)
		if v != "43" {
			t.Fatalf("Server has version %s, want 43", v)
		}
	}
}

// tempFileName creates a temp file using ioutil.TempFile, then deletes it and returns its name
func tempFileName(prefix string) string {
	tmpFile, err := ioutil.TempFile("", prefix)
	if err != nil {
		panic(fmt.Sprintf("Error creating temporary file:\n%v", err))
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())
	return tmpFile.Name()
}

func version(conn *grpc.ClientConn) string {
	client := sgrpc.NewVersionManagementClient(conn)
	version, err := client.Version(context.Background(), &spb.Empty{})
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return *version.Version
}

func TestDialAddr(t *testing.T) {
	tests := []struct {
		bindLocation  string
		wantAddr      string
		wantBindParam string
		wantType      int
	}{
		{bindLocation: "unix://foo/bar", wantAddr: "foo/bar", wantBindParam: "unix://foo/bar", wantType: uds},
		{bindLocation: "localhost:1234", wantAddr: "localhost:1234", wantBindParam: "1234", wantType: localhost},
		{bindLocation: "93.184.216.34:4317", wantAddr: "93.184.216.34:4317", wantBindParam: "93.184.216.34:4317", wantType: unknown},
	}

	for _, tt := range tests {
		addr, bind, typ := dialAddr(tt.bindLocation)
		if addr != tt.wantAddr || bind != tt.wantBindParam || typ != tt.wantType {
			t.Errorf("dialAddr(%s) = (%v, %v, %v), want (%v, %v, %v)", tt.bindLocation, addr, bind, typ, tt.wantAddr, tt.wantBindParam, tt.wantType)
		}
	}
}

type buildFile struct {
	Path     string
	Contents string
}

func createBuildFileDir(buildFiles []buildFile, workDir string) (func(), error) {
	for _, f := range buildFiles {
		err := os.MkdirAll(filepath.Join(workDir, filepath.FromSlash(f.Path)), 0700)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(filepath.Join(workDir, filepath.FromSlash(f.Path+"/BUILD")), []byte(f.Contents), 0500)
		if err != nil {
			return nil, err
		}
	}
	return func() {
		for _, build := range buildFiles {
			os.RemoveAll(filepath.Join(workDir, build.Path))
		}
	}, nil
}
