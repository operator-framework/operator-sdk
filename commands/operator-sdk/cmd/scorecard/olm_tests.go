// Copyright 2018 The Operator-SDK Authors
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

	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// crdsHaveResources returns earned points and max points
func crdsHaveResources(csv *olmApi.ClusterServiceVersion) {
	test := scorecardTest{testType: olmIntegration, name: "Owned CRDs have resources listed"}
	for _, crd := range csv.Spec.CustomResourceDefinitions.Owned {
		test.maximumPoints++
		if len(crd.Resources) > 0 {
			test.earnedPoints++
		}
	}
	scTests = append(scTests, test)
}

func annotationsContainExamples(csv *olmApi.ClusterServiceVersion) {
	test := scorecardTest{testType: olmIntegration, name: "CRs have at least 1 example", maximumPoints: 1}
	if csv.Annotations != nil && csv.Annotations["alm-examples"] != "" {
		test.earnedPoints = 1
	}
	scTests = append(scTests, test)
}

func statusDescriptors(csv *olmApi.ClusterServiceVersion, runtimeClient client.Client, obj unstructured.Unstructured) error {
	test := scorecardTest{testType: olmIntegration, name: "Status fields with descriptors"}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: name}, &obj)
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
	var crd *olmApi.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == kind {
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
				break
			}
		}
	}
	scTests = append(scTests, test)
	return nil
}

func specDescriptors(csv *olmApi.ClusterServiceVersion, runtimeClient client.Client, obj unstructured.Unstructured) error {
	test := scorecardTest{testType: olmIntegration, name: "Spec fields with descriptors"}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: name}, &obj)
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
	var crd *olmApi.CRDDescription
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		if owned.Kind == kind {
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
				break
			}
		}
	}
	scTests = append(scTests, test)
	return nil
}
