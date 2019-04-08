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

// CalculateStates updates the state fields of the JSONOut based on the TestResults in out.Tests
func CalculateStates(out *scapiv1alpha1.ScorecardResult) {
	out.Error = 0
	out.Pass = 0
	out.PartialPass = 0
	out.Fail = 0
	out.TotalTests = 0
	for _, test := range out.Tests {
		out.TotalTests++
		switch test.State {
		case scapiv1alpha1.ErrorState:
			out.Error++
		case scapiv1alpha1.PassState:
			out.Pass++
		case scapiv1alpha1.PartialPassState:
			out.PartialPass++
		case scapiv1alpha1.FailState:
			out.Fail++
		}
	}
}

// TestSuitesToScorecardTest takes an array of test suites and generates a v1alpha1 ScorecardTest object with the
// provided name, description, and log
func TestSuitesToScorecardTest(suites []*TestSuite, name, description, log string) *scapiv1alpha1.ScorecardTest {
	test := &scapiv1alpha1.ScorecardTest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scorecard",
			APIVersion: "scorecard/v1alpha1",
		},
		Spec: &scapiv1alpha1.ScorecardTestSpec{
			Name:        name,
			Description: description,
		},
		Status: &scapiv1alpha1.ScorecardTestResult{
			Log: log,
		},
	}
	scorecardResults := []scapiv1alpha1.ScorecardResult{}
	for _, suite := range suites {
		scorecardResult := scapiv1alpha1.ScorecardResult{}
		results := []*scapiv1alpha1.JSONTestResult{}
		for _, testResult := range suite.TestResults {
			results = append(results, TestResultToJSONTestResult(testResult))
		}
		scorecardResult.Tests = results
		scorecardResult.TotalScore = suite.TotalScore()
		CalculateStates(&scorecardResult)
		scorecardResults = append(scorecardResults, scorecardResult)
	}
	test.Status.Results = scorecardResults
	return test
}

// TestResultToJSONTestResult is a helper function for converting from the TestResult type to the JSONTestResult type
func TestResultToJSONTestResult(tr *TestResult) *scapiv1alpha1.JSONTestResult {
	jtr := scapiv1alpha1.JSONTestResult{}
	jtr.State = tr.State
	jtr.Name = tr.Test.GetName()
	jtr.Description = tr.Test.GetDescription()
	jtr.EarnedPoints = tr.EarnedPoints
	jtr.MaximumPoints = tr.MaximumPoints
	jtr.Suggestions = tr.Suggestions
	if jtr.Suggestions == nil {
		jtr.Suggestions = []string{}
	}
	stringErrors := []string{}
	for _, err := range tr.Errors {
		stringErrors = append(stringErrors, err.Error())
	}
	jtr.Errors = stringErrors
	return &jtr
}

// UpdateState updates the state of a TestResult.
func (res *TestResult) UpdateState() {
	if res.State == scapiv1alpha1.ErrorState {
		return
	}
	if res.EarnedPoints == 0 {
		res.State = scapiv1alpha1.FailState
	} else if res.EarnedPoints < res.MaximumPoints {
		res.State = scapiv1alpha1.PartialPassState
	} else if res.EarnedPoints == res.MaximumPoints {
		res.State = scapiv1alpha1.PassState
	}
	// TODO: decide what to do if a Test incorrectly sets points (Earned > Max)
}
