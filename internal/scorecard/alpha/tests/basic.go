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
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	scapiv1alpha3 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	BasicCheckSpecTest              = "basic-check-spec"
	BasicCheckSelfRegisteredCRDTest = "basic-self-registered-crd"
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

// CheckSelfRegisteredCRD checks if the CRDs shipped in the package bundle are
// referenced in the CSV, to motify users that they are self-registering CRDs.
// This test only adds suggestions to the scorecard out, and does not change
// the test result status.
func CheckSelfRegisteredCRDTest(bundle *apimanifests.Bundle) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{
		Name:        BasicCheckSelfRegisteredCRDTest,
		Errors:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	ownedCRD := bundle.CSV.Spec.CustomResourceDefinitions.Owned
	selfRegCRDs := make([]string, 0)

	for _, inBundleCRD := range bundle.V1CRDs {
		if !hasCRD(ownedCRD, inBundleCRD.GetName()) {
			selfRegCRDs = append(selfRegCRDs, inBundleCRD.Name)
		}
	}

	for _, inBundleCRD := range bundle.V1beta1CRDs {
		if !hasCRD(ownedCRD, inBundleCRD.GetName()) {
			selfRegCRDs = append(selfRegCRDs, inBundleCRD.Name)
		}
	}

	if len(selfRegCRDs) != 0 {
		var warning strings.Builder
		warning.WriteString("The following CRDs are present in the bundle, but not" +
			"referenced in CSV: ")
		for _, crd := range selfRegCRDs {
			warning.WriteString(crd + " ")
		}
		r.Suggestions = append(r.Suggestions, warning.String())
	}

	return scapiv1alpha3.TestStatus{
		Results: []scapiv1alpha3.TestResult{r},
	}
}

func hasCRD(ownedCRD []v1alpha1.CRDDescription, crd string) bool {
	for _, val := range ownedCRD {
		if strings.Compare(val.Name, crd) == 0 {
			return true
		}
	}
	return false
}
