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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// crdsHaveResources checks to make sure that all owned CRDs have resources listed
func crdsHaveResources(csv *olmAPI.ClusterServiceVersion) {
	test := scorecardTest{testType: olmIntegration, name: "Owned CRDs have resources listed"}
	for _, crd := range csv.Spec.CustomResourceDefinitions.Owned {
		test.maximumPoints++
		if len(crd.Resources) > 0 {
			test.earnedPoints++
		}
	}
	scTests = append(scTests, test)
	if test.earnedPoints == 0 {
		scSuggestions = append(scSuggestions, "Add resources to owned CRDs")
	}
}

// annotationsContainExamples makes sure that the CSVs list at least 1 example for the CR
func annotationsContainExamples(csv *olmAPI.ClusterServiceVersion) {
	test := scorecardTest{testType: olmIntegration, name: "CRs have at least 1 example", maximumPoints: 1}
	if csv.Annotations != nil && csv.Annotations["alm-examples"] != "" {
		test.earnedPoints = 1
	}
	scTests = append(scTests, test)
	if test.earnedPoints == 0 {
		scSuggestions = append(scSuggestions, "Add an alm-examples annotation to your CSV to pass the " + test.name + " test")
	}
}

// statusDescriptors makes sure that all status fields found in the created CR has a matching descriptor in the CSV
func statusDescriptors(csv *olmAPI.ClusterServiceVersion, runtimeClient client.Client, obj *unstructured.Unstructured) error {
	test := scorecardTest{testType: olmIntegration, name: "Status fields with descriptors"}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	if err != nil {
		return err
	}
	if obj.Object["status"] == nil {
		// what should we do if there is no status block? Maybe some kind of N/A type output?
		scTests = append(scTests, test)
		return nil
	}
	statusBlock := obj.Object["status"].(map[string]interface{})
	test.maximumPoints = len(statusBlock)
	var crd *olmAPI.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == obj.GetKind() {
			crd = &owned
			break
		}
	}
	if crd == nil {
		scTests = append(scTests, test)
		return nil
	}
	for key := range statusBlock {
		for _, statDesc := range crd.StatusDescriptors {
			if statDesc.Path == key {
				test.earnedPoints++
				delete(statusBlock, key)
				break
			}
		}
	}
	scTests = append(scTests, test)
	for key := range statusBlock {
		scSuggestions = append(scSuggestions, "Add a status descriptor for "+key)
	}
	return nil
}

// specDescriptors makes sure that all spec fields found in the created CR has a matching descriptor in the CSV
func specDescriptors(csv *olmAPI.ClusterServiceVersion, runtimeClient client.Client, obj *unstructured.Unstructured) error {
	test := scorecardTest{testType: olmIntegration, name: "Spec fields with descriptors"}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	if err != nil {
		return err
	}
	if obj.Object["spec"] == nil {
		// what should we do if there is no spec block? Maybe some kind of N/A type output?
		scTests = append(scTests, test)
		return nil
	}
	specBlock := obj.Object["spec"].(map[string]interface{})
	test.maximumPoints = len(specBlock)
	var crd *olmAPI.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == obj.GetKind() {
			crd = &owned
			break
		}
	}
	if crd == nil {
		scTests = append(scTests, test)
		return nil
	}
	for key := range specBlock {
		for _, specDesc := range crd.SpecDescriptors {
			if specDesc.Path == key {
				test.earnedPoints++
				delete(specBlock, key)
				break
			}
		}
	}
	scTests = append(scTests, test)
	for key := range specBlock {
		scSuggestions = append(scSuggestions, "Add a spec descriptor for " + key)
	}
	return nil
}
