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

package scorecard

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

func getCRDs(crdsDir string) ([]apiextv1beta1.CustomResourceDefinition, error) {
	files, err := ioutil.ReadDir(crdsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read deploy directory: (%v)", err)
	}
	crds := []apiextv1beta1.CustomResourceDefinition{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "crd.yaml") {
			obj, err := yamlToUnstructured(filepath.Join(scaffold.CRDsDir, file.Name()))
			if err != nil {
				return nil, err
			}
			crd, err := unstructuredToCRD(obj)
			if err != nil {
				return nil, err
			}
			crds = append(crds, *crd)
		}
	}
	return crds, nil
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

// matchVersion checks if a CRD contains a specified version in a case insensitive manner
func matchVersion(version string, crd apiextv1beta1.CustomResourceDefinition) bool {
	if strings.EqualFold(version, crd.Spec.Version) {
		return true
	}
	// crd.Spec.Version is deprecated, so check in crd.Spec.Versions as well
	for _, currVer := range crd.Spec.Versions {
		if strings.EqualFold(version, currVer.Name) {
			return true
		}
	}
	return false
}

// crdsHaveValidation makes sure that all CRDs have a validation block
func (t *CRDsHaveValidationTest) Run(ctx context.Context) *TestResult {
	res := &TestResult{Test: t}
	crds, err := getCRDs(t.CRDsDir)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("failed to get CRDs in %s directory: %v", t.CRDsDir, err))
		return res
	}
	err = t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, err)
		return res
	}
	// TODO: we need to make this handle multiple CRs better/correctly
	for _, crd := range crds {
		res.MaximumPoints++
		if crd.Spec.Validation == nil {
			res.Suggestions = append(res.Suggestions, fmt.Sprintf("Add CRD validation for %s/%s", crd.Spec.Names.Kind, crd.Spec.Version))
			continue
		}
		// check if the CRD matches the testing CR
		gvk := t.CR.GroupVersionKind()
		// Only check the validation block if the CRD and CR have the same Kind and Version
		if !(matchVersion(gvk.Version, crd) && matchKind(gvk.Kind, crd.Spec.Names.Kind)) {
			res.EarnedPoints++
			continue
		}
		failed := false
		if t.CR.Object["spec"] != nil {
			spec := t.CR.Object["spec"].(map[string]interface{})
			for key := range spec {
				if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"].Properties[key]; !ok {
					failed = true
					res.Suggestions = append(res.Suggestions, fmt.Sprintf("Add CRD validation for spec field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
				}
			}
		}
		if t.CR.Object["status"] != nil {
			status := t.CR.Object["status"].(map[string]interface{})
			for key := range status {
				if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["status"].Properties[key]; !ok {
					failed = true
					res.Suggestions = append(res.Suggestions, fmt.Sprintf("Add CRD validation for status field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
				}
			}
		}
		if !failed {
			res.EarnedPoints++
		}
	}
	return res
}

// crdsHaveResources checks to make sure that all owned CRDs have resources listed
func (t *CRDsHaveResourcesTest) Run(ctx context.Context) *TestResult {
	res := &TestResult{Test: t}
	for _, crd := range t.CSV.Spec.CustomResourceDefinitions.Owned {
		res.MaximumPoints++
		if len(crd.Resources) > 0 {
			res.EarnedPoints++
		}
	}
	if res.EarnedPoints < res.MaximumPoints {
		res.Suggestions = append(res.Suggestions, "Add resources to owned CRDs")
	}
	return res
}

// annotationsContainExamples makes sure that the CSVs list at least 1 example for the CR
func (t *AnnotationsContainExamplesTest) Run(ctx context.Context) *TestResult {
	res := &TestResult{Test: t}
	if t.CSV.Annotations != nil && t.CSV.Annotations["alm-examples"] != "" {
		res.EarnedPoints = 1
	}
	if res.EarnedPoints == 0 {
		res.Suggestions = append(res.Suggestions, "Add an alm-examples annotation to your CSV to pass the "+t.GetName()+" test")
	}
	return res
}

// statusDescriptors makes sure that all status fields found in the created CR has a matching descriptor in the CSV
func (t *StatusDescriptorsTest) Run(ctx context.Context) *TestResult {
	res := &TestResult{Test: t}
	err := t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, err)
		return res
	}
	if t.CR.Object["status"] == nil {
		return res
	}
	statusBlock := t.CR.Object["status"].(map[string]interface{})
	res.MaximumPoints = len(statusBlock)
	var crd *olmapiv1alpha1.CRDDescription
	for _, owned := range t.CSV.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == t.CR.GetKind() {
			crd = &owned
			break
		}
	}
	if crd == nil {
		return res
	}
	for key := range statusBlock {
		for _, statDesc := range crd.StatusDescriptors {
			if statDesc.Path == key {
				res.EarnedPoints++
				delete(statusBlock, key)
				break
			}
		}
	}
	for key := range statusBlock {
		res.Suggestions = append(res.Suggestions, "Add a status descriptor for "+key)
	}
	return res
}

// specDescriptors makes sure that all spec fields found in the created CR has a matching descriptor in the CSV
func (t *SpecDescriptorsTest) Run(ctx context.Context) *TestResult {
	res := &TestResult{Test: t}
	err := t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, err)
		return res
	}
	if t.CR.Object["spec"] == nil {
		return res
	}
	specBlock := t.CR.Object["spec"].(map[string]interface{})
	res.MaximumPoints = len(specBlock)
	var crd *olmapiv1alpha1.CRDDescription
	for _, owned := range t.CSV.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == t.CR.GetKind() {
			crd = &owned
			break
		}
	}
	if crd == nil {
		return res
	}
	for key := range specBlock {
		for _, statDesc := range crd.SpecDescriptors {
			if statDesc.Path == key {
				res.EarnedPoints++
				delete(specBlock, key)
				break
			}
		}
	}
	for key := range specBlock {
		res.Suggestions = append(res.Suggestions, "Add a spec descriptor for "+key)
	}
	return res
}
