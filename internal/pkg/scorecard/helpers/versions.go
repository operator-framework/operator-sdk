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

package schelpers

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

const v1alpha1 = "v1alpha1"
const v1alpha2 = "v1alpha2"
const DefaultScorecardVersion = v1alpha1
const LatestScorecardVersion = v1alpha2
const VersionOpt = "version"

var ScorecardVersions = []string{DefaultScorecardVersion, LatestScorecardVersion}

func ValidateVersion(version string) error {
	for _, a := range ScorecardVersions {
		if a == version {
			return nil
		}
	}
	return fmt.Errorf("invalid scorecard version (%s); valid values: %s", version, strings.Join(ScorecardVersions, ", "))

}

func IsV1alpha2() bool {
	if viper.Sub("scorecard").GetString(VersionOpt) == v1alpha2 {
		return true
	}
	return false
}

func IsLatestVersion() bool {
	if viper.Sub("scorecard").GetString(VersionOpt) == LatestScorecardVersion {
		return true
	}
	return false
}
