// Copyright 2018 The Operator-SDK Authors
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

package version

// var needs to be used instead of const for ldflags
var (
	Version           = "unknown"
	GitVersion        = "unknown"
	GitCommit         = "unknown"
	KubernetesVersion = "unknown"

	// ImageVersion represents the ansible-operator, helm-operator, and scorecard subproject versions,
	// which is used in each plugin to specify binary and/or image versions. This is set to the
	// most recent operator-sdk release tag such that samples are generated with the correct versions
	// in a release commit. Once each element that uses this version is moved to a separate repo
	// and release process, this variable will be removed.
	ImageVersion = "unknown"
)
