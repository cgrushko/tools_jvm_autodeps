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

package jadepmain

import (
	"time"

	"context"
	"github.com/bazelbuild/tools_jvm_autodeps/jadeplib"
	"github.com/bazelbuild/tools_jvm_autodeps/pkgloading"
)

// Customization is the specialization point for Main.
type Customization interface {
	// LoadDataSources loads any specialized data sources to be used by NewResolvers.
	LoadDataSources(context.Context) DataSources

	// NewDepsRanker returns the DepsRanker to be used in Jadep.
	NewDepsRanker(DataSources) jadeplib.DepsRanker

	// NewResolvers returns any specialized resolvers an organization has.
	// For example, an organization which employs Kythe to index their depot might implement a resolver that takes advantage of that index.
	NewResolvers(loader pkgloading.Loader, data interface{}) []jadeplib.Resolver

	// NewLoader returns a new Loader which will be used to load Bazel packages.
	NewLoader(ctx context.Context, flags *Flags, workspaceDir string) (pkgloading.Loader, func(), error)
}

// DataSources is customized by users of jadepmain.Main to pass information between Customization.LoadDataSources and NewDepsRanker, NewResolvers.
type DataSources interface{}

// Flags gathers command-line flags that jadepmain uses.
type Flags struct {
	// See corresponding flag in jadep.go
	Workspace string

	// See corresponding flag in jadep.go
	ContentRoots []string

	// See corresponding flag in jadep.go
	DryRun bool

	// See corresponding flag in jadep.go
	ClassNames []string

	// See corresponding flag in jadep.go
	Blacklist []string

	// See corresponding flag in jadep.go
	BlacklistedPackageList string

	// See corresponding flag in jadep.go
	BuiltinClassList string

	// See corresponding flag in jadep.go
	PkgLoaderExecutable string

	// See corresponding flag in jadep.go
	PkgLoaderAddress string

	// See corresponding flag in jadep.go
	RPCDeadline time.Duration

	// See corresponding flag in jadep.go
	Cpuprofile string

	// See corresponding flag in jadep.go
	Vlevel int
}
