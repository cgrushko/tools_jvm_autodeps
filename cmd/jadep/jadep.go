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

// The jadep command adds `BUILD` dependencies that a Java file needs.
package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"flag"
	"context"

	"github.com/bazelbuild/tools_jvm_autodeps/bazeldepsresolver"
	"github.com/bazelbuild/tools_jvm_autodeps/cli"
	"github.com/bazelbuild/tools_jvm_autodeps/filter"
	"github.com/bazelbuild/tools_jvm_autodeps/grpcloader"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/jadepmain"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
	"github.com/bazelbuild/tools_jvm_autodeps/sortingdepsranker"
)

var flags jadepmain.Flags
var strContentRoots, strClassNames, strBlacklist string

var (
	bazelInstallBase = flag.String("bazel_install_base", "", "the value of 'bazel info install_base'")
	bazelOutputBase  = flag.String("bazel_output_base", "", "the value of 'bazel info output_base'")

	thirdpartyJvmDir = flag.String("thirdparty_jvm_dir", "thirdparty/jvm", "the directory where https://github.com/johnynek/bazel-deps placed its generated BUILD files")
)

func init() {
	log.Printf("Jadep - fix BUILD for Java code")
	u, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user")
	}

	flag.StringVar(&flags.Workspace, "workspace", "", "a Bazel WORKSPACE directory to operate in. Defaults to working directory")
	flag.StringVar(&strContentRoots, "content_roots", "src/main/java,src/test/java", "locations of Java sources relative to -workspace (comma delimited)")
	flag.BoolVar(&flags.DryRun, "dry_run", false, "only prints missing/unknown deps")
	flag.StringVar(&strClassNames, "classnames", "", "when present, Jade will find dependencies for these class names instead of parsing the Java file to look for class names without dependencies (comma delimited).")
	flag.StringVar(&strBlacklist, "blacklist", `.*\.R$`, "a list of regular expressions matching names of classes for which we will not look for BUILD rules (comma delimited).")
	flag.StringVar(&flags.BlacklistedPackageList, "blacklisted_package_list", filepath.Join(u.HomeDir, "jadep/blacklisted_packages.txt"), "File containing BUILD package names that Jade will not load. Usual use-case: package takes too long to load and doesn't contain anything we need.")
	flag.StringVar(&flags.BuiltinClassList, "builtin_classlist", filepath.Join(u.HomeDir, "jadep/jdk_android_builtin_class_names.txt"), "File containing class names that don't need deps, e.g. JDK classes. One class name per line, sorted.")
	flag.StringVar(&flags.PkgLoaderExecutable, "pkgloader_executable", filepath.Join(u.HomeDir, "jadep/pkgloader_server.sh"), "path to a package loader server executable. Started when Jade fails to connect to --pkg_loader_bind_location")
	flag.StringVar(&flags.PkgLoaderAddress, "pkgloader_address", "", "Address of a pkgloader service. "+
		"If prefixed with unix://, assumed to be a Unix domain socket. "+
		"If Jade fails to connect and this flag is prefixed with one of unix:// or localhost:, it starts the executable pointed at by --pkgloader_executable. "+
		"Note that other forms, including IP addresses, will not cause Jade to start a server. "+
		"localhost:0 is unsupported. "+
		"The defaut is unix://<homedir>/pkgloader.socket")
	flag.DurationVar(&flags.RPCDeadline, "rpc_deadline", 15*time.Second, "Time before giving up on RPC connections.")
	flag.StringVar(&flags.Cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.IntVar(&flags.Vlevel, "vlevel", 0, "Enable V-leveled logging at the specified level")
	flag.BoolVar(&flags.Color, "color", true, "Colorize output. If stdout or stderr are not terminals, the output will not be colorized and this flag will have no effect")
}

func main() {
	flag.Parse()
	flags.ContentRoots = strings.Split(strContentRoots, ",")
	if strClassNames == "" {
		flags.ClassNames = nil
	} else {
		flags.ClassNames = strings.Split(strClassNames, ",")
	}
	flags.Blacklist = strings.Split(strBlacklist, ",")

	workspaceDir, _, err := cli.Workspace(flags.Workspace)
	if err != nil {
		log.Fatalf("Can't find root of workspace: %v", err)
	}

	bazelInstallBase := *bazelInstallBase
	bazelOutputBase := *bazelOutputBase
	if bazelInstallBase == "" || bazelOutputBase == "" {
		install, output, err := guessBazelBases(workspaceDir)
		if err != nil {
			log.Fatalf("Can't find Bazel install and output bases. Explicitly pass --bazel_install_base and --bazel_output_base.\n%v", err)
		}
		if bazelInstallBase == "" {
			bazelInstallBase = install
		}
		if bazelOutputBase == "" {
			bazelOutputBase = output
		}
	}

	jadepmain.Main(customization{workspaceDir, bazelInstallBase, bazelOutputBase}, &flags, flag.Args())
}

// guessBazelBases guesses the output and install bases of the current Bazel workspace.
// It is a horrible piece of hack and I'm ashamed of it.
func guessBazelBases(workspaceDir string) (installBase string, outputBase string, err error) {
	bazelOut, err := os.Readlink(filepath.Join(workspaceDir, "bazel-out"))
	if err != nil {
		return "", "", fmt.Errorf("couldn't resolve the bazel-out/ symlink: %v", err)
	}
	outputBase = filepath.Dir(filepath.Dir(filepath.Dir(bazelOut)))
	installBase, err = os.Readlink(filepath.Join(outputBase, "install"))
	if err != nil {
		return "", "", fmt.Errorf("couldn't resolve the install base symlink: %v", err)
	}
	return installBase, outputBase, nil
}

type customization struct {
	workspaceDir     string
	bazelInstallBase string
	bazelOutputBase  string
}

func (c customization) LoadDataSources(ctx context.Context) jadepmain.DataSources {
	return nil
}

func (c customization) NewDepsRanker(data jadepmain.DataSources) jadeplib.DepsRanker {
	return &sortingdepsranker.Ranker{}
}

func (c customization) NewResolvers(loader pkgloading.Loader, data interface{}) []jadeplib.Resolver {
	r, err := bazeldepsresolver.NewResolver(context.Background(), c.workspaceDir, *thirdpartyJvmDir, loader)
	if err != nil {
		log.Printf("Warning: couldn't create bazel-deps resolver: %v", err)
		return nil
	}
	return []jadeplib.Resolver{r}
}

func (c customization) NewLoader(ctx context.Context, flags *jadepmain.Flags, workspaceDir string) (pkgloading.Loader, func(), error) {
	return grpcloader.Connect(ctx, flags.PkgLoaderExecutable, flags.PkgLoaderAddress, flags.RPCDeadline, workspaceDir, c.bazelInstallBase, c.bazelOutputBase, keys(filter.RuleKindsToLoad))
}

func keys(m map[string]bool) []string {
	ret := make([]string, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}
