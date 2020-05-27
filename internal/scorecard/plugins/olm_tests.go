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

package scplugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	apivalidation "github.com/operator-framework/api/pkg/validation"
	schelpers "github.com/operator-framework/operator-sdk/internal/scorecard/helpers"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"github.com/sirupsen/logrus"

	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	statusDescriptor string = "status"
	specDescriptor   string = "spec"
)

// OLMTestConfig contains all variables required by the OLMTest tests
type OLMTestConfig struct {
	Client   client.Client
	CR       *unstructured.Unstructured
	CSV      *olmapiv1alpha1.ClusterServiceVersion
	CRDsDir  string
	ProxyPod *v1.Pod
	Bundle   string
}

// Test Defintions

// BundleValidationTest is a scorecard test that validates a bundle
type BundleValidationTest struct {
	schelpers.TestInfo
	OLMTestConfig
}

// NewBundleValidationTest returns a new BundleValidationTest object
func NewBundleValidationTest(conf OLMTestConfig) *BundleValidationTest {
	return &BundleValidationTest{
		OLMTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Bundle Validation Test",
			Description: "Validates bundle contents",
			Labels: map[string]string{necessityKey: requiredNecessity, suiteKey: olmSuiteName,
				testKey: getStructShortName(BundleValidationTest{})},
		},
	}
}

// CRDsHaveValidationTest is a scorecard test that verifies that all CRDs have a validation section
type CRDsHaveValidationTest struct {
	schelpers.TestInfo
	OLMTestConfig
}

// NewCRDsHaveValidationTest returns a new CRDsHaveValidationTest object
func NewCRDsHaveValidationTest(conf OLMTestConfig) *CRDsHaveValidationTest {
	return &CRDsHaveValidationTest{
		OLMTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Provided APIs have validation",
			Description: "All CRDs have an OpenAPI validation subsection",
			Labels: map[string]string{necessityKey: requiredNecessity, suiteKey: olmSuiteName,
				testKey: getStructShortName(CRDsHaveValidationTest{})},
		},
	}
}

// CRDsHaveResourcesTest is a scorecard test that verifies that the CSV lists used resources in its owned CRDs section
type CRDsHaveResourcesTest struct {
	schelpers.TestInfo
	OLMTestConfig
}

// NewCRDsHaveResourcesTest returns a new CRDsHaveResourcesTest object
func NewCRDsHaveResourcesTest(conf OLMTestConfig) *CRDsHaveResourcesTest {
	return &CRDsHaveResourcesTest{
		OLMTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Owned CRDs have resources listed",
			Description: "All Owned CRDs contain a resources subsection",
			Labels: map[string]string{necessityKey: requiredNecessity, suiteKey: olmSuiteName,
				testKey: getStructShortName(CRDsHaveResourcesTest{})},
		},
	}
}

// SpecDescriptorsTest is a scorecard test that verifies that all spec fields have descriptors
type SpecDescriptorsTest struct {
	schelpers.TestInfo
	OLMTestConfig
}

// NewSpecDescriptorsTest returns a new SpecDescriptorsTest object
func NewSpecDescriptorsTest(conf OLMTestConfig) *SpecDescriptorsTest {
	return &SpecDescriptorsTest{
		OLMTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Spec fields with descriptors",
			Description: "All spec fields have matching descriptors in the CSV",
			Labels: map[string]string{necessityKey: requiredNecessity, suiteKey: olmSuiteName,
				testKey: getStructShortName(SpecDescriptorsTest{})},
		},
	}
}

// StatusDescriptorsTest is a scorecard test that verifies that all status fields have descriptors
type StatusDescriptorsTest struct {
	schelpers.TestInfo
	OLMTestConfig
}

// NewStatusDescriptorsTest returns a new StatusDescriptorsTest object
func NewStatusDescriptorsTest(conf OLMTestConfig) *StatusDescriptorsTest {
	return &StatusDescriptorsTest{
		OLMTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Status fields with descriptors",
			Description: "All status fields have matching descriptors in the CSV",
			Labels: map[string]string{necessityKey: requiredNecessity, suiteKey: olmSuiteName,
				testKey: getStructShortName(StatusDescriptorsTest{})},
		},
	}
}

