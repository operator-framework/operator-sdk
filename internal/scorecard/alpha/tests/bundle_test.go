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
	"path/filepath"
	"testing"

	"github.com/operator-framework/api/pkg/operators"
	"github.com/operator-framework/operator-registry/pkg/registry"

	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var testBundle = filepath.Join("..", "testdata", "bundle")

func TestBundlePath(t *testing.T) {
	cases := []struct {
		bundlePath string
		wantError  bool
	}{
		{testBundle, false},
		{"/foo", true},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			_, err := GetBundle(c.bundlePath)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			} else if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}

		})

	}
}
func TestBundleCRs(t *testing.T) {
	cases := []struct {
		bundlePath string
		crCount    int
		wantError  bool
	}{
		{testBundle, 1, false},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			bundle, err := GetBundle(c.bundlePath)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			} else if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}
			var crList []unstructured.Unstructured
			crList, err = GetCRs(*bundle)
			if err != nil {
				t.Error(err)
				return
			}
			if len(crList) != c.crCount {
				t.Errorf("Wanted %d CRs but got: %d", c.crCount, len(crList))
				return
			}

		})

	}
}

func TestBasicAndOLM(t *testing.T) {

	cases := []struct {
		bundlePath string
		state      scapiv1alpha2.State
		function   func(registry.Bundle) scapiv1alpha2.ScorecardTestResult
	}{
		{testBundle, scapiv1alpha2.PassState, CheckSpecTest},
		{testBundle, scapiv1alpha2.PassState, CRDsHaveValidationTest},
		{testBundle, scapiv1alpha2.PassState, CRDsHaveResourcesTest},
		{testBundle, scapiv1alpha2.PassState, SpecDescriptorsTest},
		{testBundle, scapiv1alpha2.PassState, StatusDescriptorsTest},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			bundle, err := GetBundle(c.bundlePath)
			if err != nil {
				t.Fatalf("Error getting bundle: %s", err.Error())
			}

			result := c.function(*bundle)
			if result.State != c.state {
				t.Errorf("%s result State %v expected", result.Name, c.state)
				return
			}
		})
	}
}

func TestOLMBundle(t *testing.T) {
	cases := []struct {
		bundlePath string
		state      scapiv1alpha2.State
	}{
		{testBundle, scapiv1alpha2.PassState},
	}
	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {
			result := BundleValidationTest(c.bundlePath)
			if result.State != c.state {
				t.Errorf("%s result State %v expected", result.Name, c.state)
				return
			}
		})
	}
}

