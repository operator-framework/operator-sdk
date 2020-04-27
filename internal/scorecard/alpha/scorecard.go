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
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"github.com/operator-framework/operator-sdk/version"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Scorecard struct {
	Config          Config
	Selector        labels.Selector
	BundlePath      string
	WaitTime        time.Duration
	Kubeconfig      string
	Namespace       string
	bundleConfigMap *v1.ConfigMap
	ServiceAccount  string
	Client          kubernetes.Interface
	SkipCleanup     bool
}

// RunTests executes the scorecard tests as configured
func (o Scorecard) RunTests() (testOutput v1alpha2.ScorecardOutput, err error) {
	tests := selectTests(o.Selector, o.Config.Tests)
	if len(tests) == 0 {
		fmt.Println("no tests selected")
		return testOutput, err
	}

	bundleData, err := getBundleData(o.BundlePath)
	if err != nil {
		return testOutput, fmt.Errorf("error getting bundle data %w", err)
	}

	// create a ConfigMap holding the bundle contents
	o.bundleConfigMap, err = createConfigMap(o, bundleData)
	if err != nil {
		return testOutput, fmt.Errorf("error creating ConfigMap %w", err)
	}

	for i, test := range tests {
		var err error
		tests[i].TestPod, err = o.runTest(test)
		if err != nil {
			return testOutput, fmt.Errorf("test %s failed %w", test.Name, err)
		}
	}

	if !o.SkipCleanup {
		defer deletePods(o.Client, tests)
		defer deleteConfigMap(o.Client, o.bundleConfigMap)
	}

	err = o.waitForTestsToComplete(tests)
	if err != nil {
		return testOutput, err
	}

	testOutput = getTestResults(o.Client, tests)

	return testOutput, err
}

// selectTests applies an optionally passed selector expression
// against the configured set of tests, returning the selected tests
func selectTests(selector labels.Selector, tests []Test) []Test {

	selected := make([]Test, 0)

	for _, test := range tests {
		if selector.String() == "" || selector.Matches(labels.Set(test.Labels)) {
			// TODO olm manifests check
			selected = append(selected, test)
		}
	}
	return selected
}

// runTest executes a single test
func (o Scorecard) runTest(test Test) (result *v1.Pod, err error) {

	// Create a Pod to run the test
	podDef := getPodDefinition(test, o)
	result, err = o.Client.CoreV1().Pods(o.Namespace).Create(podDef)
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

// waitForTestsToComplete waits for a fixed amount of time while
// checking for test pods to complete
func (o Scorecard) waitForTestsToComplete(tests []Test) (err error) {
	waitTimeInSeconds := int(o.WaitTime.Seconds())
	for elapsedSeconds := 0; elapsedSeconds < waitTimeInSeconds; elapsedSeconds++ {
		allPodsCompleted := true
		for _, test := range tests {
			p := test.TestPod
			var tmp *v1.Pod
			tmp, err = o.Client.CoreV1().Pods(p.Namespace).Get(p.Name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("error getting pod %s %w", p.Name, err)
			}
			if tmp.Status.Phase != v1.PodSucceeded {
				allPodsCompleted = false
			}

		}
		if allPodsCompleted {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("error - wait time of %d seconds has been exceeded", o.WaitTime)

}
