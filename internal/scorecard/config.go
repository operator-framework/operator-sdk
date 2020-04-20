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

	"sigs.k8s.io/yaml"
)

// ValidateConfig takes a config for a plugin and returns a nil error if valid or an error
// explaining why the config is invalid
func (config PluginConfig) ValidateConfig(idx int) error {
	// find plugin config type
	pluginType := ""
	if config.Basic != nil {
		pluginType = "basic"
	}
	if config.Olm != nil {
		if pluginType != "" {
			return fmt.Errorf("plugin config can only contain one of: basic, olm")
		}
		pluginType = "olm"
	}
	if pluginType == "" {
		marshalledConfig, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("plugin #%d has a missing or incorrect type", idx)
		}
		return fmt.Errorf("plugin #%d has a missing or incorrect type. Invalid plugin config: %s",
			idx, marshalledConfig)
	}
	return nil
}
