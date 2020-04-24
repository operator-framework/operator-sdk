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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/operator-framework/api/pkg/operators"
	"github.com/operator-framework/operator-registry/pkg/registry"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	OLMBundleValidationTest   = "olm-bundle-validation"
	OLMCRDsHaveValidationTest = "olm-crds-have-validation"
	OLMCRDsHaveResourcesTest  = "olm-crds-have-resources"
	OLMSpecDescriptorsTest    = "olm-spec-descriptors"
	OLMStatusDescriptorsTest  = "olm-status-descriptors"
	statusDescriptor          = "status"
	specDescriptor            = "spec"
)

// BundleValidationTest validates an on-disk bundle
func BundleValidationTest(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMBundleValidationTest
	r.Description = "Validates bundle contents"
	r.State = scapiv1alpha2.PassState
	r.Log = "validation output goes here"
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

// CRDsHaveValidationTest verifies all CRDs have a validation section
func CRDsHaveValidationTest(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMCRDsHaveValidationTest
	r.Description = "All CRDs have an OpenAPI validation subsection"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

// CRDsHaveResourcesTest verifies CRDs have resources listed in its owned CRDs section
func CRDsHaveResourcesTest(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMCRDsHaveResourcesTest
	r.Description = "All Owned CRDs contain a resources subsection"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	return r
}

// SpecDescriptorsTest verifies all spec fields have descriptors
func SpecDescriptorsTest(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMSpecDescriptorsTest
	r.Description = "All spec fields have matching descriptors in the CSV"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha2.ErrorState
		return r
	}
	r.Log += fmt.Sprintf("Loaded ClusterServiceVersion: %s\n", csv.GetName())
	crs, err := getCRsFromCSV(csv.ObjectMeta.Annotations["alm-examples"], csv.GetName())
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha2.ErrorState
		return r
	}
	r.Log += fmt.Sprintf("Loaded %d Custom Resources from alm-examples\n", len(crs))
	apiCSV, err := registryToApiCSV(csv)
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha2.ErrorState
		return r
	}
	for _, cr := range crs {
		r = checkOwnedCSVDescriptors(cr, apiCSV, specDescriptor, r)
	}
	return r
}

// StatusDescriptorsTest verifies all CRDs have status descriptors
func StatusDescriptorsTest(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMStatusDescriptorsTest
	r.Description = "All status fields have matching descriptors in the CSV"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

func getCRsFromCSV(almExamples string, csvName string) ([]unstructured.Unstructured, error) {
	var crs []unstructured.Unstructured
	// Create temporary CR manifests from metadata if one is not provided.
	if almExamples != "" {
		if err := json.Unmarshal([]byte(almExamples), &crs); err != nil {
			return crs, fmt.Errorf("metadata.annotations['alm-examples'] in CSV %s"+
				"incorrectly formatted: %v", csvName, err)
		}
		if len(crs) == 0 {
			return crs, fmt.Errorf("no CRs found in metadata.annotations['alm-examples']"+
				" in CSV %s and cr-manifest config option not set", csvName)
		}
		return crs, nil
	} else {
		return crs, errors.New(
			// TODO can users still pass crs to be validated?
			"cr-manifest config option must be set if CSV has no metadata.annotations['alm-examples']")
	}
	return crs, nil
}

func registryToApiCSV(csv *registry.ClusterServiceVersion) (*operators.ClusterServiceVersion, error) {
	var apiCSV operators.ClusterServiceVersion
	csvBytes, err := json.Marshal(csv)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(csvBytes, &apiCSV)
	if err != nil {
		return nil, err
	}
	return &apiCSV, nil
}

// TODO This is the validation we did in v1, but it looks like it only validates fields that
// are in the example CRs, if you have a field in your CRD that isn't present in one of your examples,
// I don't think it will be validated.
func checkOwnedCSVDescriptors(cr unstructured.Unstructured, csv *operators.ClusterServiceVersion,
	descriptor string, r scapiv1alpha2.ScorecardTestResult) scapiv1alpha2.ScorecardTestResult {

	if cr.Object[descriptor] == nil {
		r.State = scapiv1alpha2.FailState
		return r
	}

	block := cr.Object[descriptor].(map[string]interface{})

	var crd *operators.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == cr.GetKind() {
			crd = &owned
			break
		}
	}

	if crd == nil {
		r.Errors = append(r.Errors, fmt.Sprintf("Failed to find an owned CRD for CR %s with GVK %s", cr.GetName(), cr.GroupVersionKind().String()))
		r.State = scapiv1alpha2.FailState
		return r
	}

	if descriptor == statusDescriptor {
		for key := range block {
			for _, statDesc := range crd.StatusDescriptors {
				if statDesc.Path == key {
					delete(block, key)
					break
				}
			}
		}
	}
	if descriptor == specDescriptor {
		for key := range block {
			for _, specDesc := range crd.SpecDescriptors {
				if specDesc.Path == key {
					delete(block, key)
					break
				}
			}
		}
	}

	for key := range block {
		r.Errors = append(r.Errors, fmt.Sprintf("%s does not have a %s descriptor", key, descriptor))
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("Add a %s descriptor for %s", descriptor, key))
		r.State = scapiv1alpha2.FailState
	}
	return r
}
