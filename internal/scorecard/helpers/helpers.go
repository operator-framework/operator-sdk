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
	"fmt"

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

// TestResultToScorecardTestResult is a helper function for converting from the TestResult type to the ScorecardTestResult type
func TestResultToScorecardTestResult(tr TestResult) scapiv1alpha2.ScorecardTestResult {
	sctr := scapiv1alpha2.ScorecardTestResult{}
	sctr.State = tr.State
	sctr.Name = tr.Test.GetName()
	sctr.Description = tr.Test.GetDescription()
	sctr.Log = tr.Log
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

// ResultsCumulative takes multiple TestResults and returns a single TestResult with MaximumPoints
// equal to the sum of the MaximumPoints of the input and EarnedPoints as the sum of EarnedPoints
// of the input
func ResultsCumulative(results []TestResult) (TestResult, error) {
	var name string
	var failFound bool
	finalResult := TestResult{
		State: scapiv1alpha2.PassState,
	}
	if len(results) > 0 {
		name = results[0].Test.GetName()
		// all results have the same test
		finalResult.Test = results[0].Test
	}
	for _, result := range results {
		if !result.Test.IsCumulative() {
			return finalResult, fmt.Errorf("non-cumulative test passed to ResultsCumulative: name (%s)", result.Test.GetName())
		}
		if result.Test.GetName() != name {
			return finalResult, fmt.Errorf("test name mismatch in ResultsCumulative: %s != %s", result.Test.GetName(), name)
		}
		finalResult.Suggestions = append(finalResult.Suggestions, result.Suggestions...)
		finalResult.Errors = append(finalResult.Errors, result.Errors...)
		if result.State != scapiv1alpha2.PassState {
			failFound = true
		}
	}
	if failFound {
		finalResult.State = scapiv1alpha2.FailState
	}
	return finalResult, nil
}

// ResultsPassFail combines multiple test results and returns a
// single test result
func ResultsPassFail(results []TestResult) (TestResult, error) {
	var name string
	var failFound bool
	finalResult := TestResult{
		State: scapiv1alpha2.PassState,
	}
	if len(results) > 0 {
		name = results[0].Test.GetName()
		// all results have the same test
		finalResult.Test = results[0].Test
	}
	for _, result := range results {
		if result.Test.IsCumulative() {
			return finalResult, fmt.Errorf("cumulative test passed to ResultsPassFail: name (%s)", result.Test.GetName())
		}
		if result.Test.GetName() != name {
			return finalResult, fmt.Errorf("test name mismatch in ResultsPassFail: %s != %s", result.Test.GetName(), name)
		}
		finalResult.Suggestions = append(finalResult.Suggestions, result.Suggestions...)
		finalResult.Errors = append(finalResult.Errors, result.Errors...)
		finalResult.Log = result.Log
		if result.State == scapiv1alpha2.FailState {
			failFound = true
		}
	}

	if failFound {
		finalResult.State = scapiv1alpha2.FailState
	}
	return finalResult, nil
}
