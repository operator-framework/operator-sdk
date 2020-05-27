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
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

const (
	BasicCheckSpecTest = "basic-check-spec"
)

// CheckSpecTest verifies that CRs have a spec block
func CheckSpecTest(bundle *apimanifests.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = BasicCheckSpecTest
	r.Description = "Custom Resource has a Spec Block"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	crSet, err := GetCRs(bundle)
	if err != nil {
		r.Errors = append(r.Errors, "error getting custom resources")
		r.State = scapiv1alpha2.FailState
		return r
	}

	for _, cr := range crSet {
		if cr.Object["spec"] == nil {
			r.Errors = append(r.Errors, "error spec does not exist")
			r.State = scapiv1alpha2.FailState
			return r
		}
	}

	return r
}
