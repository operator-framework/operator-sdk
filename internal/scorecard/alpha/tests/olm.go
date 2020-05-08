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
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	apivalidation "github.com/operator-framework/api/pkg/validation"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/sirupsen/logrus"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
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
func BundleValidationTest(dir string) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMBundleValidationTest
	r.Description = "Validates bundle contents"
	r.State = scapiv1alpha2.PassState
	r.Errors = []string{}
	r.Suggestions = []string{}

	defaultOutput := logrus.StandardLogger().Out
	defer logrus.SetOutput(defaultOutput)

	// Log output from the test will be captured in this buffer
	buf := &bytes.Buffer{}
	logger := logrus.WithField("name", "bundle-test")
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(buf)

	val := registrybundle.NewImageValidator("", logger)

	// Validate bundle format.
	if err := val.ValidateBundleFormat(dir); err != nil {
		r.State = scapiv1alpha2.FailState
		r.Errors = append(r.Errors, err.Error())
	}

	// Validate bundle content.
	manifestsDir := filepath.Join(dir, registrybundle.ManifestsDir)
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		r.State = scapiv1alpha2.FailState
		r.Errors = append(r.Errors, err.Error())
	}

	objs := []interface{}{bundle, bundle.CSV}
	for _, crd := range bundle.V1CRDs {
		objs = append(objs, crd)
	}
	for _, crd := range bundle.V1beta1CRDs {
		objs = append(objs, crd)
	}
	validationResults := apivalidation.AllValidators.Validate(objs...)
	for _, result := range validationResults {
		for _, e := range result.Errors {
			r.Errors = append(r.Errors, e.Error())
			r.State = scapiv1alpha2.FailState
		}

		for _, w := range result.Warnings {
			r.Suggestions = append(r.Suggestions, w.Error())
		}
	}

	r.Log = buf.String()
	return r
}

// CRDsHaveValidationTest verifies all CRDs have a validation section
func CRDsHaveValidationTest(bundle *apimanifests.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMCRDsHaveValidationTest
	r.Description = "All CRDs have an OpenAPI validation subsection"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	crs, err := GetCRs(bundle)
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha2.ErrorState
		return r
	}
	r.Log += fmt.Sprintf("Loaded %d Custom Resources from alm-examples\n", len(crs))

	var crds []*apiextv1.CustomResourceDefinition
	for _, crd := range bundle.V1CRDs {
		crds = append(crds, crd.DeepCopy())
	}
	for _, crd := range bundle.V1beta1CRDs {
		out, err := k8sutil.Convertv1beta1Tov1CustomResourceDefinition(crd)
		if err != nil {
			r.Errors = append(r.Errors, err.Error())
			r.State = scapiv1alpha2.ErrorState
			return r
		}
		crds = append(crds, out)
	}
	r.Log += fmt.Sprintf("Loaded CustomresourceDefinitions: %s\n", crds)

	for _, cr := range crs {
		r = isCRFromCRDApi(cr, crds, r)
	}
	return r
}

// CRDsHaveResourcesTest verifies CRDs have resources listed in its owned CRDs section
func CRDsHaveResourcesTest(bundle *apimanifests.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMCRDsHaveResourcesTest
	r.Description = "All Owned CRDs contain a resources subsection"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	r.Log += fmt.Sprintf("Loaded ClusterServiceVersion: %s\n", bundle.CSV.GetName())

	return CheckResources(bundle.CSV.Spec.CustomResourceDefinitions, r)
}

// CheckResources verified if the owned CRDs have the resources field.
func CheckResources(crd operatorsv1alpha1.CustomResourceDefinitions,
	r scapiv1alpha2.ScorecardTestResult) scapiv1alpha2.ScorecardTestResult {
	for _, description := range crd.Owned {
		if description.Resources == nil || len(description.Resources) == 0 {
			r.State = scapiv1alpha2.FailState
			r.Errors = append(r.Errors, "Owned CRDs do not have resources specified")
			return r
		}
	}
	return r
}

// SpecDescriptorsTest verifies all spec fields have descriptors
func SpecDescriptorsTest(bundle *apimanifests.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMSpecDescriptorsTest
	r.Description = "All spec fields have matching descriptors in the CSV"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	r = checkCSVDescriptors(bundle, r, specDescriptor)
	return r
}

