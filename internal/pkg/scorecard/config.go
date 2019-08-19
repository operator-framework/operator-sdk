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
	"strings"

	"gopkg.in/yaml.v2"
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

func (e externalPluginConfig) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Command: %s %s", e.Command, strings.Join(e.Args, " ")))
	if e.Env != nil {
		sb.WriteString(fmt.Sprintf("\nEnvironment: "))
	}
	for _, env := range e.Env {
		sb.WriteString(fmt.Sprintf("\n\t%s", env))
	}
	return sb.String()
}

func (e externalPluginEnv) String() string {
	return fmt.Sprintf("%s=%s", e.Name, e.Value)
}

// validateConfig takes a viper config for a plugin and returns a nil error if valid or an error explaining why the config is invalid
func validateConfig(config pluginConfig, idx int) error {
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
		marshalledConfig, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("plugin #%d has a missing or incorrect type", idx)
		}
		return fmt.Errorf("plugin #%d has a missing or incorrect type. Invalid plugin config: %s", idx, marshalledConfig)
	}
	return nil
}
