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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// State is a type used to indicate the result state of a Test.
type State string

const (
	// UnsetState is the default state for a TestResult. It must be updated by UpdateState or by the Test.
	UnsetState State = "unset"
	// PassState occurs when a Test's ExpectedPoints == MaximumPoints.
	PassState State = "pass"
	// PartialPassState occurs when a Test's ExpectedPoints < MaximumPoints and ExpectedPoints > 0.
	PartialPassState State = "partial_pass"
	// FailState occurs when a Test's ExpectedPoints == 0.
	FailState State = "fail"
	// ErrorState occurs when a Test encounters a fatal error and the reported points should not be considered.
	ErrorState State = "error"
)

// ScorecardSuiteResult contains the combined results of a suite of tests.
// +k8s:openapi-gen=true
type ScorecardSuiteResult struct {
	// Name is the name of the test suite
	Name string `json:"name"`
	// Description is a description of the test suite
	Description string `json:"description"`
	// Error is the number of tests that ended in the Error state
	Error int `json:"error"`
	// Pass is the number of tests that ended in the Pass state
	Pass int `json:"pass"`
	// PartialPass is the number of tests that ended in the PartialPass state
	PartialPass int `json:"partialPass"`
	// Fail is the number of tests that ended in the Fail state
	Fail int `json:"fail"`
	// TotalTests is the total number of tests run in this suite
	TotalTests int `json:"totalTests"`
	// TotalScore is the total score of this suite as a percentage
	TotalScore int `json:"totalScorePercent"`
	// Tests is an array containing a json-ified version of the TestResults for the suite
	Tests []ScorecardTestResult `json:"tests"`
	// Log is extra logging information from the scorecard suite/plugin.
	// +optional
	Log string `json:"log"`
}

// ScorecardTestResult contains the results of an individual scorecard test.
// +k8s:openapi-gen=true
type ScorecardTestResult struct {
	// State is the final state of the test
	State State `json:"state"`
	// Name is the name of the test
	Name string `json:"name"`
	// Description describes what the test does
	Description string `json:"description"`
	// EarnedPoints is how many points the test received after running
	EarnedPoints int `json:"earnedPoints"`
	// MaximumPoints is the maximum number of points possible for the test
	MaximumPoints int `json:"maximumPoints"`
	// Suggestions is a list of suggestions for the user to improve their score (if applicable)
	Suggestions []string `json:"suggestions"`
	// Errors is a list of the errors that occured during the test (this can include both fatal and non-fatal errors)
	Errors []string `json:"errors"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScorecardOutput is the schema for the scorecard API
// +k8s:openapi-gen=true
type ScorecardOutput struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Log contains the scorecard's log.
	Log string `json:"log"`
	// Results is an array of ScorecardResult for each suite of the current scorecard run.
	Results []ScorecardSuiteResult `json:"results"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScorecardOutputList contains a list of ScorecardTest
type ScorecardOutputList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScorecardOutput `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScorecardOutput{}, &ScorecardOutputList{})
}
