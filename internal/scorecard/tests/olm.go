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

	scapiv1alpha3 "github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	apivalidation "github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	"github.com/operator-framework/operator-registry/pkg/image/execregistry"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/sirupsen/logrus"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
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
func BundleValidationTest(bundleRoot string, metadata registryutil.LabelsMap) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{}
	r.Name = OLMBundleValidationTest
	r.State = scapiv1alpha3.PassState
	r.Errors = []string{}
	r.Suggestions = []string{}

	defaultOutput := logrus.StandardLogger().Out
	defer logrus.SetOutput(defaultOutput)

	// Log output from the test will be captured in this buffer
	buf := &bytes.Buffer{}
	logger := logrus.WithField("name", "bundle-test")
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(buf)

	// Despite NewRegistry appearing to create a docker image registry, all this function does is return a type
	// that shells out to the docker client binary. Since BundleValidationTest only ever calls ValidateBundleFormat,
	// which does not use the underlying registry, we can use this object as a dummy registry. We shouldn't
	// use the containerd registry because that actually creates an underlying registry.
	// NB(estroz): previously NewImageValidator constructed a docker registry internally, which is what we've done
	// here. However it might be nice to create a mock registry that returns an error if any method is called.
	reg, err := execregistry.NewRegistry(containertools.DockerTool, logger)
	if err != nil {
		// This function should never return an error since it's wrapping the docker client binary in a struct.
		logger.Fatalf("Scorecard: this docker registry error should never occur: %v", err)
	}
	val := registrybundle.NewImageValidator(reg, logger)

	// Validate bundle format.
	if err := val.ValidateBundleFormat(bundleRoot); err != nil {
		r.State = scapiv1alpha3.FailState
		r.Errors = append(r.Errors, err.Error())
	}

	// Since a custom manifests directory may be used, check metadata for its base
	// path. Use the default base path if that label doesn't exist.
	manifestsDir := registrybundle.ManifestsDir
	if value, hasKey := metadata.GetManifestsDir(); hasKey {
		manifestsDir = value
	}

	// Validate bundle content.
	bundle, err := apimanifests.GetBundleFromDir(filepath.Join(bundleRoot, manifestsDir))
	if err != nil {
		r.State = scapiv1alpha3.FailState
		r.Errors = append(r.Errors, err.Error())
		r.Log = buf.String()
		return wrapResult(r)
	}

	objs := []interface{}{bundle, bundle.CSV}
	for _, crd := range bundle.V1CRDs {
		objs = append(objs, crd)
	}
	for _, crd := range bundle.V1beta1CRDs {
		objs = append(objs, crd)
	}
	validationResults := apivalidation.DefaultBundleValidators.Validate(objs...)
	for _, result := range validationResults {
		for _, e := range result.Errors {
			r.Errors = append(r.Errors, e.Error())
			r.State = scapiv1alpha3.FailState
		}

		for _, w := range result.Warnings {
			r.Suggestions = append(r.Suggestions, w.Error())
		}
	}

	r.Log = buf.String()
	return wrapResult(r)
}

// CRDsHaveValidationTest verifies all CRDs have a validation section
func CRDsHaveValidationTest(bundle *apimanifests.Bundle) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{}
	r.Name = OLMCRDsHaveValidationTest
	r.State = scapiv1alpha3.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	crs, err := GetCRs(bundle)
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha3.ErrorState
		return wrapResult(r)
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
			r.State = scapiv1alpha3.ErrorState
			return wrapResult(r)
		}
		crds = append(crds, out)
	}
	r.Log += fmt.Sprintf("Loaded CustomresourceDefinitions: %s\n", crds)

	for _, cr := range crs {
		r = isCRFromCRDApi(cr, crds, r)
	}
	return wrapResult(r)
}

// CRDsHaveResourcesTest verifies CRDs have resources listed in its owned CRDs section
func CRDsHaveResourcesTest(bundle *apimanifests.Bundle) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{}
	r.Name = OLMCRDsHaveResourcesTest
	r.State = scapiv1alpha3.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)

	r.Log += fmt.Sprintf("Loaded ClusterServiceVersion: %s\n", bundle.CSV.GetName())

	return wrapResult(CheckResources(bundle.CSV.Spec.CustomResourceDefinitions, r))
}

// CheckResources verified if the owned CRDs have the resources field.
func CheckResources(crd operatorsv1alpha1.CustomResourceDefinitions,
	r scapiv1alpha3.TestResult) scapiv1alpha3.TestResult {
	for _, description := range crd.Owned {
		if len(description.Resources) == 0 {
			r.State = scapiv1alpha3.FailState
			r.Errors = append(r.Errors, "Owned CRDs do not have resources specified")
			return r
		}
	}
	return r
}

// SpecDescriptorsTest verifies all spec fields have descriptors
func SpecDescriptorsTest(bundle *apimanifests.Bundle) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{}
	r.Name = OLMSpecDescriptorsTest
	r.State = scapiv1alpha3.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	r = checkCSVDescriptors(bundle, r, specDescriptor)
	return wrapResult(r)
}

