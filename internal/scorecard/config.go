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
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

const (
	// ConfigFileName is the scorecard's hard-coded config file name.
	ConfigFileName = "config.yaml"
	// DefaultConfigDir is the default scorecard path within a bundle.
	DefaultConfigDir = "tests/scorecard/"
)

type Stage struct {
	Parallel bool   `yaml:"parallel"`
	Tests    []Test `yaml:"tests"`
}

type Test struct {
	// Image is the name of the testimage
	Image string `json:"image"`
	// Entrypoint is list of commands and arguments passed to the test image
	Entrypoint []string `json:"entrypoint,omitempty"`
	// Labels that further describe the test and enable selection
	Labels map[string]string `json:"labels,omitempty"`
}

// Config represents the set of test configurations which scorecard
// would run based on user input
type Config struct {
	Stages []Stage `yaml:"stages"`
}

// LoadConfig will find and return the scorecard config, the config file
// is found from a bundle location (TODO bundle image)
// scorecard config.yaml is expected to be in the bundle at the following
// location:  tests/scorecard/config.yaml
// the user can override this location using the --config CLI flag
func LoadConfig(configFilePath string) (Config, error) {
	c := Config{}

	// TODO handle bundle images, not just on-disk
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		return c, err
	}

	return c, nil
}
