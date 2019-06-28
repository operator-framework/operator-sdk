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

package scorecard

import (
	"fmt"
)

type externalPluginConfig struct {
	Command string              `mapstructure:"command"`
	Args    []string            `mapstructure:"args"`
	Env     []externalPluginEnv `mapstructure:"env"`
}

type externalPluginEnv struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

// validateConfig takes a viper config for a plugin and returns a nil error if valid or an error explaining why the config is invalid
func validateConfig(config pluginConfig) error {
	if config.Name == "" {
		return fmt.Errorf("plugin name not set in config")
	}
	// find plugin config type
	pluginType := ""
	if config.Basic != nil {
		pluginType = "basic"
	}
	if config.Olm != nil {
		if pluginType != "" {
			return fmt.Errorf("plugin config can only contain one of: basic, olm, external")
		}
		pluginType = "olm"
	}
	if config.External != nil {
		if pluginType != "" {
			return fmt.Errorf("plugin config can only contain one of: basic, olm, external")
		}
		pluginType = "external"
	}
	if pluginType == "" {
		return fmt.Errorf("plugin %s missing type", config.Name)
	}
	return nil
}
