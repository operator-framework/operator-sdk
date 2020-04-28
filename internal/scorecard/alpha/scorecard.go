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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"github.com/operator-framework/operator-sdk/version"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Scorecard struct {
	Config              Config
	Selector            labels.Selector
	BundlePath          string
	WaitTime            time.Duration
	Kubeconfig          string
	Namespace           string
	ServiceAccount      string
	BundleConfigMapName string
	Client              kubernetes.Interface
	SkipCleanup         bool
	Parallel            bool
}

// RunTests executes the scorecard tests as configured
func (o Scorecard) RunTests() (testOutput v1alpha2.ScorecardOutput, err error) {
	tests := o.selectTests()
	if len(tests) == 0 {
		fmt.Println("no tests selected")
		return testOutput, err
	}

	bundleData, err := getBundleData(o.BundlePath)
	if err != nil {
		return testOutput, fmt.Errorf("error getting bundle data %w", err)
	}

	// create a ConfigMap holding the bundle contents
	o.BundleConfigMapName, err = createConfigMap(o, bundleData)
	if err != nil {
		return testOutput, fmt.Errorf("error creating ConfigMap %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), o.WaitTime)
	defer cancel()

	if o.Parallel {
		wg := sync.WaitGroup{}
		wg.Add(len(tests))
		for idx, test := range tests {
			go func(i int, t Test) {
				defer wg.Done()
				result, err := o.runTest(ctx, test)
				if err != nil {
					result = convertErrorToResult(t, err)
				}
				testOutput.Results = append(testOutput.Results, result)
			}(idx, test)
		}
		wg.Wait()
	} else {

		for _, test := range tests {
			result, err := o.runTest(ctx, test)
			if err != nil {
				result = convertErrorToResult(test, err)
			}
			testOutput.Results = append(testOutput.Results, result)
		}
	}

	if !o.SkipCleanup {
		defer deletePods(o)
		defer deleteConfigMap(o)
	}

	return testOutput, err
}

// selectTests applies an optionally passed selector expression
// against the configured set of tests, returning the selected tests
func (o Scorecard) selectTests() []Test {

	selected := make([]Test, 0)

	for _, test := range o.Config.Tests {
		if o.Selector.String() == "" || o.Selector.Matches(labels.Set(test.Labels)) {
			// TODO olm manifests check
			selected = append(selected, test)
		}
	}
	return selected
}

// runTest executes a single test
func (o Scorecard) runTest(ctx context.Context, test Test) (result v1alpha2.ScorecardTestResult, err error) {

	// Create a Pod to run the test
	podDef := getPodDefinition(test, o)
	pod, err := o.Client.CoreV1().Pods(o.Namespace).Create(podDef)
	if err != nil {
		return result, err
	}

	err = o.waitForTestToComplete(pod)
	if err != nil {
		return result, err
	}

	result = getTestResult(o.Client, pod, test)
	return result, nil
}

func ConfigDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf(
		"https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md",
		version.Version)
}

// waitForTestToComplete waits for a fixed amount of time while
// checking for a test pod to complete
func (o Scorecard) waitForTestToComplete(p *v1.Pod) (err error) {
	waitTimeInSeconds := int(o.WaitTime.Seconds())
	for elapsedSeconds := 0; elapsedSeconds < waitTimeInSeconds; elapsedSeconds++ {
		var tmp *v1.Pod
		tmp, err = o.Client.CoreV1().Pods(p.Namespace).Get(p.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting pod %s %w", p.Name, err)
		}
		if tmp.Status.Phase == v1.PodSucceeded {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("error - wait time of %d seconds has been exceeded", o.WaitTime)

}

func convertErrorToResult(t Test, err error) (result v1alpha2.ScorecardTestResult) {
	result.Name = t.Name
	result.Description = t.Description
	result.Errors = []string{err.Error()}
	result.Suggestions = []string{}
	result.State = v1alpha2.FailState
	return result
}
