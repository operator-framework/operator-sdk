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
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These functions should be in the public test definitions file, but they are not complete/stable,
// so we'll keep these here until they get fully implemented

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

// CalculateResult returns a ScorecardSuiteResult with the state and Tests fields set based on a slice of ScorecardTestResults
func CalculateResult(tests []scapiv1alpha1.ScorecardTestResult) scapiv1alpha1.ScorecardSuiteResult {
	scorecardSuiteResult := scapiv1alpha1.ScorecardSuiteResult{}
	scorecardSuiteResult.Tests = tests
	for _, test := range scorecardSuiteResult.Tests {
		scorecardSuiteResult.TotalTests++
		switch test.State {
		case scapiv1alpha1.ErrorState:
			scorecardSuiteResult.Error++
		case scapiv1alpha1.PassState:
			scorecardSuiteResult.Pass++
		case scapiv1alpha1.PartialPassState:
			scorecardSuiteResult.PartialPass++
		case scapiv1alpha1.FailState:
			scorecardSuiteResult.Fail++
		}
	}
	return scorecardSuiteResult
}

// TestSuitesToScorecardOutput takes an array of test suites and generates a v1alpha1 ScorecardOutput object with the
// provided name, description, and log
func TestSuitesToScorecardOutput(suites []*TestSuite, log string) *scapiv1alpha1.ScorecardOutput {
	test := &scapiv1alpha1.ScorecardOutput{
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
func UpdateState(res TestResult) TestResult {
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