// StatusDescriptorsTest verifies all CRDs have status descriptors
func StatusDescriptorsTest(bundle *apimanifests.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = OLMStatusDescriptorsTest
	r.Description = "All status fields have matching descriptors in the CSV"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	r = checkCSVDescriptors(bundle, r, statusDescriptor)
	return r
}

func checkCSVDescriptors(bundle *apimanifests.Bundle, r scapiv1alpha2.ScorecardTestResult,
	descriptor string) scapiv1alpha2.ScorecardTestResult {

	r.Log += fmt.Sprintf("Loaded ClusterServiceVersion: %s\n", bundle.CSV.GetName())

	crs, err := GetCRs(bundle)
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha2.ErrorState
		return r
	}
	r.Log += fmt.Sprintf("Loaded %d Custom Resources from alm-examples\n", len(crs))

	for _, cr := range crs {
		r = checkOwnedCSVDescriptors(cr, bundle.CSV, descriptor, r)
	}

	return r
}

// TODO This is the validation we did in v1, but it looks like it only validates fields that
// are in the example CRs, if you have a field in your CRD that isn't present in one of your examples,
// I don't think it will be validated.
func checkOwnedCSVDescriptors(cr unstructured.Unstructured, csv *operatorsv1alpha1.ClusterServiceVersion,
	descriptor string, r scapiv1alpha2.ScorecardTestResult) scapiv1alpha2.ScorecardTestResult {

	if cr.Object[descriptor] == nil {
		r.State = scapiv1alpha2.FailState
		return r
	}

	block := cr.Object[descriptor].(map[string]interface{})

	var crd *operatorsv1alpha1.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == cr.GetKind() {
			crd = &owned
			break
		}
	}

	if crd == nil {
		msg := fmt.Sprintf("Failed to find an owned CRD for CR %s with GVK %s", cr.GetName(), cr.GroupVersionKind().String())
		r.Errors = append(r.Errors, msg)
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

// hasVersion checks if a CRD contains a specified version in a case insensitive manner
func hasVersion(version string, crdVersion apiextv1.CustomResourceDefinitionVersion) bool {
	return strings.EqualFold(version, crdVersion.Name)
}

func hasKind(kind1, kind2 string, r scapiv1alpha2.ScorecardTestResult) bool {

	var restMapper meta.DefaultRESTMapper
	singularKind1, err := restMapper.ResourceSingularizer(kind1)
	if err != nil {
		singularKind1 = kind1
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("could not find singular version of %s", kind1))
	}
	singularKind2, err := restMapper.ResourceSingularizer(kind2)
	if err != nil {
		singularKind2 = kind2
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("could not find singular version of %s", kind2))
	}
	return strings.EqualFold(singularKind1, singularKind2)
}

func isCRFromCRDApi(cr unstructured.Unstructured, crds []*apiextv1.CustomResourceDefinition,
	r scapiv1alpha2.ScorecardTestResult) scapiv1alpha2.ScorecardTestResult {

	// check if the CRD matches the testing CR
	for _, crd := range crds {
		gvk := cr.GroupVersionKind()
		// Only check the validation block if the CRD and CR have the same Kind and Version
		for _, version := range crd.Spec.Versions {

			if !hasVersion(gvk.Version, version) || !hasKind(gvk.Kind, crd.Spec.Names.Kind, r) {
				continue
			}

			if version.Schema == nil {
				r.Suggestions = append(r.Suggestions, fmt.Sprintf("Add CRD validation for %s/%s",
					crd.Spec.Names.Kind, version.Name))
				continue
			}
			failed := false
			if cr.Object["spec"] != nil {
				spec := cr.Object["spec"].(map[string]interface{})
				for key := range spec {
					if _, ok := version.Schema.OpenAPIV3Schema.Properties["spec"].Properties[key]; !ok {
						failed = true
						r.Suggestions = append(r.Suggestions,
							fmt.Sprintf("Add CRD validation for spec field `%s` in %s/%s",
								key, gvk.Kind, gvk.Version))
					}
				}
			}
			if cr.Object["status"] != nil {
				status := cr.Object["status"].(map[string]interface{})
				for key := range status {
					if _, ok := version.Schema.OpenAPIV3Schema.Properties["status"].Properties[key]; !ok {
						failed = true
						r.Suggestions = append(r.Suggestions, fmt.Sprintf("Add CRD validation for status"+
							" field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
					}
				}
			}
			if failed {
				r.State = scapiv1alpha2.FailState
				return r
			}
		}
	}
	return r
}