// StatusDescriptorsTest verifies all CRDs have status descriptors
func StatusDescriptorsTest(bundle *apimanifests.Bundle) scapiv1alpha3.TestStatus {
	r := scapiv1alpha3.TestResult{}
	r.Name = OLMStatusDescriptorsTest
	r.State = scapiv1alpha3.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	r = checkCSVDescriptors(bundle, r, statusDescriptor)
	return wrapResult(r)
}

func checkCSVDescriptors(bundle *apimanifests.Bundle, r scapiv1alpha3.TestResult,
	descriptor string) scapiv1alpha3.TestResult {

	r.Log += fmt.Sprintf("Loaded ClusterServiceVersion: %s\n", bundle.CSV.GetName())

	crs, err := GetCRs(bundle)
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		r.State = scapiv1alpha3.ErrorState
		return r
	}
	r.Log += fmt.Sprintf("Loaded %d Custom Resources from alm-examples\n", len(crs))

	// if the descriptor is status then check the status
	// otherwise check the spec
	for _, cr := range crs {
		if descriptor == statusDescriptor {
			r = checkOwnedCSVStatusDescriptor(cr, bundle.CSV, r)
		} else {
			r = checkOwnedCSVSpecDescriptors(cr, bundle.CSV, r)
		}
	}
	return r
}

func checkOwnedCSVStatusDescriptor(cr unstructured.Unstructured, csv *operatorsv1alpha1.ClusterServiceVersion,
	r scapiv1alpha3.TestResult) scapiv1alpha3.TestResult {

	var crdDescription *operatorsv1alpha1.CRDDescription

	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == cr.GetKind() && owned.Version == cr.GroupVersionKind().Version {
			crdDescription = &owned
			break
		}
	}

	if crdDescription == nil {
		msg := fmt.Sprintf("Failed to find an owned CRD for CR %s with GVK %s", cr.GetName(), cr.GroupVersionKind().String())
		r.Errors = append(r.Errors, msg)
		r.State = scapiv1alpha3.FailState
		return r
	}

	hasStatusDefinition := false
	if cr.Object["status"] != nil {
		// Ensure that has no empty keys
		hasStatusDefinition = len(cr.Object["status"].(map[string]interface{})) > 0
	}

	if !hasStatusDefinition {
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("%s does not have status spec. Note that "+
			"all objects that represent a physical resource whose state may vary from the user's desired "+
			"intent SHOULD have a spec and a status. "+
			"More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status", crdDescription.Name))
	}

	if hasStatusDefinition && len(crdDescription.StatusDescriptors) == 0 {
		r.Errors = append(r.Errors, fmt.Sprintf("%s does not have a status descriptor", crdDescription.Name))
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("add status descriptor for the crd %s. "+
			"If your project is built using Golang you can use the csv markers. "+
			"More info: https://sdk.operatorframework.io/docs/building-operators/golang/references/markers/", crdDescription.Name))
		r.State = scapiv1alpha3.FailState
	}

	return r
}

// TODO This is the validation we did in v1, but it looks like it only validates fields that
// are in the example CRs, if you have a field in your CRD that isn't present in one of your examples,
// I don't think it will be validated.
func checkOwnedCSVSpecDescriptors(cr unstructured.Unstructured, csv *operatorsv1alpha1.ClusterServiceVersion,
	r scapiv1alpha3.TestResult) scapiv1alpha3.TestResult {
	if cr.Object[specDescriptor] == nil {
		r.State = scapiv1alpha3.FailState
		return r
	}

	block := cr.Object[specDescriptor].(map[string]interface{})

	var crd *operatorsv1alpha1.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == cr.GetKind() && owned.Version == cr.GroupVersionKind().Version {
			crd = &owned
			break
		}
	}

	if crd == nil {
		msg := fmt.Sprintf("Failed to find an owned CRD for CR %s with GVK %s", cr.GetName(), cr.GroupVersionKind().String())
		r.Errors = append(r.Errors, msg)
		r.State = scapiv1alpha3.FailState
		return r
	}

	for key := range block {
		for _, specDesc := range crd.SpecDescriptors {
			if specDesc.Path == key {
				delete(block, key)
				break
			}
		}
	}

	for key := range block {
		r.Errors = append(r.Errors, fmt.Sprintf("%s does not have a %s descriptor", key, specDescriptor))
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("Add a %s descriptor for %s", specDescriptor, key))
		r.State = scapiv1alpha3.FailState
	}
	return r
}

// hasVersion checks if a CRD contains a specified version in a case insensitive manner
func hasVersion(version string, crdVersion apiextv1.CustomResourceDefinitionVersion) bool {
	return strings.EqualFold(version, crdVersion.Name)
}

func hasKind(kind1, kind2 string, r scapiv1alpha3.TestResult) bool {

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
	r scapiv1alpha3.TestResult) scapiv1alpha3.TestResult {

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
				r.State = scapiv1alpha3.FailState
				return r
			}
		}
	}
	return r
}

func wrapResult(r scapiv1alpha3.TestResult) scapiv1alpha3.TestStatus {
	return scapiv1alpha3.TestStatus{
		Results: []scapiv1alpha3.TestResult{r},
	}
}
