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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

var _ = Describe("Basic and OLM tests", func() {
	var (
		testBundle = filepath.Join("..", "testdata", "bundle")
		res        scapiv1alpha2.ScorecardTestResult
	)

	BeforeEach(func() {
		res = scapiv1alpha2.ScorecardTestResult{
			Name:   "Scorecard result struct",
			State:  scapiv1alpha2.PassState,
			Errors: make([]string, 0),
		}
	})

	Describe("Test Bundle CRs", func() {
		var (
			bundle  *apimanifests.Bundle
			err     error
			crCount int
			crList  []unstructured.Unstructured
		)

		It("Check Bundle CRs", func() {
			crCount = 1

			bundle, err = apimanifests.GetBundleFromDir(testBundle)

			Expect(err).ToNot(HaveOccurred())

			crList, err = GetCRs(bundle)
			Expect(err).ToNot(HaveOccurred())
			Expect(crCount).To(Equal(len(crList)))
		})
	})

	Describe("Testing Basic and OLM tests", func() {
		var (
			bundle *apimanifests.Bundle
			err    error
		)

		BeforeEach(func() {
			bundle, err = apimanifests.GetBundleFromDir(testBundle)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("CheckSpecTest", func() {
			It("returns a pass state when Spec field exists", func() {
				res = CheckSpecTest(bundle)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})
		})

		Context("CRDsHaveValidationTest", func() {
			It("returns a pass state when CRDs have validations", func() {
				res = CRDsHaveValidationTest(bundle)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})
		})

		Context("CRDsHaveResourcesTest", func() {
			It("returns a pass state when CRDs have resources", func() {
				res = CRDsHaveResourcesTest(bundle)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})
		})

		Context("SpecDescriptorsTest", func() {
			It("returns a pass state then spec descriptors are present", func() {
				res = SpecDescriptorsTest(bundle)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})
		})

		Context("StatusDescriptorsTest", func() {
			It("returns a pass state then status descriptors are present", func() {
				res = StatusDescriptorsTest(bundle)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})
		})
	})

	Describe("Testing OLM Bundle", func() {
		It("should pass when test bundle is at the desired location", func() {
			res = BundleValidationTest(testBundle)
			Expect(res.State).To(Equal(scapiv1alpha2.PassState))
		})
	})

	Describe("OLM Tests", func() {

		Describe("Test Status and Spec Descriptors", func() {
			var (
				cr         unstructured.Unstructured
				descriptor string
				csv        operatorsv1alpha1.ClusterServiceVersion
			)

			csv = operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					CustomResourceDefinitions: operatorsv1alpha1.CustomResourceDefinitions{
						Owned: []operatorsv1alpha1.CRDDescription{
							operatorsv1alpha1.CRDDescription{
								Name:    "Test",
								Version: "v1",
								Kind:    "TestKind",
								StatusDescriptors: []operatorsv1alpha1.StatusDescriptor{
									operatorsv1alpha1.StatusDescriptor{
										Path: "status",
									},
								},
								SpecDescriptors: []operatorsv1alpha1.SpecDescriptor{
									operatorsv1alpha1.SpecDescriptor{
										Path: "spec",
									},
								},
							},
						},
					},
				},
			}

			descriptor = "status"

			It("should pass when csv with owned cr and required fields is present", func() {
				cr = unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"status": "val",
						},
						"spec": map[string]interface{}{
							"spec": "val",
						},
					},
				}
				cr.SetGroupVersionKind(schema.GroupVersionKind{
					Kind:  "TestKind",
					Group: "test.example.com",
				})

				res = checkOwnedCSVDescriptors(cr, &csv, descriptor, res)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})

			It("should fail when CR Object Descriptor is nil", func() {
				cr := unstructured.Unstructured{
					Object: nil,
				}

				res = checkOwnedCSVDescriptors(cr, &csv, descriptor, res)
				Expect(res.State).To(Equal(scapiv1alpha2.FailState))
			})

			It("should fail when owned CRD for CR does not have GVK set", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"status": "val",
						},
					},
				}

				res = checkOwnedCSVDescriptors(cr, &csv, descriptor, res)
				Expect(res.State).To(Equal(scapiv1alpha2.FailState))
			})

			It("should fail when required descriptor field is not present in CR", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"node": map[string]interface{}{
							"node": "val",
						},
					},
				}

				res = checkOwnedCSVDescriptors(cr, &csv, descriptor, res)
				Expect(res.State).To(Equal(scapiv1alpha2.FailState))
			})
			It("should pass when required descriptor field is present in CR", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"status": "val",
						},
						"spec": map[string]interface{}{
							"spec": "val",
						},
					},
				}
				cr.SetGroupVersionKind(schema.GroupVersionKind{
					Kind:  "TestKind",
					Group: "test.example.com",
				})

				res = checkOwnedCSVDescriptors(cr, &csv, descriptor, res)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})
			It("should fail when required spec descriptor field is not present in CR", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"status": "val",
						},
					},
				}

				res = checkOwnedCSVDescriptors(cr, &csv, descriptor, res)
				Expect(res.State).To(Equal(scapiv1alpha2.FailState))
			})
			It("should fail when CRs do not have spec field specified", func() {
				cr := []unstructured.Unstructured{
					unstructured.Unstructured{
						Object: map[string]interface{}{},
					},
				}
				res = checkSpec(cr, res)
				Expect(res.State).To(Equal(scapiv1alpha2.FailState))
			})
			It("should pass when CRs do have spec field specified", func() {
				cr := []unstructured.Unstructured{
					unstructured.Unstructured{
						Object: map[string]interface{}{
							"spec": map[string]interface{}{
								"spec": "val",
							},
						},
					},
				}
				res = checkSpec(cr, res)
				Expect(res.State).To(Equal(scapiv1alpha2.PassState))
			})

		})

	})

	Describe("CRDs have validation test", func() {
		var (
			cr  unstructured.Unstructured
			crd []*apiextv1.CustomResourceDefinition
		)

		crd = []*apiextv1.CustomResourceDefinition{
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

		It("should pass when CR has Spec field", func() {
			cr = unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"node": "val",
					},
				},
			}
			cr.SetGroupVersionKind(schema.GroupVersionKind{
				Kind:    "TestKind",
				Group:   "test.example.com",
				Version: "v1",
			})

			res = isCRFromCRDApi(cr, crd, res)
			Expect(res.State).To(Equal(scapiv1alpha2.PassState))

		})

		It("should fail when cr does not have required fields in Spec", func() {
			cr = unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"items": "val",
					},
				},
			}
			cr.SetGroupVersionKind(schema.GroupVersionKind{
				Kind:    "TestKind",
				Group:   "test.example.com",
				Version: "v1",
			})

			res = isCRFromCRDApi(cr, crd, res)
			Expect(res.State).To(Equal(scapiv1alpha2.FailState))

		})

		It("should skip and pass when version/kind does not match for CR with CRD", func() {
			cr = unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"node": "val",
					},
				},
			}
			cr.SetGroupVersionKind(schema.GroupVersionKind{
				Kind:    "MemcachedKind",
				Group:   "Cache",
				Version: "v2",
			})

			res = isCRFromCRDApi(cr, crd, res)
			Expect(res.State).To(Equal(scapiv1alpha2.PassState))

		})

	})

	Describe("Check CRDs for resources", func() {
		var (
			crd operatorsv1alpha1.CustomResourceDefinitions
		)

		It("Should pass when CSV has Owned CRD's with resources", func() {
			crd = operatorsv1alpha1.CustomResourceDefinitions{
				Owned: []operatorsv1alpha1.CRDDescription{
					operatorsv1alpha1.CRDDescription{
						Name:              "Test",
						Version:           "v1",
						Kind:              "Test",
						StatusDescriptors: make([]operatorsv1alpha1.StatusDescriptor, 0),
						Resources: []operatorsv1alpha1.APIResourceReference{
							operatorsv1alpha1.APIResourceReference{
								Name:    "operator",
								Kind:    "Test",
								Version: "v1",
							},
						},
					},
				},
			}
			res = CheckResources(crd, res)
			Expect(res.State).To(Equal(scapiv1alpha2.PassState))
		})

		It("Should fail when CSV does not have Owned CRD's with resources", func() {

			crd = operatorsv1alpha1.CustomResourceDefinitions{
				Owned: []operatorsv1alpha1.CRDDescription{
					operatorsv1alpha1.CRDDescription{
						Name:              "Test",
						Version:           "v1",
						Kind:              "Test",
						StatusDescriptors: make([]operatorsv1alpha1.StatusDescriptor, 0),
						Resources:         make([]operatorsv1alpha1.APIResourceReference, 0),
					},
				},
			}
			res = CheckResources(crd, res)
			Expect(res.State).To(Equal(scapiv1alpha2.FailState))
		})

	})

})

func TestScorecard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scorecard Basic and OLM Tests")
}
