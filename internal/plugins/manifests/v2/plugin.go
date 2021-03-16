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

package v2

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/plugins"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests"
)

const (
	pluginName = "manifests" + plugins.DefaultNameQualifier
)

var (
	pluginVersion   = plugin.Version{Number: 2}
	pluginConfigKey = plugin.Key(pluginName, pluginVersion.String())
)

// Config configures this plugin, and is saved in the project config file.
type Config struct{}

// HasPluginConfig returns true if cfg.Layout contains an exact match for this plugin's key.
func HasPluginConfig(cfg config.Config) bool {
	info := Config{}
	return cfg.DecodePluginConfig(pluginConfigKey, &info) == nil
}

// RunInit modifies the project scaffolded by kubebuilder's Init plugin.
func RunInit(cfg config.Config) error {
	// Only run these if project version is v3.
	if err := manifests.RunInit(cfg); err != nil {
		return err
	}

	// Update the plugin config section with this plugin's configuration.
	mCfg := Config{}
	if err := cfg.EncodePluginConfig(pluginConfigKey, mCfg); err != nil {
		return fmt.Errorf("error writing plugin config for %s: %v", pluginConfigKey, err)
	}

	return nil
}

// RunCreateAPI runs the manifests SDK phase 2 plugin.
func RunCreateAPI(cfg config.Config, fs machinery.Filesystem, res resource.Resource) error {
	if !HasPluginConfig(cfg) {
		return nil
	}
	return manifests.RunCreateAPI(cfg, fs, res)
}
