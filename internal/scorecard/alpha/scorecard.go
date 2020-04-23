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
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/version"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	Config          Config
	Selector        labels.Selector
	BundlePath      string
	WaitTime        int
	OutputFormat    string
	Kubeconfig      string
	Namespace       string
	BundleConfigMap *v1.ConfigMap
	ServiceAccount  string
	Client          kubernetes.Interface
	List            bool
	Cleanup         bool
}

// RunTests executes the scorecard tests as configured
func RunTests(o Options) error {
	tests := selectTests(o.Selector, o.Config.Tests)
	if len(tests) == 0 {
		fmt.Println("no tests selected")
		return nil
	}

	bundleData, err := getBundleData(o.BundlePath)
	if err != nil {
		fmt.Printf("error getting bundle data %s\n", err.Error())
		return nil
	}

	// create a ConfigMap holding the bundle contents
	o.BundleConfigMap, err = createConfigMap(o, bundleData)
	if err != nil {
		fmt.Printf("error creating ConfigMap %s\n", err.Error())
		return nil
	}

	for i := 0; i < len(tests); i++ {
		var err error
		tests[i].TestPod, err = runTest(o, tests[i])
		if err != nil {
			return fmt.Errorf("test %s failed %s", tests[i].Name, err.Error())
		}
	}

	if o.Cleanup {
		defer deletePods(o.Client, tests)
		defer deleteConfigMap(o.Client, o.BundleConfigMap)
	}

	err = waitForTestsToComplete(o, tests)
	if err != nil {
		return err
	}

	testOutput := getTestResults(o.Client, tests)
	err = printOutput(o.OutputFormat, testOutput)

	return err
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
func waitForTestsToComplete(o Options, tests []ScorecardTest) (err error) {
	for elapsedSeconds := 0; elapsedSeconds < o.WaitTime; elapsedSeconds++ {
		allPodsCompleted := true
		for i := 0; i < len(tests); i++ {
			p := tests[i].TestPod
			var tmp *v1.Pod
			tmp, err = o.Client.CoreV1().Pods(p.Namespace).Get(p.Name, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("error getting pod %s %s\n", p.Name, err.Error())
				return err
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
