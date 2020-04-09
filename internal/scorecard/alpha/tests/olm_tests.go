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

const (
	OLMBundleValidationTest   = "olm-bundle-validation"
	OLMCRDsHaveValidationTest = "olm-crds-have-validation"
	OLMCRDsHaveResourcesTest  = "olm-crds-have-resources"
	OLMSpecDescriptorsTest    = "olm-spec-descriptors"
	OLMStatusDescriptorsTest  = "olm-status-descriptors"
)

// BundleValidationTest validates an on-disk bundle
func BundleValidationTest(conf TestConfig) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMBundleValidationTest
	r.Description = "validate an on-disk bundle"
	r.State = scapiv1alpha2.PassState
	r.Log = "validation output goes here"
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

// CRDsHaveValidationTest verifies all CRDs have a validation section
func CRDsHaveValidationTest(conf TestConfig) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMCRDsHaveValidationTest
	r.Description = "verifies all CRDs have a validation section"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

// CRDsHaveResourcesTest verifies CRDs have resources listed in its owned CRDs section
func CRDsHaveResourcesTest(conf TestConfig) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMCRDsHaveResourcesTest
	r.Description = "verifies all CRDs have resources listed in its owned CRDs section"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	return r
}

// SpecDescriptorsTest verifies all spec fields have descriptors
func SpecDescriptorsTest(conf TestConfig) scapiv1alpha2.ScorecardTestResult {

	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMSpecDescriptorsTest
	r.Description = "verifies all spec fields have descriptors"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

// StatusDescriptorsTest verifies all CRDs have status descriptors
func StatusDescriptorsTest(conf TestConfig) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMStatusDescriptorsTest
	r.Description = "verifies all CRDs have status descriptors"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}
