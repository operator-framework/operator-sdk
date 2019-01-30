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
	"encoding/json"
	"strings"

	log "github.com/sirupsen/logrus"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// crdsHaveResources checks to make sure that all owned CRDs have resources listed
// Until there is full support for multiple CRs, we will only be able to check the
// actual used resources of one CRD, but only the existence of a resources section
// for other CRDs
func crdsHaveResources(obj *unstructured.Unstructured, csv *olmapiv1alpha1.ClusterServiceVersion) {
	test := scorecardTest{testType: olmIntegration, name: "Owned CRDs have resources listed"}
	for _, crd := range csv.Spec.CustomResourceDefinitions.Owned {
		test.maximumPoints++
		gvk := obj.GroupVersionKind()
		if strings.EqualFold(crd.Version, gvk.Version) && matchKind(gvk.Kind, crd.Kind) {
			resources, err := getUsedResources()
			if err != nil {
				log.Warningf("getUsedResource failed: %v", err)
			}
			allResourcesListed := true
			for _, resource := range resources {
				foundResource := false
				for _, listedResource := range crd.Resources {
					if matchKind(resource.Kind, listedResource.Kind) && strings.EqualFold(resource.Version, listedResource.Version) {
						foundResource = true
					}
				}
				if foundResource == false {
					allResourcesListed = false
				}
			}
			if allResourcesListed {
				test.earnedPoints++
			}
		} else {
			if len(crd.Resources) > 0 {
				test.earnedPoints++
			}
		}
	}
	scTests = append(scTests, test)
	if test.earnedPoints == 0 {
		scSuggestions = append(scSuggestions, "Add resources to owned CRDs")
	}
}

func getUsedResources() ([]schema.GroupVersionKind, error) {
	logs, err := getProxyLogs()
	if err != nil {
		return nil, err
	}
	var resources []schema.GroupVersionKind
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
				Collection: /apis/GROUP/VERSION/KIND
				Individual: /apis/GROUP/VERSION/KIND/NAME
				Core:       /api/v1/KIND
			Namespaces:
				All Namespaces:          /apis/GROUP/VERSION/KIND (same as cluster collection)
				Collection in Namespace: /apis/GROUP/VERSION/namespaces/NAMESPACE/KIND
				Individual:              /apis/GROUP/VERSION/namespaces/NAMESPACE/KIND/NAME
				Core:                    /api/v1/namespaces/NAMESPACE/KIND

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
		splitURI = splitURI[1:]
		switch len(splitURI) {
		case 3:
			if splitURI[0] == "api" {
				resources = append(resources, schema.GroupVersionKind{Version: splitURI[1], Kind: splitURI[2]})
			}
		case 4:
			if splitURI[0] == "apis" {
				resources = append(resources, schema.GroupVersionKind{Group: splitURI[1], Version: splitURI[2], Kind: splitURI[3]})
			}
		case 5:
			if splitURI[0] == "api" {
				resources = append(resources, schema.GroupVersionKind{Version: splitURI[1], Kind: splitURI[4]})
			} else if splitURI[0] == "apis" {
				resources = append(resources, schema.GroupVersionKind{Group: splitURI[1], Version: splitURI[2], Kind: splitURI[3]})
			}
		case 6, 7:
			if splitURI[0] == "apis" {
				resources = append(resources, schema.GroupVersionKind{Group: splitURI[1], Version: splitURI[2], Kind: splitURI[5]})
			}
		}
	}
	// remove duplicates
	addedResources := map[schema.GroupVersionKind]bool{}
	var deduplicatedResources []schema.GroupVersionKind
	for _, resource := range resources {
		if !addedResources[resource] {
			addedResources[resource] = true
			deduplicatedResources = append(deduplicatedResources, resource)
		}
	}
	return deduplicatedResources, nil
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

// annotationsContainExamples makes sure that the CSVs list at least 1 example for the CR
func annotationsContainExamples(csv *olmapiv1alpha1.ClusterServiceVersion) {
	test := scorecardTest{testType: olmIntegration, name: "CRs have at least 1 example", maximumPoints: 1}
	if csv.Annotations != nil && csv.Annotations["alm-examples"] != "" {
		test.earnedPoints = 1
	}
	scTests = append(scTests, test)
	if test.earnedPoints == 0 {
		scSuggestions = append(scSuggestions, "Add an alm-examples annotation to your CSV to pass the "+test.name+" test")
	}
}

// statusDescriptors makes sure that all status fields found in the created CR has a matching descriptor in the CSV
func statusDescriptors(csv *olmapiv1alpha1.ClusterServiceVersion, runtimeClient client.Client, obj *unstructured.Unstructured) error {
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
	var crd *olmapiv1alpha1.CRDDescription
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
func specDescriptors(csv *olmapiv1alpha1.ClusterServiceVersion, runtimeClient client.Client, obj *unstructured.Unstructured) error {
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
	var crd *olmapiv1alpha1.CRDDescription
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
		scSuggestions = append(scSuggestions, "Add a spec descriptor for "+key)
	}
	return nil
}
