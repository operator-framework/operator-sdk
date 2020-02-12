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
	// NotRun occurs when a user specifies the --list flag
	NotRunState State = ""
	// PassState occurs when a Test's ExpectedPoints == MaximumPoints.
	PassState State = "pass"
	// FailState occurs when a Test's ExpectedPoints == 0.
	FailState State = "fail"
	// ErrorState occurs when a Test encounters a fatal error and the reported points should not be considered.
	ErrorState State = "error"
)

// ScorecardTestResult contains the results of an individual scorecard test.
type ScorecardTestResult struct {
	// Name is the name of the test
	Name string `json:"name"`
	// Description describes what the test does
	Description string `json:"description"`
	// Labels that further describe the test and enable selection
	Labels map[string]string `json:"labels,omitempty"`
	// State is the final state of the test
	State State `json:"state,omitempty"`
	// Errors is a list of the errors that occurred during the test (this can include both fatal and non-fatal errors)
	Errors []string `json:"errors,omitempty"`
	// Suggestions is a list of suggestions for the user to improve their score (if applicable)
	Suggestions []string `json:"suggestions,omitempty"`
	// Log holds a log produced from the test (if applicable)
	Log string `json:"log,omitempty"`
	// CRName holds the CR name this test was run with (if applicable)
	CRName string `json:"crname,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScorecardOutput is the schema for the scorecard API
type ScorecardOutput struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Log contains the scorecard's log.
	Log string `json:"log"`
	// Results is an array of ScorecardTestResult for the current scorecard run.
	Results []ScorecardTestResult `json:"results"`
}

func NewScorecardOutput() *ScorecardOutput {
	return &ScorecardOutput{
		// The TypeMeta is mandatory because it is used to distinguish the versions (v1alpha1 and v1alpha2)
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScorecardOutput",
			APIVersion: "osdk.openshift.io/v1alpha2",
		},
	}
}

func init() {
	SchemeBuilder.Register(&ScorecardOutput{})
}
