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

	olmAPI "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

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
	var crd *olmAPI.CRDDescription
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
	var crd *olmAPI.CRDDescription
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
