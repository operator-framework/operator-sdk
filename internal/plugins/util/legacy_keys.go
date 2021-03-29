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

package util

import (
	"errors"

	log "github.com/sirupsen/logrus"
	gofunk "github.com/thoas/go-funk"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
)

const (
	// The catch-all plugin key for the go/v2+manifests+scorecard plugins.
	// Should still be accepted for backwards-compat.
	legacyGoPluginKey = "go.sdk.operatorframework.io/v2-alpha"

	// Hard-code the latest manifests and scorecard keys here to avoid a circular import.
	manifestsKey = "manifests.sdk.operatorframework.io/v2"
	scorecardKey = "scorecard.sdk.operatorframework.io/v2"
)

// Plugin keys that existed when manifests/scorecard keys did not.
var acceptedLayoutKeys = []string{
	"ansible.sdk.operatorframework.io/v1",
	"helm.sdk.operatorframework.io/v1",
}

// UpdateIfLegacyKey returns true if c's "plugins" map or "layout" value contains
// a legacy key that may require this plugin be executed, even if the "manifests" key
// isn't in "plugins".
func UpdateIfLegacyKey(c config.Config) bool {
	if c.GetVersion().Compare(cfgv3.Version) < 0 {
		return false
	}

	err := c.DecodePluginConfig(legacyGoPluginKey, struct{}{})
	if err == nil || !errors.As(err, &config.PluginKeyNotFoundError{}) {
		// There is no way to remove keys from "plugins", so print a warning.
		log.Warnf("Plugin key %q is deprecated. Replace this key with %q and %q on separate lines.",
			legacyGoPluginKey, manifestsKey, scorecardKey)
		return true
	}

	chain := c.GetPluginChain()
	for _, key := range acceptedLayoutKeys {
		if gofunk.ContainsString(chain, key) {
			// Encode missing plugin keys.
			if !gofunk.ContainsString(chain, manifestsKey) {
				if err := c.EncodePluginConfig(manifestsKey, struct{}{}); err != nil {
					log.Error(err)
				}
			}
			if !gofunk.ContainsString(chain, scorecardKey) {
				if err := c.EncodePluginConfig(scorecardKey, struct{}{}); err != nil {
					log.Error(err)
				}
			}
			return true
		}
	}

	return false
}
