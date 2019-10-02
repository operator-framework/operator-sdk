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
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConvertScorecardOutputV1ToV2(v1ScorecardOutput []scapiv1alpha1.ScorecardOutput) scapiv1alpha2.ScorecardOutputList {

	list := scapiv1alpha2.ScorecardOutputList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScorecardOutput",
			APIVersion: "osdk.openshift.io/v1alpha2",
		},
	}

	items := make([]scapiv1alpha2.ScorecardOutput, 0)
	for _, o := range v1ScorecardOutput {
		output := scapiv1alpha2.ScorecardOutput{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ScorecardOutput",
				APIVersion: "osdk.openshift.io/v1alpha2",
			},
			Log: o.Log,
		}
		output.Results = make([]scapiv1alpha2.ScorecardSuiteResult, 0)
		for _, v1SuiteResult := range o.Results {
			v2SuiteResult := ConvertSuiteResultV1ToV2(v1SuiteResult)
			output.Results = append(output.Results, v2SuiteResult)
		}
		items = append(items, output)
	}

	list.Items = items

	return list
}

func ConvertSuiteResultV1ToV2(v1SuiteResult scapiv1alpha1.ScorecardSuiteResult) scapiv1alpha2.ScorecardSuiteResult {
	output := scapiv1alpha2.ScorecardSuiteResult{
		Name:        v1SuiteResult.Name,
		Description: v1SuiteResult.Description,
		Error:       v1SuiteResult.Error,
		Pass:        v1SuiteResult.Pass,
		Fail:        v1SuiteResult.Fail,
		TotalTests:  v1SuiteResult.TotalTests,
		Log:         v1SuiteResult.Log,
	}

	for _, v1TestResult := range v1SuiteResult.Tests {
		output.Tests = append(output.Tests, ConvertTestResultV1ToV2(v1TestResult))
	}
	return output
}

func ConvertTestResultV1ToV2(v1TestResult scapiv1alpha1.ScorecardTestResult) scapiv1alpha2.ScorecardTestResult {
	output := scapiv1alpha2.ScorecardTestResult{
		State:       scapiv1alpha2.FailState,
		Name:        v1TestResult.Name,
		Description: v1TestResult.Description,
	}

	if v1TestResult.EarnedPoints == v1TestResult.MaximumPoints {
		output.State = scapiv1alpha2.PassState
	}

	output.Suggestions = make([]string, len(v1TestResult.Suggestions))
	copy(output.Suggestions, v1TestResult.Suggestions)
	output.Errors = make([]string, len(v1TestResult.Errors))
	copy(output.Errors, v1TestResult.Errors)
	return output
}
