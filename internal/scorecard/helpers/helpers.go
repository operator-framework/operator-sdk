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

package schelpers

import (
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

// TestSuitesToScorecardOutput takes an array of test suites and
// generates a v1alpha2 ScorecardOutput object with the provided suites and log
func TestSuitesToScorecardOutput(suites []TestSuite, log string) scapiv1alpha2.ScorecardOutput {
	test := scapiv1alpha2.NewScorecardOutput()
	test.Log = log

	for _, suite := range suites {
		for _, testResult := range suite.TestResults {
			test.Results = append(test.Results, TestResultToScorecardTestResult(testResult))
		}
	}
	return *test
}

// TestResultToScorecardTestResult is a helper function for converting from the TestResult type
// to the ScorecardTestResult type
func TestResultToScorecardTestResult(tr TestResult) scapiv1alpha2.ScorecardTestResult {
	sctr := scapiv1alpha2.ScorecardTestResult{}
	sctr.State = tr.State
	sctr.Name = tr.Test.GetName()
	sctr.Description = tr.Test.GetDescription()
	sctr.Log = tr.Log
	sctr.CRName = tr.CRName
	sctr.Suggestions = tr.Suggestions
	if sctr.Suggestions == nil {
		sctr.Suggestions = []string{}
	}
	stringErrors := []string{}
	for _, err := range tr.Errors {
		stringErrors = append(stringErrors, err.Error())
	}
	sctr.Errors = stringErrors
	sctr.Labels = tr.Test.GetLabels()
	return sctr
}
