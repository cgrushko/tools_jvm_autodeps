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

// Package vlog implements conditional verbose/debug logging.
package vlog

import (
	"log"
)

// Level controls which verbose logging statements are executed.
// It is the minimal number for which V(x) returns true.
var Level = 0

// Verbose is a boolean type that implements info log methods. See V().
type Verbose bool

// V reports whether verbosity at the call site is at least the requested level.
// The returned value is a boolean of type Verbose, which implements Printf.
// This method will write to the log if called.
// Whether an individual call to V generates a log record depends on the setting of Level.
func V(x int) Verbose {
	return Level >= x
}

// Printf is equivalent to log.Printf, guarded by the value of v.
func (v Verbose) Printf(format string, values ...interface{}) {
	if v {
		log.Printf(format, values...)
	}
}
