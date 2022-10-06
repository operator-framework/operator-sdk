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

package v2

import (
	"errors"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

const (
	goV3APluginKey         = "go.sdk.operatorframework.io/v3"
	legacyGoPluginKey      = "go.sdk.operatorframework.io/v2"
	legacyGoPluginAlphaKey = "go.sdk.operatorframework.io/v2-alpha"
	kubebuilderGoV3        = "go.kubebuilder.io/v3"
	kubebuilderGoV2        = "go.kubebuilder.io/v2"
)

// HasSupportForKustomizeV4 will return true when we can do the scaffolds
// using kustomize version
func HasSupportForKustomizeV4(c config.Config) bool {

	// When we are creating the project with the Bundle Plugin
	// we have data only into the pluginChain
	for _, pluginKey := range c.GetPluginChain() {
		if strings.HasPrefix(pluginKey, "go") {
			switch {
			case pluginKey == kubebuilderGoV3:
				return false
			case pluginKey == kubebuilderGoV2:
				return false
			}
		}
	}

	// If we call the implementation afterwords as we do
	// via the subCommand "operator-sdk generate kustomize"
	// we have the PROJECT file and the plugin layout
	err := c.DecodePluginConfig(goV3APluginKey, struct{}{})
	if err == nil || !errors.As(err, &config.PluginKeyNotFoundError{}) {
		return false
	}

	err = c.DecodePluginConfig(legacyGoPluginKey, struct{}{})
	if err == nil || !errors.As(err, &config.PluginKeyNotFoundError{}) {
		return false
	}

	err = c.DecodePluginConfig(legacyGoPluginAlphaKey, struct{}{})
	if err == nil || !errors.As(err, &config.PluginKeyNotFoundError{}) {
		return false
	}

	err = c.DecodePluginConfig(kubebuilderGoV2, struct{}{})
	if err == nil || !errors.As(err, &config.PluginKeyNotFoundError{}) {
		return false
	}

	err = c.DecodePluginConfig(kubebuilderGoV3, struct{}{})
	if err == nil || !errors.As(err, &config.PluginKeyNotFoundError{}) {
		return false
	}

	return true
}
