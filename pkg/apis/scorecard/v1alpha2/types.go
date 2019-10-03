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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// State is a type used to indicate the result state of a Test.
type State string

const (
	// PassState occurs when a Test's ExpectedPoints == MaximumPoints.
	PassState State = "pass"
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
	// Fail is the number of tests that ended in the Fail state
	Fail int `json:"fail"`
	// TotalTests is the total number of tests run in this suite
	TotalTests int `json:"totalTests"`
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
	// FailedRequiredTests is the number of required tests run that failed
	FailedRequiredTests int `json:"failedRequiredTests"`
	// RequiredTestStatus is a description of the required tests status
	RequiredTestStatus string `json:"requiredTestStatus"`
}

func init() {
	SchemeBuilder.Register(&ScorecardOutput{})
}
