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
	// InputDir is a dir containing relevant input files. If not set, a default
	// is used on a per-generator basis.
	InputDir string
	// OutputDir is a dir in which to generate output files. If not set, a
	// default is used on a per-generator basis.
	OutputDir string
	// ExcludeFuncs contains a set of filters for paths that a generator
	// may encounter while gathering data for generation. If a func returns
	// true, that path will be excluded by the generator.
	ExcludeFuncs []func(string) bool
}

// MakeExcludeFuncs creates a set of closures around each path in paths
// to populate Config.ExcludeFuncs. If the argument to the closure has
// a prefix of path, it returns true.
func MakeExcludeFuncs(paths ...string) (excludes []func(string) bool) {
	pathSet := map[string]struct{}{}
	for _, path := range paths {
		pathSet[filepath.Clean(path)] = struct{}{}
	}
	wd := projutil.MustGetwd() + string(filepath.Separator)
	for path := range pathSet {
		excludes = append(excludes, func(p string) bool {
			// Handle absolute paths referencing the project directory.
			p = strings.TrimPrefix(p, wd)
			return strings.HasPrefix(filepath.Clean(p), path)
		})
	}
	return excludes
}
