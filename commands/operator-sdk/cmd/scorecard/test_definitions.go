// Copyright 2019 The Operator-SDK Authors
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

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Type Definitions

type Test interface {
	GetName() string
	GetDescription() string
	IsCumulative() bool
	Run(context.Context) *TestResult
}

type TestResult struct {
	Test          Test
	EarnedPoints  int
	MaximumPoints int
	Suggestions   []string
	Errors        []error
}

type TestInfo struct {
	Name        string
	Description string
	// If a test is set to cumulative, the scores of multiple runs of the same test on separate CRs are added together for the total score.
	// If cumulative is false, if any test failed, the total score is 0/1. Otherwise 1/1.
	Cumulative bool
}

// Any struct that embeds TestInfo only needs to
// implement Run to implement the Test interface
func (i TestInfo) GetName() string        { return i.Name }
func (i TestInfo) GetDescription() string { return i.Description }
func (i TestInfo) IsCumulative() bool     { return i.Cumulative }

type BasicTestConfig struct {
	Client   client.Client
	CR       *unstructured.Unstructured
	ProxyPod *v1.Pod
}

type OLMTestConfig struct {
	Client   client.Client
	CR       *unstructured.Unstructured
	CSV      *olmapiv1alpha1.ClusterServiceVersion
	CRDsDir  string
	ProxyPod *v1.Pod
}

type TestSuite struct {
	TestInfo
	Tests       []Test
	TestResults []*TestResult
	Weights     map[string]float64
}

// Test definitions

type CheckSpecTest struct {
	TestInfo
	BasicTestConfig
}

func NewCheckSpecTest(conf BasicTestConfig) *CheckSpecTest {
	return &CheckSpecTest{
		BasicTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Spec Block Exists",
			Description: "Custom Resource has a Spec Block",
			Cumulative:  false,
		},
	}
}

type CheckStatusTest struct {
	TestInfo
	BasicTestConfig
}

func NewCheckStatusTest(conf BasicTestConfig) *CheckStatusTest {
	return &CheckStatusTest{
		BasicTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Status Block Exists",
			Description: "Custom Resource has a Status Block",
			Cumulative:  false,
		},
	}
}

type WritingIntoCRsHasEffectTest struct {
	TestInfo
	BasicTestConfig
}

func NewWritingIntoCRsHasEffectTest(conf BasicTestConfig) *WritingIntoCRsHasEffectTest {
	return &WritingIntoCRsHasEffectTest{
		BasicTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Writing into CRs has an effect",
			Description: "A CR sends PUT/POST requests to the API server to modify resources in response to spec block changes",
			Cumulative:  false,
		},
	}
}

type CRDsHaveValidationTest struct {
	TestInfo
	OLMTestConfig
}

func NewCRDsHaveValidationTest(conf OLMTestConfig) *CRDsHaveValidationTest {
	return &CRDsHaveValidationTest{
		OLMTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Provided APIs have validation",
			Description: "All CRDs have an OpenAPI validation subsection",
			Cumulative:  true,
		},
	}
}

type CRDsHaveResourcesTest struct {
	TestInfo
	OLMTestConfig
}

func NewCRDsHaveResourcesTest(conf OLMTestConfig) *CRDsHaveResourcesTest {
	return &CRDsHaveResourcesTest{
		OLMTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Owned CRDs have resources listed",
			Description: "All Owned CRDs contain a resources subsection",
			Cumulative:  true,
		},
	}
}

type AnnotationsContainExamplesTest struct {
	TestInfo
	OLMTestConfig
}

func NewAnnotationsContainExamplesTest(conf OLMTestConfig) *AnnotationsContainExamplesTest {
	return &AnnotationsContainExamplesTest{
		OLMTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "CRs have at least 1 example",
			Description: "The CSV's metadata contains an alm-examples section",
			Cumulative:  true,
		},
	}
}

type SpecDescriptorsTest struct {
	TestInfo
	OLMTestConfig
}

func NewSpecDescriptorsTest(conf OLMTestConfig) *SpecDescriptorsTest {
	return &SpecDescriptorsTest{
		OLMTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Spec fields with descriptors",
			Description: "All spec fields have matching descriptors in the CSV",
			Cumulative:  true,
		},
	}
}

type StatusDescriptorsTest struct {
	TestInfo
	OLMTestConfig
}

func NewStatusDescriptorsTest(conf OLMTestConfig) *StatusDescriptorsTest {
	return &StatusDescriptorsTest{
		OLMTestConfig: conf,
		TestInfo: TestInfo{
			Name:        "Status fields with descriptors",
			Description: "All status fields have matching descriptors in the CSV",
			Cumulative:  true,
		},
	}
}

// Test Suite Declarations

func NewBasicTestSuite(conf BasicTestConfig) *TestSuite {
	ts := NewTestSuite(
		"Basic Tests",
		"Test suite that runs basic, functional operator tests",
	)
	ts.AddTest(NewCheckSpecTest(conf), 1.5)
	ts.AddTest(NewCheckStatusTest(conf), 1)
	ts.AddTest(NewWritingIntoCRsHasEffectTest(conf), 1)

	return ts
}

func NewOLMTestSuite(conf OLMTestConfig) *TestSuite {
	ts := NewTestSuite(
		"OLM Tests",
		"Test suite checks if an operator's CSV follows best practices",
	)

	ts.AddTest(NewCRDsHaveValidationTest(conf), 1.25)
	ts.AddTest(NewCRDsHaveResourcesTest(conf), 1)
	ts.AddTest(NewAnnotationsContainExamplesTest(conf), 1)
	ts.AddTest(NewSpecDescriptorsTest(conf), 1)
	ts.AddTest(NewStatusDescriptorsTest(conf), 1)

	return ts
}

// Helper functions

// ResultsPassFail will be used when multiple CRs are supported
func ResultsPassFail(results []TestResult) (earned, max int) {
	for _, result := range results {
		if result.EarnedPoints != result.MaximumPoints {
			return 0, 1
		}
	}
	return 1, 1
}

// ResultsCumulative will be used when multiple CRs are supported
func ResultsCumulative(results []TestResult) (earned, max int) {
	for _, result := range results {
		earned += result.EarnedPoints
		max += result.MaximumPoints
	}
	return earned, max
}

func (ts *TestSuite) AddTest(t Test, weight float64) {
	ts.Tests = append(ts.Tests, t)
	ts.Weights[t.GetName()] = weight
}

func (ts *TestSuite) TotalScore() (score int) {
	floatScore := 0.0
	for _, result := range ts.TestResults {
		if result.MaximumPoints != 0 {
			floatScore += (float64(result.EarnedPoints) / float64(result.MaximumPoints)) * ts.Weights[result.Test.GetName()]
		}
	}
	// scale to a percentage
	addedWeights := 0.0
	for _, weight := range ts.Weights {
		addedWeights += weight
	}
	floatScore = floatScore * (100 / addedWeights)
	return int(floatScore)
}

func (ts *TestSuite) Run(ctx context.Context) {
	for _, test := range ts.Tests {
		ts.TestResults = append(ts.TestResults, test.Run(ctx))
	}
}

func NewTestSuite(name, description string) *TestSuite {
	return &TestSuite{
		TestInfo: TestInfo{
			Name:        name,
			Description: description,
		},
		Weights: make(map[string]float64),
	}
}
