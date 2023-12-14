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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

type TestRunner interface {
	Initialize(context.Context) error
	RunTest(context.Context, v1alpha3.TestConfiguration, bool) (*v1alpha3.TestStatus, error)
	Cleanup(context.Context) error
}

type Scorecard struct {
	Config      v1alpha3.Configuration
	Selector    labels.Selector
	TestRunner  TestRunner
	SkipCleanup bool
	PodSecurity bool
}

type PodTestRunner struct {
	Namespace      string
	ServiceAccount string
	BundlePath     string
	TestOutput     string
	BundleMetadata registryutil.LabelsMap
	Client         kubernetes.Interface
	RESTConfig     *rest.Config
	StorageImage   string
	UntarImage     string

	configMapName string
	PodSecurity   bool
}

type FakeTestRunner struct {
	Sleep      time.Duration
	TestStatus *v1alpha3.TestStatus
	Error      error
}

// cleanupTimeout is the time given to clean up resources, regardless of how long ctx's deadline is.
var cleanupTimeout = time.Second * 30

// Run executes the scorecard tests as configured
func (o Scorecard) Run(ctx context.Context) (testOutput v1alpha3.TestList, err error) {
	testOutput = v1alpha3.NewTestList()

	if err := o.TestRunner.Initialize(ctx); err != nil {
		return testOutput, err
	}

	for _, stage := range o.Config.Stages {
		tests := o.selectTests(stage)
		if len(tests) == 0 {
			continue
		}
		tests = o.setTestDefaults(tests)

		output := make(chan v1alpha3.Test, len(tests))
		if stage.Parallel {
			o.runStageParallel(ctx, tests, output)
		} else {
			o.runStageSequential(ctx, tests, output)
		}
		close(output)
		for o := range output {
			testOutput.Items = append(testOutput.Items, o)
		}
	}

	// Get timeout error, if any, before calling Cleanup() so deletes don't cause a timeout.
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
	}

	if !o.SkipCleanup {
		// Use a separate context for cleanup, which needs to run regardless of a prior timeout.
		clctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		if err := o.TestRunner.Cleanup(clctx); err != nil {
			return testOutput, err
		}
	}

	return testOutput, err
}

func (o Scorecard) setTestDefaults(tests []v1alpha3.TestConfiguration) []v1alpha3.TestConfiguration {
	for i := range tests {
		if len(tests[i].Storage.Spec.MountPath.Path) == 0 {
			tests[i].Storage.Spec.MountPath.Path = o.Config.Storage.Spec.MountPath.Path
		}
	}
	return tests
}

func (o Scorecard) runStageParallel(ctx context.Context, tests []v1alpha3.TestConfiguration, results chan<- v1alpha3.Test) {
	var wg sync.WaitGroup
	for _, t := range tests {
		wg.Add(1)
		go func(test v1alpha3.TestConfiguration) {
			results <- o.runTest(ctx, test)
			wg.Done()
		}(t)
	}
	wg.Wait()
}

func (o Scorecard) runStageSequential(ctx context.Context, tests []v1alpha3.TestConfiguration, results chan<- v1alpha3.Test) {
	for _, test := range tests {
		results <- o.runTest(ctx, test)
	}
}

func (o Scorecard) runTest(ctx context.Context, test v1alpha3.TestConfiguration) v1alpha3.Test {
	result, err := o.TestRunner.RunTest(ctx, test, o.PodSecurity)
	if err != nil {
		result = convertErrorToStatus(err, "")
	}

	out := v1alpha3.NewTest()
	//TODO: Add timestamp to result when API version updates
	//out.Tstamp = time.Now().Format(time.RFC850)
	out.Spec = test
	out.Status = *result
	return out
}

