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

package tests

import (
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	scapiv1alpha3 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	BasicCheckSpecTest = "basic-check-spec"
)

// CheckSpecTest verifies that CRs have a spec block
func CheckSpecTest(bundle *apimanifests.Bundle) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{
		Name:        BasicCheckSpecTest,
		State:       scapiv1alpha3.PassState,
		Errors:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	crSet, err := GetCRs(bundle)
	if err != nil {
		r.Errors = append(r.Errors, "error getting custom resources")
		r.State = scapiv1alpha3.FailState
	}

	return scapiv1alpha3.TestStatus{
		Results: []scapiv1alpha3.TestResult{checkSpec(crSet, r)},
	}
}

func checkSpec(crSet []unstructured.Unstructured,
	res scapiv1alpha3.TestResult) scapiv1alpha3.TestResult {
	for _, cr := range crSet {
		if cr.Object["spec"] == nil {
			res.Errors = append(res.Errors, "error spec does not exist")
			res.State = scapiv1alpha3.FailState
			return res
		}
	}
	return res
}
