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
	"fmt"

	olmAPI "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Struct definitions

// ScorecardVars contains all necessary variables for running scorecard tests
type ScorecardVars struct {
	client        *client.Client
	crObj         *unstructured.Unstructured
	csvObj        *olmAPI.ClusterServiceVersion
	proxyPod      *v1.Pod
	timeout       int
	retryInterval int
}

// Score contains the number of earned points and maximum number of points
type Score struct {
	earnedPoints  int
	maximumPoints int
}

// Test defines a scorecard test
type Test struct {
	name        string
	description string
	// by having an array of scores, we can more easily add support for multiple CR
	// testing in a future PR
	scores     []Score
	cumulative bool
	run        func(*Test, ScorecardVars) error
}

// TestSuite defines a suite of scorecard tests
type TestSuite struct {
	name        string
	description string
	tests       []*Test
	// we cannot use Test as a key as it contains a slice (which cannot be a key)
	// use the test name instead
	weights map[string]int
}

// Test definitions
var checkSpecTest = &Test{
	name:        "Spec Block Exists",
	description: "Custom Resource has a Status Block",
	run:         checkSpec,
}
var checkStatTest = &Test{
	name:        "Status Block Exists",
	description: "Custom Resource has a Status Block",
	run:         checkStat,
}
var checkStatusUpdateTest = &Test{
	name:        "Operator actions are reflected in status",
	description: "Custom Resource status section gets updated after the spec block is modified",
	run:         checkStatusUpdate,
}
var writingIntoCRsHasEffectTest = &Test{
	name:        "Writing into CRs has an effect",
	description: "A CR sends PUT/POST requests to the API server to modify resources in response to spec block changes",
	run:         writingIntoCRsHasEffect,
}
var crdsHaveResourcesTest = &Test{
	name:        "Owned CRDs have resources listed",
	description: "All Owned CRDs contain a resources subsection",
	cumulative:  true,
	run:         crdsHaveResources,
}
var annotationsContainExamplesTest = &Test{
	name:        "CRs have at least 1 example",
	description: "The CSV's metadata contains an alm-examples section",
	cumulative:  true,
	run:         annotationsContainExamples,
}
var specDescriptorsTest = &Test{
	name:        "Spec fields with descriptors",
	description: "All spec fields have matching descriptors in the CSV",
	cumulative:  true,
	run:         specDescriptors,
}
var statusDescriptorsTest = &Test{
	name:        "Status fields with descriptors",
	description: "All status fields have matching descriptors in the CSV",
	cumulative:  true,
	run:         statusDescriptors,
}

// Test Suite Declarations

var BasicTests = &TestSuite{
	name:        "Basic Tests",
	description: "Test suite that runs basic, functional operator tests",
	tests:       []*Test{checkSpecTest, checkStatTest, checkStatusUpdateTest, writingIntoCRsHasEffectTest},
	weights: map[string]int{
		checkSpecTest.name:               34,
		checkStatTest.name:               22,
		checkStatusUpdateTest.name:       22,
		writingIntoCRsHasEffectTest.name: 22,
	},
}
var OLMTests = &TestSuite{
	name:        "OLM Tests",
	description: "Test suite checks if an operator's CSV follows best practices",
	tests:       []*Test{crdsHaveResourcesTest, annotationsContainExamplesTest, specDescriptorsTest, statusDescriptorsTest},
	weights: map[string]int{
		crdsHaveResourcesTest.name:          25,
		annotationsContainExamplesTest.name: 25,
		specDescriptorsTest.name:            25,
		statusDescriptorsTest.name:          25,
	},
}

// Helper functions

func scorePassFail(test *Test) Score {
	totalScore := Score{maximumPoints: 1}
	for _, score := range test.scores {
		if score.earnedPoints != score.maximumPoints {
			totalScore.earnedPoints = 0
			return totalScore
		}
	}
	totalScore.earnedPoints = 1
	return totalScore
}

func scoreCumulative(test *Test) Score {
	totalScore := Score{}
	for _, score := range test.scores {
		totalScore.earnedPoints += score.earnedPoints
		totalScore.maximumPoints += score.maximumPoints
	}
	return totalScore
}

func (test *Test) totalScore() Score {
	if test.cumulative {
		return scoreCumulative(test)
	}
	return scorePassFail(test)
}
func (test *Test) execute(vars ScorecardVars) error {
	fmt.Printf("Running %s test\n", test.name)
	return test.run(test, vars)
}

func (suite TestSuite) calculateTotalScore() int {
	score := 0
	for _, test := range suite.tests {
		score += (test.totalScore().earnedPoints / test.totalScore().maximumPoints) * suite.weights[test.name]
	}
	return score
}