// selectTests applies an optionally passed selector expression
// against the configured set of tests, returning the selected tests
func (o *Scorecard) selectTests(stage v1alpha3.StageConfiguration) []v1alpha3.TestConfiguration {
	selected := make([]v1alpha3.TestConfiguration, 0)
	for _, test := range stage.Tests {
		if o.Selector == nil || o.Selector.String() == "" || o.Selector.Matches(labels.Set(test.Labels)) {
			// TODO olm manifests check
			selected = append(selected, test)
		}
	}
	return selected
}

func (r FakeTestRunner) Initialize(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
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

func (r FakeTestRunner) Cleanup(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
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
func (r PodTestRunner) RunTest(ctx context.Context, test v1alpha3.TestConfiguration, podSec bool) (*v1alpha3.TestStatus, error) {

	// Create a Pod to run the test
	podDef := getPodDefinition(r.configMapName, test, r)
	if podSec {
		// creating a pod security context to support running in default namespace
		podSecCtx := v1.PodSecurityContext{}
		podSecCtx.RunAsNonRoot = &podSec
		podSecCtx.SeccompProfile = &v1.SeccompProfile{
			Type: v1.SeccompProfileTypeRuntimeDefault,
		}

		// creating a security context to be used by all containers in the pod
		secCtx := v1.SecurityContext{}
		secCtx.RunAsNonRoot = &podSec
		secCtx.AllowPrivilegeEscalation = &[]bool{false}[0]
		secCtx.Capabilities = &v1.Capabilities{
			Drop: []v1.Capability{
				"ALL",
			},
		}

		podDef.Spec.SecurityContext = &podSecCtx

		podDef.Spec.Containers[0].SecurityContext = &secCtx
		podDef.Spec.InitContainers[0].SecurityContext = &secCtx
	}

	if test.Storage.Spec.MountPath.Path != "" {
		addStorageToPod(podDef, test.Storage.Spec.MountPath.Path, r.StorageImage)
	}

	pod, err := r.Client.CoreV1().Pods(r.Namespace).Create(ctx, podDef, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	err = r.waitForTestToComplete(ctx, pod)
	if err != nil {
		return nil, err
	}

	// gather test output if necessary
	if test.Storage.Spec.MountPath.Path != "" {
		err := gatherTestOutput(r, test.Labels["suite"], test.Labels["test"], pod.Name, test.Storage.Spec.MountPath.Path)
		if err != nil {
			return nil, err
		}
	}

	return r.getTestStatus(ctx, pod), nil
}

// RunTest executes a single test
func (r FakeTestRunner) RunTest(ctx context.Context, _ v1alpha3.TestConfiguration, _ bool) (result *v1alpha3.TestStatus, err error) {
	select {
	case <-time.After(r.Sleep):
		return r.TestStatus, r.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func ConfigDocLink() string {
	return "https://sdk.operatorframework.io/docs/scorecard/"
}

// waitForTestToComplete waits for a fixed amount of time while
// checking for a test pod to complete
func (r PodTestRunner) waitForTestToComplete(ctx context.Context, p *v1.Pod) (err error) {

	podCheck := wait.ConditionWithContextFunc(func(pctx context.Context) (done bool, err error) {
		var tmp *v1.Pod
		tmp, err = r.Client.CoreV1().Pods(p.Namespace).Get(pctx, p.Name, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Errorf("error getting pod %s %w", p.Name, err)
		}
		for _, s := range tmp.Status.ContainerStatuses {
			if s.Name == "scorecard-test" {
				if s.State.Terminated != nil {
					return true, nil
				}
			}
		}

		return false, nil
	})

	err = wait.PollUntilContextCancel(ctx, 1*time.Second, false, podCheck)
	return err

}

func convertErrorToStatus(err error, log string) *v1alpha3.TestStatus {
	result := v1alpha3.TestResult{}
	result.State = v1alpha3.FailState
	result.Errors = []string{err.Error()}
	result.Log = log
	return &v1alpha3.TestStatus{
		Results: []v1alpha3.TestResult{result},
	}
}
