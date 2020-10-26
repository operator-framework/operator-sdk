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

package scorecard

import (
	"path/filepath"
)

// Static bundle annotation values.
const (
	mediaTypeV1 = "scorecard+v1"
)

// Bundle annotation keys.
// NB(estroz): version these keys based on their "vX" version (either with the version in their names,
// or in subpackages). This may be a requirement if we create "v2" keys.
const (
	mediaTypeBundleKey = "operators.operatorframework.io.test.mediatype.v1"
	configBundleKey    = "operators.operatorframework.io.test.config.v1"
)

func MakeBundleMetadataLabels(configDir string) map[string]string {
	return map[string]string{
		mediaTypeBundleKey: mediaTypeV1,
		configBundleKey:    configDir,
	}
}

func GetConfigDir(labels map[string]string) (value string, hasKey bool) {
	if configKey, hasMTKey := configKeyForMediaType(labels); hasMTKey {
		value, hasKey = labels[configKey]
	}
	return filepath.Clean(filepath.FromSlash(value)), hasKey
}

func configKeyForMediaType(labels map[string]string) (string, bool) {
	switch labels[mediaTypeBundleKey] {
	case mediaTypeV1:
		return configBundleKey, true
	}
	return "", false
}