func matchKind(kind1, kind2 string) bool {
	singularKind1, err := restMapper.ResourceSingularizer(kind1)
	if err != nil {
		singularKind1 = kind1
		log.Warningf("could not find singular version of %s", kind1)
	}
	singularKind2, err := restMapper.ResourceSingularizer(kind2)
	if err != nil {
		singularKind2 = kind2
		log.Warningf("could not find singular version of %s", kind2)
	}
	return strings.EqualFold(singularKind1, singularKind2)
}

// Test Implentations

// Run - implements Test interface
func (t *BundleValidationTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, CRName: t.CR.GetName(), State: scapiv1alpha2.PassState}

	if t.OLMTestConfig.Bundle == "" {
		res.Errors = append(res.Errors,
			errors.New("unable to find the OLM 'bundle' directory which is required for this test"))
		res.State = scapiv1alpha2.ErrorState
		return res
	}

	// Get the validation API log because it contains
	// validation output that we include into the scorecard test output.
	validationLogOutput := new(bytes.Buffer)
	origOutput := logrus.StandardLogger().Out
	logrus.SetOutput(validationLogOutput)
	defer logrus.SetOutput(origOutput)

	bundle, err := apimanifests.GetBundleFromDir(t.OLMTestConfig.Bundle)
	if err != nil {
		res.Errors = append(res.Errors, err)
		res.State = scapiv1alpha2.ErrorState
		return res
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
			res.Errors = append(res.Errors, &e)
			res.State = scapiv1alpha2.FailState
		}

		for _, w := range result.Warnings {
			res.Suggestions = append(res.Suggestions, w.Error())
		}
	}

	res.Log = validationLogOutput.String()

	return res
}

// Run - implements Test interface
func (t *CRDsHaveValidationTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, CRName: t.CR.GetName(), State: scapiv1alpha2.PassState}
	v1crds, v1beta1crds, err := k8sutil.GetCustomResourceDefinitions(t.CRDsDir)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("failed to get CRDs in %s directory: %v", t.CRDsDir, err))
		res.State = scapiv1alpha2.ErrorState
		return res
	}
	err = t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, err)
		res.State = scapiv1alpha2.ErrorState
		return res
	}

	// check if the CRD matches the testing CR
	gvk := t.CR.GroupVersionKind()
	for _, crd := range v1crds {
		for _, ver := range crd.Spec.Versions {
			// Only check the validation block if the CRD and CR have the same Kind and Version
			if strings.EqualFold(gvk.Version, ver.Name) && matchKind(gvk.Kind, crd.Spec.Names.Kind) {
				checkV1CRDVersion(res, t.CR, ver.Schema)
			}
		}
	}
	for _, crd := range v1beta1crds {
		if len(crd.Spec.Versions) == 0 {
			// Only check the validation block if the CRD and CR have the same Kind and Version
			if strings.EqualFold(gvk.Version, crd.Spec.Version) && matchKind(gvk.Kind, crd.Spec.Names.Kind) {
				checkV1beta1CRDVersion(res, t.CR, crd.Spec.Validation)
			}
		} else {
			for _, ver := range crd.Spec.Versions {
				// Only check the validation block if the CRD and CR have the same Kind and Version
				if strings.EqualFold(gvk.Version, ver.Name) && matchKind(gvk.Kind, crd.Spec.Names.Kind) {
					checkV1beta1CRDVersion(res, t.CR, ver.Schema)
				}
			}
		}
	}
	return res
}

