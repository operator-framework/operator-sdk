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

// BundleValidationTest validates an on-disk bundle
func BundleValidationTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	r := scapiv1alpha2.ScorecardTestResult{}
	r.State = scapiv1alpha2.PassState
	r.Log = "validation output goes here"
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	results = append(results, r)
	return results
}

// CRDsHaveValidationTest verifies all CRDs have a validation section
func CRDsHaveValidationTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	bundle := conf.Bundles[0]
	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	r := scapiv1alpha2.ScorecardTestResult{}
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	crds, err := bundle.CustomResourceDefinitions()
	if err != nil {
	}
	for i := 0; i < len(crds); i++ {
		r.Errors = append(r.Errors, "todo")
		r.Suggestions = append(r.Suggestions, "todo")
	}
	results = append(results, r)
	return results
}

// CRDsHaveResourcesTest verifies CRDs have resources listed in its owned CRDs section
func CRDsHaveResourcesTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	r := scapiv1alpha2.ScorecardTestResult{}
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	bundle := conf.Bundles[0]

	crds, err := bundle.CustomResourceDefinitions()
	if err != nil {
	}

	for i := 0; i < len(crds); i++ {
		r.Errors = append(r.Errors, "todo")
		r.Suggestions = append(r.Suggestions, "todo")
	}
	results = append(results, r)
	return results
}

// SpecDescriptorsTest verifies all spec fields have descriptors
func SpecDescriptorsTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	bundle := conf.Bundles[0]
	crds, err := bundle.CustomResourceDefinitions()
	if err != nil {
	}

	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	for i := 0; i < len(crds); i++ {
		r := scapiv1alpha2.ScorecardTestResult{}
		r.State = scapiv1alpha2.PassState
		r.Errors = make([]string, 0)
		r.Suggestions = make([]string, 0)
		results = append(results, r)
	}
	return results
}

// StatusDescriptorsTest verifies all CRDs have status descriptors
func StatusDescriptorsTest(conf TestConfig) []scapiv1alpha2.ScorecardTestResult {
	bundle := conf.Bundles[0]
	crds, err := bundle.CustomResourceDefinitions()
	if err != nil {
	}
	results := make([]scapiv1alpha2.ScorecardTestResult, 0)
	for i := 0; i < len(crds); i++ {
		r := scapiv1alpha2.ScorecardTestResult{}
		r.State = scapiv1alpha2.PassState
		r.Errors = make([]string, 0)
		r.Suggestions = make([]string, 0)
		results = append(results, r)
	}
	return results
}
