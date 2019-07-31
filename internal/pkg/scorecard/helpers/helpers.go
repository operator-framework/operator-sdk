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

	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These functions should be in the public test definitions file, but they are not complete/stable,
// so we'll keep these here until they get fully implemented

// ResultsPassFail combines multiple test results and returns a single test results
// with 1 maximum point and either 0 or 1 earned points
func ResultsPassFail(results []TestResult) (TestResult, error) {
	var name string
	finalResult := TestResult{}
	if len(results) > 0 {
		name = results[0].Test.GetName()
		// all results have the same test
		finalResult.Test = results[0].Test
		finalResult.MaximumPoints = 1
		finalResult.EarnedPoints = 1
	}
	for _, result := range results {
		if result.Test.IsCumulative() {
			return finalResult, fmt.Errorf("cumulative test passed to ResultsPassFail: name (%s)", result.Test.GetName())
		}
		if result.Test.GetName() != name {
			return finalResult, fmt.Errorf("test name mismatch in ResultsPassFail: %s != %s", result.Test.GetName(), name)
		}
		if result.EarnedPoints != result.MaximumPoints {
			finalResult.EarnedPoints = 0
		}
		finalResult.Suggestions = append(finalResult.Suggestions, result.Suggestions...)
		finalResult.Errors = append(finalResult.Errors, result.Errors...)
	}
	return finalResult, nil
}

// ResultsCumulative takes multiple TestResults and returns a single TestResult with MaximumPoints
// equal to the sum of the MaximumPoints of the input and EarnedPoints as the sum of EarnedPoints
// of the input
func ResultsCumulative(results []TestResult) (TestResult, error) {
	var name string
	finalResult := TestResult{}
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
		finalResult.EarnedPoints += result.EarnedPoints
		finalResult.MaximumPoints += result.MaximumPoints
		finalResult.Suggestions = append(finalResult.Suggestions, result.Suggestions...)
		finalResult.Errors = append(finalResult.Errors, result.Errors...)
	}
	return finalResult, nil
}

// CalculateResult returns a ScorecardSuiteResult with the state and Tests fields set based on a slice of ScorecardTestResults
func CalculateResult(tests []scapiv1alpha1.ScorecardTestResult) scapiv1alpha1.ScorecardSuiteResult {
	scorecardSuiteResult := scapiv1alpha1.ScorecardSuiteResult{}
	scorecardSuiteResult.Tests = tests
	scorecardSuiteResult = UpdateSuiteStates(scorecardSuiteResult)
	return scorecardSuiteResult
}

// TestSuitesToScorecardOutput takes an array of test suites and generates a v1alpha1 ScorecardOutput object with the
// provided suites and log
func TestSuitesToScorecardOutput(suites []TestSuite, log string) scapiv1alpha1.ScorecardOutput {
	test := scapiv1alpha1.ScorecardOutput{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScorecardOutput",
			APIVersion: "osdk.openshift.io/v1alpha1",
		},
		Log: log,
	}
	scorecardSuiteResults := []scapiv1alpha1.ScorecardSuiteResult{}
	for _, suite := range suites {
		results := []scapiv1alpha1.ScorecardTestResult{}
		for _, testResult := range suite.TestResults {
			results = append(results, TestResultToScorecardTestResult(testResult))
		}
		scorecardSuiteResult := CalculateResult(results)
		scorecardSuiteResult.TotalScore = suite.TotalScore()
		scorecardSuiteResult.Name = suite.GetName()
		scorecardSuiteResult.Description = suite.GetDescription()
		scorecardSuiteResult.Log = suite.Log
		scorecardSuiteResults = append(scorecardSuiteResults, scorecardSuiteResult)
	}
	test.Results = scorecardSuiteResults
	return test
}

// TestResultToScorecardTestResult is a helper function for converting from the TestResult type to the ScorecardTestResult type
func TestResultToScorecardTestResult(tr TestResult) scapiv1alpha1.ScorecardTestResult {
	sctr := scapiv1alpha1.ScorecardTestResult{}
	sctr.State = tr.State
	sctr.Name = tr.Test.GetName()
	sctr.Description = tr.Test.GetDescription()
	sctr.EarnedPoints = tr.EarnedPoints
	sctr.MaximumPoints = tr.MaximumPoints
	sctr.Suggestions = tr.Suggestions
	if sctr.Suggestions == nil {
		sctr.Suggestions = []string{}
	}
	stringErrors := []string{}
	for _, err := range tr.Errors {
		stringErrors = append(stringErrors, err.Error())
	}
	sctr.Errors = stringErrors
	return sctr
}

// UpdateState updates the state of a TestResult.
func UpdateState(res scapiv1alpha1.ScorecardTestResult) scapiv1alpha1.ScorecardTestResult {
	if res.State == scapiv1alpha1.ErrorState {
		return res
	}
	if res.EarnedPoints == 0 {
		res.State = scapiv1alpha1.FailState
	} else if res.EarnedPoints < res.MaximumPoints {
		res.State = scapiv1alpha1.PartialPassState
	} else if res.EarnedPoints == res.MaximumPoints {
		res.State = scapiv1alpha1.PassState
	}
	return res
	// TODO: decide what to do if a Test incorrectly sets points (Earned > Max)
}

// UpdateSuiteStates update the state of each test in a suite and updates the count to the suite's states to match
func UpdateSuiteStates(suite scapiv1alpha1.ScorecardSuiteResult) scapiv1alpha1.ScorecardSuiteResult {
	suite.TotalTests = len(suite.Tests)
	// reset all state values
	suite.Error = 0
	suite.Fail = 0
	suite.PartialPass = 0
	suite.Pass = 0
	for idx, test := range suite.Tests {
		suite.Tests[idx] = UpdateState(test)
		switch test.State {
		case scapiv1alpha1.ErrorState:
			suite.Error++
		case scapiv1alpha1.PassState:
			suite.Pass++
		case scapiv1alpha1.PartialPassState:
			suite.PartialPass++
		case scapiv1alpha1.FailState:
			suite.Fail++
		}
	}
	return suite
}

func CombineScorecardOutput(outputs []scapiv1alpha1.ScorecardOutput, log string) scapiv1alpha1.ScorecardOutput {
	output := scapiv1alpha1.ScorecardOutput{
		Log: log,
	}
	for _, item := range outputs {
		output.Results = append(output.Results, item.Results...)
	}
	return output
}
