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

package v1alpha3

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

// TestSpec contains the spec details of an individual scorecard test
type TestSpec struct {
	// Image is the name of the testimage
	Image string `json:"image"`
	// Entrypoint is list of commands and arguments passed to the test image
	Entrypoint []string `json:"entrypoint,omitempty"`
	// Labels that further describe the test and enable selection
	Labels map[string]string `json:"labels,omitempty"`
}

// TestResult contains the results of an individual scorecard test
type TestResult struct {
	// Name is the name of the test
	Name string `json:"name,omitempty"`
	// Log holds a log produced from the test (if applicable)
	Log string `json:"log,omitempty"`
	// State is the final state of the test
	State State `json:"state"`
	// Errors is a list of the errors that occurred during the test (this can include both fatal and non-fatal errors)
	Errors []string `json:"errors,omitempty"`
	// Suggestions is a list of suggestions for the user to improve their score (if applicable)
	Suggestions []string `json:"suggestions,omitempty"`
}

// TestStatus contains collection of testResults.
type TestStatus struct {
	Results []TestResult `json:"results,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Test specifies a single test run.
type Test struct {
	metav1.TypeMeta `json:",inline"`
	Spec            TestSpec   `json:"spec,omitempty"`
	Status          TestStatus `json:"status,omitempty"`
}

// TestList is a list of tests.
type TestList struct {
	metav1.TypeMeta `json:",inline"`
	Items           []Test `json:"items"`
}

func NewTest() Test {
	return Test{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "Test",
		},
	}
}

func NewTestList() TestList {
	return TestList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "TestList",
		},
	}
}

func init() {
	SchemeBuilder.Register(&Test{})
}
