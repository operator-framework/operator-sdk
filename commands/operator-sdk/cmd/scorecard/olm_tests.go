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
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
func crdsHaveValidation(crdsDir string, runtimeClient client.Client, obj *unstructured.Unstructured) error {
	test := scorecardTest{testType: olmIntegration, name: "Provided APIs have validation"}
	crds, err := getCRDs(crdsDir)
	if err != nil {
		return fmt.Errorf("failed to get CRDs in %s directory: %v", crdsDir, err)
	}
	err = runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	if err != nil {
		return err
	}
	// TODO: we need to make this handle multiple CRs better/correctly
	for _, crd := range crds {
		test.maximumPoints++
		if crd.Spec.Validation == nil {
			scSuggestions = append(scSuggestions, fmt.Sprintf("Add CRD validation for %s/%s", crd.Spec.Names.Kind, crd.Spec.Version))
			continue
		}
		// check if the CRD matches the testing CR
		gvk := obj.GroupVersionKind()
		// Only check the validation block if the CRD and CR have the same Kind and Version
		if !(matchVersion(gvk.Version, crd) && matchKind(gvk.Kind, crd.Spec.Names.Kind)) {
			test.earnedPoints++
			continue
		}
		failed := false
		if obj.Object["spec"] != nil {
			spec := obj.Object["spec"].(map[string]interface{})
			for key := range spec {
				if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"].Properties[key]; !ok {
					failed = true
					scSuggestions = append(scSuggestions, fmt.Sprintf("Add CRD validation for spec field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
				}
			}
		}
		if obj.Object["status"] != nil {
			status := obj.Object["status"].(map[string]interface{})
			for key := range status {
				if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["status"].Properties[key]; !ok {
					failed = true
					scSuggestions = append(scSuggestions, fmt.Sprintf("Add CRD validation for status field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
				}
			}
		}
		if !failed {
			test.earnedPoints++
		}
	}
	scTests = append(scTests, test)
	return nil
}

// crdsHaveResources checks to make sure that all owned CRDs have resources listed
func crdsHaveResources(test *Test, vars ScorecardVars) error {
	score := Score{}
	for _, crd := range vars.csvObj.Spec.CustomResourceDefinitions.Owned {
		score.maximumPoints++
		if len(crd.Resources) > 0 {
			score.earnedPoints++
		}
	}
	test.scores = append(test.scores, score)
	if score.earnedPoints < score.maximumPoints {
		scSuggestions = append(scSuggestions, "Add resources to owned CRDs")
	}
	return nil
}

// annotationsContainExamples makes sure that the CSVs list at least 1 example for the CR
func annotationsContainExamples(test *Test, vars ScorecardVars) error {
	score := Score{maximumPoints: 1}
	if vars.csvObj.Annotations != nil && vars.csvObj.Annotations["alm-examples"] != "" {
		score.earnedPoints = 1
	}
	test.scores = append(test.scores, score)
	if score.earnedPoints == 0 {
		scSuggestions = append(scSuggestions, "Add an alm-examples annotation to your CSV to pass the "+test.name+" test")
	}
	return nil
}

// statusDescriptors makes sure that all status fields found in the created CR has a matching descriptor in the CSV
func statusDescriptors(test *Test, vars ScorecardVars) error {
	score := Score{}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: vars.crObj.GetNamespace(), Name: vars.crObj.GetName()}, vars.crObj)
	if err != nil {
		return err
	}
	if vars.crObj.Object["status"] == nil {
		// what should we do if there is no status block? Maybe some kind of N/A type output?
		return nil
	}
	statusBlock := vars.crObj.Object["status"].(map[string]interface{})
	score.maximumPoints = len(statusBlock)
	var crd *olmapiv1alpha1.CRDDescription
	for _, owned := range vars.csvObj.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == vars.crObj.GetKind() {
			crd = &owned
			break
		}
	}
	if crd == nil {
		return nil
	}
	for key := range statusBlock {
		for _, statDesc := range crd.StatusDescriptors {
			if statDesc.Path == key {
				score.earnedPoints++
				delete(statusBlock, key)
				break
			}
		}
	}
	test.scores = append(test.scores, score)
	for key := range statusBlock {
		scSuggestions = append(scSuggestions, "Add a status descriptor for "+key)
	}
	return nil
}

// specDescriptors makes sure that all spec fields found in the created CR has a matching descriptor in the CSV
func specDescriptors(test *Test, vars ScorecardVars) error {
	score := Score{}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: vars.crObj.GetNamespace(), Name: vars.crObj.GetName()}, vars.crObj)
	if err != nil {
		return err
	}
	if vars.crObj.Object["spec"] == nil {
		// what should we do if there is no spec block? Maybe some kind of N/A type output?
		return nil
	}
	specBlock := vars.crObj.Object["spec"].(map[string]interface{})
	score.maximumPoints = len(specBlock)
	var crd *olmapiv1alpha1.CRDDescription
	for _, owned := range vars.csvObj.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == vars.crObj.GetKind() {
			crd = &owned
			break
		}
	}
	if crd == nil {
		return nil
	}
	for key := range specBlock {
		for _, specDesc := range crd.SpecDescriptors {
			if specDesc.Path == key {
				score.earnedPoints++
				delete(specBlock, key)
				break
			}
		}
	}
	test.scores = append(test.scores, score)
	for key := range specBlock {
		scSuggestions = append(scSuggestions, "Add a spec descriptor for "+key)
	}
	return nil
}
