// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genutil

import (
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// Config configures a generator with common operator project information.
type Config struct {
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	// Inputs is an arbitrary map of keys to paths that an individual generator
	// understands. Keys are either exported by the generator's package, or the
	// global "" key can be used if none are exported. Inputs is meant to be
	// flexible in the case that multiple on-disk input files are required.
	// If not set, a default is used on a per-generator basis.
	Inputs map[string]string
	// OutputDir is a dir in which to generate output files. If not set, a
	// default is used on a per-generator basis.
	OutputDir string
	// IncludeFuncs contains a set of filters for paths that a generator
	// may encounter while gathering data for generation. If any func returns
	// true, that path will be included by the generator. IncludeFuncs provides
	// fine-grained control over Inputs, since often those paths are often
	// top-level directories.
	IncludeFuncs IncludeFuncs
}

// GetInputPaths returns all paths in c.Inputs for a set of keys. If the global
// key "" is set, only that path will be returned.
func (c Config) GetInputPaths(keys ...string) (paths []string) {
	if len(c.Inputs) == 0 {
		return
	}
	if global, ok := c.Inputs[""]; ok {
		return []string{global}
	}
	for _, key := range keys {
		if path, ok := c.Inputs[key]; ok {
			paths = append(paths, path)
		}
	}
	return paths
}

// IncludeFuncs is a slice of filter funcs. A string passing any func in
// IncludeFuncs satisfies the filter.
type IncludeFuncs []func(string) bool

// MakeIncludeFuncs creates a set of closures around each path in paths
// to populate Config.IncludeFuncs. If the argument to the closure has
// a prefix of path, it returns true.
func MakeIncludeFuncs(paths ...string) (includes IncludeFuncs) {
	pathSet := map[string]struct{}{}
	for _, path := range paths {
		pathSet[filepath.Clean(path)] = struct{}{}
	}
	wd := projutil.MustGetwd() + string(filepath.Separator)
	for path := range pathSet {
		// Copy the string for the closure.
		pb := strings.Builder{}
		pb.WriteString(path)
		includes = append(includes, func(p string) bool {
			// Handle absolute paths referencing the project directory.
			p = strings.TrimPrefix(p, wd)
			return strings.HasPrefix(filepath.Clean(p), pb.String())
		})
	}
	return includes
}

// IsInclude checks if path passes any filter in funcs.
func (funcs IncludeFuncs) IsInclude(path string) bool {
	for _, f := range funcs {
		if f(path) {
			return true
		}
	}
	return false
}
