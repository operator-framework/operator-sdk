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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"github.com/operator-framework/operator-sdk/version"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	Config         Config
	Selector       labels.Selector
	List           bool
	OutputFormat   string
	Kubeconfig     string
	Namespace      string
	ServiceAccount string
	Client         kubernetes.Interface
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

	testOutput := getTestResults(o.Client, createdPods)
	printOutput(o.OutputFormat, testOutput)

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

	// Create a Pod to run the test
	podDef := getPodDefinition(test, o.Namespace, o.ServiceAccount)
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

func getTestResults(client kubernetes.Interface, pods []*v1.Pod) (output v1alpha2.ScorecardOutput) {
	output.Results = make([]v1alpha2.ScorecardTestResult, 0)
	for i := 0; i < len(pods); i++ {
		logBytes, err := getPodLog(client, *pods[i])
		if err != nil {
			fmt.Printf("error getting pod log %s\n", err.Error())
		} else {
			// marshal pod log into ScorecardTestResult
			var sc v1alpha2.ScorecardTestResult
			err := json.Unmarshal(logBytes, &sc)
			if err != nil {
				fmt.Printf("error unmarshalling test result %s\n", err.Error())
			} else {
				output.Results = append(output.Results, sc)
			}
		}
	}
	return output
}

// ListTests lists the scorecard tests as configured that would be
// run based on user selection
func ListTests(o Options) error {
	tests := selectTests(o.Selector, o.Config.Tests)
	if len(tests) == 0 {
		fmt.Println("no tests selected")
		return nil
	}

	output := v1alpha2.ScorecardOutput{}
	output.Results = make([]v1alpha2.ScorecardTestResult, 0)

	for i := 0; i < len(tests); i++ {
		testResult := v1alpha2.ScorecardTestResult{}
		testResult.Name = tests[i].Name
		testResult.Labels = tests[i].Labels
		testResult.Description = tests[i].Description
		output.Results = append(output.Results, testResult)
	}

	printOutput(o.OutputFormat, output)

	return nil
}

func printOutput(outputFormat string, output v1alpha2.ScorecardOutput) {
	if outputFormat == "text" {
		o, err := output.MarshalText()
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		fmt.Printf("%s\n", o)
		return
	}
	if outputFormat == "json" {
		bytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		fmt.Printf("%s\n", string(bytes))
		return
	}

	fmt.Printf("error, invalid output format selected")

}
