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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/version"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	Config       Config
	Selector     labels.Selector
	List         bool
	OutputFormat string
	Kubeconfig   string
	Client       kubernetes.Interface
}

// RunTests executes the scorecard tests as configured
func RunTests(o Options) error {
	tests := selectTests(o.Selector, o.Config.Tests)
	if len(tests) == 0 {
		fmt.Println("no tests selected")
		return nil
	}

	createdPods := make([]*v1.Pod, 0)
	for i := 0; i < len(tests); i++ {
		pod, err := runTest(o, tests[i])
		if err != nil {
			return fmt.Errorf("test %s failed %s", tests[i].Name, err.Error())
		}
		createdPods = append(createdPods, pod)
	}

	// TODO replace sleep with a watch on the list of pods
	time.Sleep(7 * time.Second)

	printTestResults(o.Client, createdPods)

	return nil
}

// LoadConfig will find and return the scorecard config, the config file
// can be passed in via command line flag or from a bundle location or
// bundle image
func LoadConfig(configFilePath string) (Config, error) {
	c := Config{}

	// TODO handle getting config from bundle (ondisk or image)
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		return c, err
	}

	return c, nil
}

// selectTests applies an optionally passed selector expression
// against the configured set of tests, returning the selected tests
func selectTests(selector labels.Selector, tests []ScorecardTest) []ScorecardTest {

	selected := make([]ScorecardTest, 0)
	for i := 0; i < len(tests); i++ {
		if selector.String() == "" || selector.Matches(labels.Set(tests[i].Labels)) {
			// TODO olm manifests check
			selected = append(selected, tests[i])
		}
	}
	return selected
}

// runTest executes a single test
// TODO once tests exists, handle the test output
func runTest(o Options, test ScorecardTest) (result *v1.Pod, err error) {
	if test.Name == "" {
		return result, errors.New("todo - remove later, only for linter")
	}
	log.Printf("running test %s labels %v", test.Name, test.Labels)

	// Create a Pod to run the test

	podDef := getPodDefinition(test, "default", "default")
	result, err = o.Client.CoreV1().Pods("default").Create(podDef)
	return result, err
}

func ConfigDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf(
		"https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md",
		version.Version)
}

func printTestResults(client kubernetes.Interface, pods []*v1.Pod) {
	for i := 0; i < len(pods); i++ {
		testResult, err := getPodLog(client, *pods[i])
		if err != nil {
			fmt.Printf("error getting pod log %s\n", err.Error())
		} else {
			fmt.Println(testResult)
		}
	}
}
