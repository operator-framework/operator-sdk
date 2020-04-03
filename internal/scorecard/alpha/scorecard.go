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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/operator-framework/operator-sdk/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	Log = logrus.New()
)

type ScorecardFlags struct {
	Config       string
	Bundle       string
	Selector     string
	ListAll      bool
	OutputFormat string
	Kubeconfig   string
}

// RunTests executes the scorecard tests as configured
func RunTests(flags ScorecardFlags) error {
	scConfig, err := lookupConfig(flags)
	if err != nil {
		return err
	}

	selector, err := labels.Parse(flags.Selector)
	if err != nil {
		return err
	}

	tests := selectTests(selector, scConfig.Tests)

	for i := 0; i < len(tests); i++ {
		runTest(tests[i])
	}

	return nil
}

// lookupConfig will find and return the scorecard config, the config file
// can be passed in via command line flag or from a bundle location or
// bundle image
func lookupConfig(flags ScorecardFlags) (ScorecardConfig, error) {
	sc := ScorecardConfig{}

	_, err := os.Stat(flags.Config)
	if os.IsNotExist(err) {
		// TODO get config from bundle (ondisk or image)
		return sc, err
	} else {
		yamlFile, err := ioutil.ReadFile(flags.Config)
		if err != nil {
			return sc, err
		}

		if err := yaml.Unmarshal(yamlFile, &sc); err != nil {
			return sc, err
		}
		return sc, nil
	}

	return sc, nil
}

// selectTests applies an optionally passed selector expression
// against the configured set of tests, returning the selected tests
func selectTests(selector labels.Selector, tests []ScorecardTest) []ScorecardTest {

	selected := make([]ScorecardTest, 0)
	for i := 0; i < len(tests); i++ {
		if selector.String() == "" || selector.Matches(labels.Set(tests[i].Labels)) {
			selected = append(selected, tests[i])
		}
	}
	return selected
}

// runTest executes a single test
// TODO handle the test output
func runTest(test ScorecardTest) error {
	log.Printf("running test %s labels %v", test.Name, test.Labels)
	return nil
}

func ConfigDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf(
		"https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md",
		version.Version)
}
