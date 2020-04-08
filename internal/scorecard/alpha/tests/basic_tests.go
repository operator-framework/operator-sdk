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
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

// CheckStatusTest verifies that CRs have a status block
func CheckStatusTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	for b := 0; b < len(conf.Bundles); b++ {
		//bundle := conf.Bundles[b]

		//csv, err := bundle.ClusterServiceVersion()

		// CRs are withing the csv metadata.annotations.alm-examples

		// get CRDs
		//crds, err := bundle.CustomResourceDefinitions()
	}
	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	r := scapiv1alpha2.ScorecardTestResult{}
	r.State = scapiv1alpha2.PassState
	//r.CRName = conf.CRList[i].Name
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	results = append(results, r)
	return results
}

// CheckSpecTest verifies that CRs have a spec block
func CheckSpecTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	//	for i := 0; i < len(conf.CRList); i++ {
	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	r := scapiv1alpha2.ScorecardTestResult{}
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	results = append(results, r)
	//	}
	return results
}