func TestDescriptors(t *testing.T) {
	crWithDescriptor := unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"status": "val",
			},
			"spec": map[string]interface{}{
				"spec": "val",
			},
		},
	}
	crWithDescriptor.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:  "TestKind",
		Group: "test.example.com",
	})

	crWithoutDescriptor := unstructured.Unstructured{
		Object: nil,
	}

	crWithoutGVK := unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"status": "val",
			},
		},
	}

	crWithNoRequiredDescriptor := unstructured.Unstructured{
		Object: map[string]interface{}{
			"node": map[string]interface{}{
				"node": "val",
			},
		},
	}

	crwithNoSpecDescriptor := unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"status": "val",
			},
		},
	}

	csvWithOwnedCR := operators.ClusterServiceVersion{
		Spec: operators.ClusterServiceVersionSpec{
			CustomResourceDefinitions: operators.CustomResourceDefinitions{
				Owned: []operators.CRDDescription{
					operators.CRDDescription{
						Name:    "Test",
						Version: "v1",
						Kind:    "TestKind",
						StatusDescriptors: []operators.StatusDescriptor{
							operators.StatusDescriptor{
								Path: "status",
							},
						},
						SpecDescriptors: []operators.SpecDescriptor{
							operators.SpecDescriptor{
								Path: "spec",
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		cr         unstructured.Unstructured
		csv        *operators.ClusterServiceVersion
		descriptor string
		want       scapiv1alpha2.State
	}{
		{
			name:       "should pass when csv with owned cr and required fields is present",
			cr:         crWithDescriptor,
			descriptor: "status",
			csv:        &csvWithOwnedCR,
			want:       scapiv1alpha2.PassState,
		},
		{
			name:       "should fail when CR Object Descriptor is nil",
			cr:         crWithoutDescriptor,
			descriptor: "status",
			csv:        &csvWithOwnedCR,
			want:       scapiv1alpha2.FailState,
		},
		{
			name:       "should fail when owned CRD for CR does not have GVK set",
			cr:         crWithoutGVK,
			descriptor: "status",
			csv:        &csvWithOwnedCR,
			want:       scapiv1alpha2.FailState,
		},
		{
			name:       "should fail when required descriptor field is not present in CR",
			cr:         crWithNoRequiredDescriptor,
			descriptor: "status",
			csv:        &csvWithOwnedCR,
			want:       scapiv1alpha2.FailState,
		},
		{
			name:       "should pass when required descriptor field is present in CR",
			cr:         crWithDescriptor,
			descriptor: "spec",
			csv:        &csvWithOwnedCR,
			want:       scapiv1alpha2.PassState,
		},
		{
			name:       "should fail when required spec descriptor field is not present in CR",
			cr:         crwithNoSpecDescriptor,
			descriptor: "spec",
			csv:        &csvWithOwnedCR,
			want:       scapiv1alpha2.FailState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := scapiv1alpha2.ScorecardTestResult{
				Name:   "Test status and spec descriptor",
				State:  scapiv1alpha2.PassState,
				Errors: make([]string, 0),
			}
			if res = checkOwnedCSVDescriptors(tt.cr, tt.csv, tt.descriptor, res); res.State != tt.want {
				t.Errorf("%s result State %v expected but obtained %v ",
					res.Name, tt.want, res.State)
			}
		})
	}
}

func TestCRDsHaveValidationTests(t *testing.T) {
	crWithSpec := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"node": "val",
			},
		},
	}
	crWithSpec.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "TestKind",
		Group:   "test.example.com",
		Version: "v1",
	})

	crWithoutSpec := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"items": "val",
			},
		},
	}
	crWithoutSpec.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "TestKind",
		Group:   "test.example.com",
		Version: "v1",
	})

	crWithNoMatchingGVK := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"node": "val",
			},
		},
	}
	crWithNoMatchingGVK.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "MemcachedKind",
		Group:   "Cache",
		Version: "v2",
	})

	crdsWithSpecInSchema := []*apiextv1.CustomResourceDefinition{
		&apiextv1.CustomResourceDefinition{
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					apiextv1.CustomResourceDefinitionVersion{
						Name: "v1",
						Schema: &apiextv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextv1.JSONSchemaProps{
								ID:          "Test",
								Schema:      "URL",
								Description: "Schema for test",
								Properties: map[string]apiextv1.JSONSchemaProps{
									"spec": apiextv1.JSONSchemaProps{
										Properties: map[string]apiextv1.JSONSchemaProps{
											"node": apiextv1.JSONSchemaProps{
												ID: "node",
											},
										},
									},
								},
							},
						},
					},
				},
				Names: apiextv1.CustomResourceDefinitionNames{
					Kind: "TestKind",
				},
			},
		},
	}
	tests := []struct {
		name string
		cr   unstructured.Unstructured
		crd  []*apiextv1.CustomResourceDefinition
		want scapiv1alpha2.State
	}{
		{
			name: "should pass when CR has Spec field",
			cr:   crWithSpec,
			crd:  crdsWithSpecInSchema,
			want: scapiv1alpha2.PassState,
		},
		{
			name: "should fail when cr does not have required fields in Spec",
			cr:   crWithoutSpec,
			crd:  crdsWithSpecInSchema,
			want: scapiv1alpha2.FailState,
		},
		{
			name: "should skip and pass when version/kind does not match for CR with CRD",
			cr:   crWithNoMatchingGVK,
			crd:  crdsWithSpecInSchema,
			want: scapiv1alpha2.PassState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := scapiv1alpha2.ScorecardTestResult{
				Name:   "Test CRDs have validation test",
				State:  scapiv1alpha2.PassState,
				Errors: make([]string, 0),
			}
			if res = isCRFromCRDApi(tt.cr, tt.crd, res); res.State != tt.want {
				t.Errorf("%s result State %v expected but obtained %v ",
					res.Name, tt.want, res.State)
			}
		})
	}
}

func TestCheckForResources(t *testing.T) {
	crdWithResources := operators.CustomResourceDefinitions{
		Owned: []operators.CRDDescription{
			operators.CRDDescription{
				Name:              "Test",
				Version:           "v1",
				Kind:              "Test",
				StatusDescriptors: make([]operators.StatusDescriptor, 0),
				Resources: []operators.APIResourceReference{
					operators.APIResourceReference{
						Name:    "operator",
						Kind:    "Test",
						Version: "v1",
					},
				},
			},
		},
	}

	crdWithoutResources := operators.CustomResourceDefinitions{
		Owned: []operators.CRDDescription{
			operators.CRDDescription{
				Name:              "Test",
				Version:           "v1",
				Kind:              "Test",
				StatusDescriptors: make([]operators.StatusDescriptor, 0),
				Resources:         make([]operators.APIResourceReference, 0),
			},
		},
	}

	tests := []struct {
		name string
		args operators.CustomResourceDefinitions
		want scapiv1alpha2.State
	}{
		{
			name: "Should pass when CSV has Owned CRD's with resources",
			args: crdWithResources,
			want: scapiv1alpha2.PassState,
		},
		{
			name: "Should fail when CSV does not have Owned CRD's with resources",
			args: crdWithoutResources,
			want: scapiv1alpha2.FailState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := scapiv1alpha2.ScorecardTestResult{
				Name:   "Check if CRDs have resources",
				State:  scapiv1alpha2.PassState,
				Errors: make([]string, 0),
			}
			if res = CheckResources(tt.args, res); res.State != tt.want {
				t.Errorf("%s result State %v expected but obtained %v",
					res.Name, tt.want, res.State)
			}
		})
	}
}
