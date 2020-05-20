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
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

type TestRunner interface {
	Initialize(context.Context) error
	RunTest(context.Context, Test) (*v1alpha2.ScorecardTestResult, error)
	Cleanup(context.Context) error
}

type Scorecard struct {
	Config      Config
	Selector    labels.Selector
	TestRunner  TestRunner
	SkipCleanup bool
}

type PodTestRunner struct {
	Namespace      string
	ServiceAccount string
	BundlePath     string
	Client         kubernetes.Interface

	configMapName string
}

type FakeTestRunner struct {
	TestResult *v1alpha2.ScorecardTestResult
	Error      error
}

// RunTests executes the scorecard tests as configured
func (o Scorecard) RunTests(ctx context.Context) (testOutput v1alpha2.ScorecardOutput, err error) {

	err = o.TestRunner.Initialize(ctx)
	if err != nil {
		return testOutput, err
	}

	tests := o.selectTests()
	if len(tests) == 0 {
		testOutput.Results = make([]v1alpha2.ScorecardTestResult, 0)
		return testOutput, nil
	}

	testOutput.Results = make([]v1alpha2.ScorecardTestResult, len(tests))

	for idx, test := range tests {
		result, err := o.TestRunner.RunTest(ctx, test)
		if err != nil {
			result = convertErrorToResult(test.Name, test.Description, err)
		}
		testOutput.Results[idx] = *result
	}

	if !o.SkipCleanup {
		err = o.TestRunner.Cleanup(ctx)
		if err != nil {
			return testOutput, err
		}
	}
	return testOutput, nil
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

func (r FakeTestRunner) Initialize(ctx context.Context) (err error) {
	return nil
}

// Initialize sets up the bundle configmap for tests
func (r *PodTestRunner) Initialize(ctx context.Context) error {
	bundleData, err := r.getBundleData()
	if err != nil {
		return fmt.Errorf("error getting bundle data %w", err)
	}

	r.configMapName, err = r.CreateConfigMap(ctx, bundleData)
	if err != nil {
		return fmt.Errorf("error creating ConfigMap %w", err)
	}
	return nil

}

func (r FakeTestRunner) Cleanup(ctx context.Context) (err error) {
	return nil
}

// Cleanup deletes pods and configmap resources from this test run
func (r PodTestRunner) Cleanup(ctx context.Context) (err error) {
	err = r.deletePods(ctx, r.configMapName)
	if err != nil {
		return err
	}
	err = r.deleteConfigMap(ctx, r.configMapName)
	if err != nil {
		return err
	}
	return nil
}

// RunTest executes a single test
func (r PodTestRunner) RunTest(ctx context.Context, test Test) (result *v1alpha2.ScorecardTestResult, err error) {

	// Create a Pod to run the test
	podDef := getPodDefinition(r.configMapName, test, r)
	pod, err := r.Client.CoreV1().Pods(r.Namespace).Create(ctx, podDef, metav1.CreateOptions{})
	if err != nil {
		return result, err
	}

	err = r.waitForTestToComplete(ctx, pod)
	if err != nil {
		return result, err
	}

	result = r.getTestResult(ctx, pod, test)
	return result, nil
}

// RunTest executes a single test
func (r FakeTestRunner) RunTest(ctx context.Context, test Test) (result *v1alpha2.ScorecardTestResult, err error) {
	return r.TestResult, r.Error
}

func ConfigDocLink() string {
	return "https://sdk.operatorframework.io/docs/scorecard/"
}

// waitForTestToComplete waits for a fixed amount of time while
// checking for a test pod to complete
func (r PodTestRunner) waitForTestToComplete(ctx context.Context, p *v1.Pod) (err error) {

	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		var tmp *v1.Pod
		tmp, err = r.Client.CoreV1().Pods(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Errorf("error getting pod %s %w", p.Name, err)
		}
		if tmp.Status.Phase == v1.PodSucceeded {
			return true, nil
		}
		return false, nil
	})

	err = wait.PollImmediateUntil(time.Duration(1*time.Second), podCheck, ctx.Done())
	return err

}

func convertErrorToResult(name, description string, err error) *v1alpha2.ScorecardTestResult {
	result := v1alpha2.ScorecardTestResult{}
	result.Name = name
	result.Description = description
	result.Errors = []string{err.Error()}
	result.Suggestions = []string{}
	result.State = v1alpha2.FailState
	return &result
}
