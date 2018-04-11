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

// Package jadepmain contains Jadep's main() function.
// Its purpose is to allow organizations to build specialized Jadep binaries without forking the main repo.
package jadepmain

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/bazel"
	"github.com/bazelbuild/tools_jvm_autodeps/buildozer"
	"github.com/bazelbuild/tools_jvm_autodeps/cli"
	"github.com/bazelbuild/tools_jvm_autodeps/dictresolver"
	"github.com/bazelbuild/tools_jvm_autodeps/fsresolver"
	"github.com/bazelbuild/tools_jvm_autodeps/future"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/lang/java/ruleconsts"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
	"github.com/bazelbuild/tools_jvm_autodeps/vlog"
)

// Main is an entry point to Jadep program.
// Its purpose is to allow organizations to build their own specialized Jadep's without forking the main repo.
// This function should only be called from a main.main() function, as it uses and modifies global variables.
// args are the non-flag command-line arguments.
func Main(custom Customization, flags *Flags, args []string) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	vlog.Level = flags.Vlevel
	ctx := context.Background()
	stopProfiler := cli.StartProfiler(flags.Cpuprofile)
	defer stopProfiler()
	if len(args) == 0 {
		log.Fatalln("Must provide at least one Java file or BUILD rule to process.")
	}
	vlog.V(3).Printf("Processing files/rules: %v", args)
	wd, relWorkingDir, err := cli.Workspace(flags.Workspace)
	if err != nil {
		log.Fatalf("Can't find root of workspace: %v", err)
	}
	config := jadeplib.Config{WorkspaceDir: wd}

	blacklistedPackageList := readFileLines(flags.BlacklistedPackageList)
	builtinClassList := readDictFromCSV(flags.BuiltinClassList)
	implicitImports := jadeplib.ImplicitImports(builtinClassList)

	dataSources := custom.LoadDataSources(ctx)

	var cleanup func()
	config.Loader, cleanup = newLoader(ctx, custom, flags, config.WorkspaceDir, blacklistedPackageList.Get().([]string))
	defer cleanup()

	config.DepsRanker = custom.NewDepsRanker(dataSources)

	config.Resolvers = []jadeplib.Resolver{
		dictresolver.NewResolver("Built-in JDK/Android", builtinClassList, config.Loader),
		fsresolver.NewResolver(flags.ContentRoots, config.WorkspaceDir, config.Loader),
	}
	config.Resolvers = append(config.Resolvers, custom.NewResolvers(config.Loader, dataSources)...)

	for _, arg := range args {
		rulesToFix, err := cli.RulesToFix(ctx, config, relWorkingDir, arg, ruleconsts.NewRuleNamingRules, ruleconsts.DefaultNewRuleKind)
		if err != nil {
			log.Fatal(err)
		}
		cli.LogRulesToFix(rulesToFix)
		classNamesToResolve := cli.ClassNamesToResolve(ctx, filepath.Join(config.WorkspaceDir, relWorkingDir), config.Loader, arg, flags.ClassNames, implicitImports, flags.Blacklist)
		missingDepsMap, unresClasses, err := jadeplib.MissingDeps(ctx, config, rulesToFix, classNamesToResolve)
		if err != nil {
			log.Printf("WARNING: Error computing missing dependencies:\n%v.", err)
			continue
		}

		cli.ReportUnresolvedClassnames(unresClasses)
		fmt.Println()
		if flags.DryRun {
			cli.ReportMissingDeps(missingDepsMap)
		} else {
			// for each rule that's missing deps, which deps to add
			depsToAdd, err := jadeplib.SelectDepsToAdd(os.Stdin, missingDepsMap)
			if err != nil {
				log.Printf("WARNING: Error asking user to choose dependencies to add:\n%v", err)
				continue
			}
			err = buildozer.AddDepsToRules(config.WorkspaceDir, depsToAdd)
			if err != nil {
				log.Printf("WARNING: error adding missing deps to rules:\n%v", err)
				continue
			}
		}
	}
}

func newLoader(ctx context.Context, custom Customization, flags *Flags, workspaceDir string, blacklistedPackageList []string) (pkgloading.Loader, func()) {
	if flags.PkgLoaderAddress == "" {
		flags.PkgLoaderAddress = defaultPkgLoaderAddress()
	}
	rpcLoader, cleanup, err := custom.NewLoader(ctx, flags, workspaceDir)
	if err != nil {
		log.Fatalf("Error connecting to PackageLoader service:\n%v", err)
	}
	filteringLoader := &pkgloading.FilteringLoader{rpcLoader, listToSet(blacklistedPackageList)}
	return pkgloading.NewCachingLoader(filteringLoader), cleanup
}

func defaultPkgLoaderAddress() string {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user. Pass a non-empty --pkg_loader_bind_location explicitly to avoid needing it")
	}
	return "unix://" + filepath.Join(u.HomeDir, "pkgloader.socket")
}

func readFileLines(fileName string) *future.Value {
	return future.NewValue(func() interface{} {
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Printf("WARNING: Error while reading %q: %v", fileName, err)
			return []string(nil)
		}
		return strings.Split(string(content), "\n")
	})
}

// readDictFromCSV reads a CSV whose first column is a class name, and the rest of the columns are Bazel rules that resolve it.
// The return type is a future that wraps a map[jadeplib.ClassName][]bazel.Label
func readDictFromCSV(fileName string) *future.Value {
	return future.NewValue(func() interface{} {
		f, err := os.Open(fileName)
		if err != nil {
			log.Printf("Error opening %s: %v", fileName, err)
			return map[jadeplib.ClassName][]bazel.Label(nil)
		}
		result, err := dictresolver.ReadDictFromCSV(f)
		if err != nil {
			log.Printf("Error while reading %q: %v", fileName, err)
			return map[jadeplib.ClassName][]bazel.Label(nil)
		}
		return result
	})
}

func listToSet(strs []string) map[string]bool {
	ret := make(map[string]bool)
	for _, s := range strs {
		ret[s] = true
	}
	return ret
}
