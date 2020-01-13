// Copyright 2020 The Operator-SDK Authors
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

package gen

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
	// understands. Keys are exported by the generator's package if any inputs
	// are required. Inputs is meant to be flexible in the case that multiple
	// on-disk inputs are required. If not set, a default is used on a
	// per-generator basis.
	Inputs map[string]string
	// OutputDir is a dir in which to generate output files. If not set, a
	// default is used on a per-generator basis.
	OutputDir string
	// Filters is a set of functional filters for paths that a generator may
	// encounter while gathering data for generation. Filters provides
	// fine-grained control over Inputs, since often those paths are often
	// top-level directories.
	Filters FilterFuncs
}

// FilterFuncs is a slice of filter funcs.
type FilterFuncs []func(string) bool

// MakeFilters creates a set of closures around each path in paths.
// If the argument to a closure has a prefix of path, it returns true.
func MakeFilters(paths ...string) (filters FilterFuncs) {
	pathSet := map[string]struct{}{}
	for _, path := range paths {
		pathSet[filepath.Clean(path)] = struct{}{}
	}
	wd := projutil.MustGetwd() + string(filepath.Separator)
	for path := range pathSet {
		// Copy the string for the closure.
		pb := strings.Builder{}
		pb.WriteString(path)
		filters = append(filters, func(p string) bool {
			// Handle absolute paths referencing the project directory.
			p = strings.TrimPrefix(p, wd)
			return strings.HasPrefix(filepath.Clean(p), pb.String())
		})
	}
	return filters
}

// SatisfiesAny returns true if path passes any filter in funcs.
func (funcs FilterFuncs) SatisfiesAny(path string) bool {
	for _, f := range funcs {
		if f(path) {
			return true
		}
	}
	return false
}
