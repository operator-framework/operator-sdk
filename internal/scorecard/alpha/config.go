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

package alpha

import (
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

const (
	ConfigDirName = "scorecard"
	ConfigDirPath = "/tests/" + ConfigDirName + "/"
)

type Test struct {
	Name  string `yaml:"name"`  // The container test name
	Image string `yaml:"image"` // The container image name
	// An list of commands and arguments passed to the test image
	Entrypoint  []string          `yaml:"entrypoint,omitempty"`
	Labels      map[string]string `yaml:"labels"`      // User defined labels used to filter tests
	Description string            `yaml:"description"` // User readable test description
}

// Config represents the set of test configurations which scorecard
// would run based on user input
type Config struct {
	Tests []Test `yaml:"tests"`
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