//nolint:dupl
func checkV1CRDVersion(res *schelpers.TestResult, cr *unstructured.Unstructured,
	val *apiextv1.CustomResourceValidation) {

	gvk := cr.GroupVersionKind()
	if val == nil {
		res.Suggestions = append(res.Suggestions, fmt.Sprintf("Add CRD validation for %s/%s", gvk.Kind, gvk.Version))
		return
	}
	failed := false
	if cr.Object["spec"] != nil {
		spec := cr.Object["spec"].(map[string]interface{})
		for key := range spec {
			if _, ok := val.OpenAPIV3Schema.Properties["spec"].Properties[key]; !ok {
				failed = true
				res.Suggestions = append(res.Suggestions,
					fmt.Sprintf("Add CRD validation for spec field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
			}
		}
	}
	if cr.Object["status"] != nil {
		status := cr.Object["status"].(map[string]interface{})
		for key := range status {
			if _, ok := val.OpenAPIV3Schema.Properties["status"].Properties[key]; !ok {
				failed = true
				res.Suggestions = append(res.Suggestions,
					fmt.Sprintf("Add CRD validation for status field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
			}
		}
	}

	if failed {
		res.State = scapiv1alpha2.FailState
	}
}

//nolint:dupl
func checkV1beta1CRDVersion(res *schelpers.TestResult, cr *unstructured.Unstructured,
	val *apiextv1beta1.CustomResourceValidation) {

	gvk := cr.GroupVersionKind()
	if val == nil {
		res.Suggestions = append(res.Suggestions, fmt.Sprintf("Add CRD validation for %s/%s", gvk.Kind, gvk.Version))
		return
	}
	failed := false
	if cr.Object["spec"] != nil {
		spec := cr.Object["spec"].(map[string]interface{})
		for key := range spec {
			if _, ok := val.OpenAPIV3Schema.Properties["spec"].Properties[key]; !ok {
				failed = true
				res.Suggestions = append(res.Suggestions,
					fmt.Sprintf("Add CRD validation for spec field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
			}
		}
	}
	if cr.Object["status"] != nil {
		status := cr.Object["status"].(map[string]interface{})
		for key := range status {
			if _, ok := val.OpenAPIV3Schema.Properties["status"].Properties[key]; !ok {
				failed = true
				res.Suggestions = append(res.Suggestions,
					fmt.Sprintf("Add CRD validation for status field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
			}
		}
	}

	if failed {
		res.State = scapiv1alpha2.FailState
	}
}

// Run - implements Test interface
func (t *CRDsHaveResourcesTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, CRName: t.CR.GetName(), State: scapiv1alpha2.PassState}

	var missingResources []string
	for _, crd := range t.CSV.Spec.CustomResourceDefinitions.Owned {
		gvk := t.CR.GroupVersionKind()
		if strings.EqualFold(crd.Version, gvk.Version) && matchKind(gvk.Kind, crd.Kind) {
			resources, err := getUsedResources(t.ProxyPod)
			if err != nil {
				log.Warningf("getUsedResource failed: %v", err)
			}
			for _, resource := range resources {
				foundResource := false
				for _, listedResource := range crd.Resources {
					if matchKind(resource.Kind, listedResource.Kind) &&
						strings.EqualFold(resource.Version, listedResource.Version) {
						foundResource = true
						break
					}
				}
				if !foundResource {
					missingResources = append(missingResources, fmt.Sprintf("%s/%s",
						resource.Kind, resource.Version))
				}
			}
		}
	}
	if len(missingResources) > 0 {
		res.Suggestions = append(res.Suggestions, fmt.Sprintf("If it would be helpful to an end-user to"+
			" understand or troubleshoot your CR, consider adding resources %v to the resources section for owned"+
			" CRD %s", missingResources, t.CR.GroupVersionKind().Kind))
		res.State = scapiv1alpha2.FailState
	}

	return res
}

func getUsedResources(proxyPod *v1.Pod) ([]schema.GroupVersionKind, error) {
	const api = "api"
	const apis = "apis"
	logs, err := getProxyLogs(proxyPod)
	if err != nil {
		return nil, err
	}
	resources := map[schema.GroupVersionKind]bool{}
	for _, line := range strings.Split(logs, "\n") {
		logMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(line), &logMap)
		if err != nil {
			// it is very common to get "unexpected end of JSON input", so we'll leave this at the debug level
			log.Debugf("could not unmarshal line: %v", err)
			continue
		}
		/*
			There are 6 formats a resource uri can have:
			Cluster-Scoped:
				Collection:      /apis/GROUP/VERSION/KIND
				Individual:      /apis/GROUP/VERSION/KIND/NAME
				Core:            /api/v1/KIND
				Core Individual: /api/v1/KIND/NAME

			Namespaces:
				All Namespaces:          /apis/GROUP/VERSION/KIND (same as cluster collection)
				Collection in Namespace: /apis/GROUP/VERSION/namespaces/NAMESPACE/KIND
				Individual:              /apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME
				Core:                    /api/v1/namespaces/NAMESPACE/KIND
				Core Indiviual:          /api/v1/namespaces/NAMESPACE/KIND/NAME

			These urls are also often appended with options, which are denoted by the '?' symbol
		*/
		if msg, ok := logMap["msg"].(string); !ok || msg != "Request Info" {
			continue
		}
		uri, ok := logMap["uri"].(string)
		if !ok {
			log.Warn("URI type is not string")
			continue
		}
		removedOptions := strings.Split(uri, "?")[0]
		splitURI := strings.Split(removedOptions, "/")
		// first string is empty string ""
		if len(splitURI) < 2 {
			log.Warnf("Invalid URI: \"%s\"", uri)
			continue
		}
		splitURI = splitURI[1:]
		switch len(splitURI) {
		case 3:
			if splitURI[0] == api {
				resources[schema.GroupVersionKind{Version: splitURI[1], Kind: splitURI[2]}] = true
				break
			}
			if splitURI[0] == apis {
				// this situation happens when the client enumerates the available resources of the server
				// Example: "/apis/apps/v1?timeout=32s"
				break
			}
			log.Warnf("Invalid URI: \"%s\"", uri)
		case 4:
			if splitURI[0] == api {
				resources[schema.GroupVersionKind{Version: splitURI[1], Kind: splitURI[2]}] = true
				break
			}
			if splitURI[0] == apis {
				resources[schema.GroupVersionKind{Group: splitURI[1], Version: splitURI[2], Kind: splitURI[3]}] = true
				break
			}
			log.Warnf("Invalid URI: \"%s\"", uri)
		case 5:
			if splitURI[0] == api {
				resources[schema.GroupVersionKind{Version: splitURI[1], Kind: splitURI[4]}] = true
				break
			}
			if splitURI[0] == apis {
				resources[schema.GroupVersionKind{Group: splitURI[1], Version: splitURI[2], Kind: splitURI[3]}] = true
				break
			}
			log.Warnf("Invalid URI: \"%s\"", uri)
		case 6, 7:
			if splitURI[0] == api {
				resources[schema.GroupVersionKind{Version: splitURI[1], Kind: splitURI[4]}] = true
				break
			}
			if splitURI[0] == apis {
				resources[schema.GroupVersionKind{Group: splitURI[1], Version: splitURI[2], Kind: splitURI[5]}] = true
				break
			}
			log.Warnf("Invalid URI: \"%s\"", uri)
		}
	}
	var resourcesArr []schema.GroupVersionKind
	for gvk := range resources {
		resourcesArr = append(resourcesArr, gvk)
	}
	return resourcesArr, nil
}

// Run - implements Test interface
func (t *StatusDescriptorsTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, CRName: t.CR.GetName(), State: scapiv1alpha2.PassState}

	err := t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, err)
		res.State = scapiv1alpha2.ErrorState
		return res
	}

	return checkOwnedCSVDescriptors(t.CR, t.CSV, statusDescriptor, res)
}

// Run - implements Test interface
func (t *SpecDescriptorsTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, CRName: t.CR.GetName(), State: scapiv1alpha2.PassState}
	err := t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, err)
		res.State = scapiv1alpha2.ErrorState
		return res
	}

	return checkOwnedCSVDescriptors(t.CR, t.CSV, specDescriptor, res)
}

func checkOwnedCSVDescriptors(cr *unstructured.Unstructured, csv *olmapiv1alpha1.ClusterServiceVersion,
	descriptor string, res *schelpers.TestResult) *schelpers.TestResult {
	if cr.Object[descriptor] == nil {
		res.State = scapiv1alpha2.FailState
		return res
	}
	block := cr.Object[descriptor].(map[string]interface{})

	var crd *olmapiv1alpha1.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == cr.GetKind() {
			crd = &owned
			break
		}
	}

	if crd == nil {
		res.State = scapiv1alpha2.FailState
		return res
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
		res.Suggestions = append(res.Suggestions, fmt.Sprintf("Add a %s descriptor for %s", descriptor, key))
		res.State = scapiv1alpha2.FailState
	}
	return res
}
