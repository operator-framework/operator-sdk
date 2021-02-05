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

package olm

var availableVersions = map[string]struct{}{
	"0.16.1": {},
	"0.15.1": {},
	"0.17.0": {},
}

// HasVersion returns whether version maps to released OLM manifests as bindata.
func HasVersion(version string) bool {
	_, ok := availableVersions[version]
	return ok
}
