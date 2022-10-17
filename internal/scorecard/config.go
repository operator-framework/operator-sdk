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
	"os"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"sigs.k8s.io/yaml"
)

const (
	// ConfigFileName is the scorecard's hard-coded config file name.
	ConfigFileName = "config.yaml"
	// DefaultConfigDir is the default scorecard path within a bundle.
	DefaultConfigDir = "tests/scorecard/"
)

// LoadConfig will find and return the scorecard config, the config file
// is found from a bundle location (TODO bundle image)
// scorecard config.yaml is expected to be in the bundle at the following
// location:  tests/scorecard/config.yaml
// the user can override this location using the --config CLI flag
// TODO: version this.
func LoadConfig(configFilePath string) (v1alpha3.Configuration, error) {
	c := v1alpha3.Configuration{}

	yamlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(yamlFile, &c)
	return c, err
}
