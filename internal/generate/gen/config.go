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

// TODO(hasbro17/estroz): Remove the generator config in favor of generator
// specific option structs configured with Inputs and OutputDir.
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
	// OutputDir is the root directory where the output files will be generated.
	// If not set, a default is used on a per-generator basis.
	OutputDir string
}
