// Copyright 2021 The Operator-SDK Authors
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

package index

import "fmt"

// BundleAddMode is the mode to add a bundle to an index.
type BundleAddMode string

const (
	// SemverBundleAddMode - bundle add mode for semver
	SemverBundleAddMode BundleAddMode = "semver"
	// ReplacesBundleAddMode - bundle add mode for replaces
	ReplacesBundleAddMode BundleAddMode = "replaces"
)

var modes = []BundleAddMode{SemverBundleAddMode, ReplacesBundleAddMode}

func (m BundleAddMode) Validate() error {
	switch m {
	case SemverBundleAddMode, ReplacesBundleAddMode:
	case "":
		return fmt.Errorf("bundle add mode cannot be empty, must be one of: %+q", modes)
	default:
		return fmt.Errorf("bundle add mode %q does not exist, must be one of: %+q", m, modes)
	}

	return nil
}
