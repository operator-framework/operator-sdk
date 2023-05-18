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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	scapiv1alpha3 "github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

var _ = Describe("Basic and OLM tests", func() {
	var (
		testBundle = filepath.Join("..", "testdata", "bundle")
		status     scapiv1alpha3.TestStatus
		result     scapiv1alpha3.TestResult
	)

	BeforeEach(func() {
		result = scapiv1alpha3.TestResult{
			Name:   "Scorecard result struct",
			State:  scapiv1alpha3.PassState,
			Errors: make([]string, 0),
		}
		status = scapiv1alpha3.TestStatus{
			Results: []scapiv1alpha3.TestResult{result}}
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
				status = CheckSpecTest(bundle)
				Expect(status.Results[0].State).To(Equal(scapiv1alpha3.PassState))
			})
		})

		Context("CRDsHaveValidationTest", func() {
			It("returns a pass state when CRDs have validations", func() {
				status = CRDsHaveValidationTest(bundle)
				Expect(status.Results[0].State).To(Equal(scapiv1alpha3.PassState))
			})
		})

		Context("CRDsHaveResourcesTest", func() {
			It("returns a pass state when CRDs have resources", func() {
				status = CRDsHaveResourcesTest(bundle)
				Expect(status.Results[0].State).To(Equal(scapiv1alpha3.PassState))
			})
		})

		Context("SpecDescriptorsTest", func() {
			It("returns a pass state then spec descriptors are present", func() {
				status = SpecDescriptorsTest(bundle)
				Expect(status.Results[0].State).To(Equal(scapiv1alpha3.PassState))
			})
		})

		Context("StatusDescriptorsTest", func() {
			It("returns a pass state then status descriptors are present", func() {
				status = StatusDescriptorsTest(bundle)
				Expect(status.Results[0].State).To(Equal(scapiv1alpha3.PassState))
			})
		})
	})

	Describe("Testing OLM Bundle", func() {
		It("should pass when test bundle is at the desired location", func() {
			metadata, _, err := registryutil.FindBundleMetadata(testBundle)
			Expect(err).NotTo(HaveOccurred())
			status = BundleValidationTest(testBundle, metadata)
			Expect(status.Results[0].State).To(Equal(scapiv1alpha3.PassState))
		})
	})

	Describe("OLM Tests", func() {

		Describe("Test Status and Spec Descriptors", func() {
			var (
				cr  unstructured.Unstructured
				csv operatorsv1alpha1.ClusterServiceVersion
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

				result = checkOwnedCSVStatusDescriptor(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.PassState))
			})

			It("should return warning when no spec status are defined for CRD", func() {
				cr = unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							"spec": "val",
						},
					},
				}
				cr.SetGroupVersionKind(schema.GroupVersionKind{
					Kind:  "TestKind",
					Group: "test.example.com",
				})

				result = checkOwnedCSVStatusDescriptor(cr, &csv, result)
				Expect(result.Suggestions).To(HaveLen(1))
				Expect(result.State).To(Equal(scapiv1alpha3.PassState))
			})

			It("should pass when CR Object Descriptor is nil", func() {
				cr := unstructured.Unstructured{
					Object: nil,
				}
				cr.SetGroupVersionKind(schema.GroupVersionKind{
					Kind:  "TestKind",
					Group: "test.example.com",
				})

				result = checkOwnedCSVStatusDescriptor(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.PassState))
			})

			It("should fail when CR Object Descriptor is nil and CRD with given GVK cannot be found", func() {
				cr := unstructured.Unstructured{
					Object: nil,
				}
				cr.SetGroupVersionKind(schema.GroupVersionKind{
					Kind:  "TestKindNotPresent",
					Group: "testnotpresent.example.com",
				})

				result = checkOwnedCSVStatusDescriptor(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.FailState))
			})

			It("should fail when owned CRD for CR does not have GVK set", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"status": "val",
						},
					},
				}

				result = checkOwnedCSVStatusDescriptor(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.FailState))
			})

			It("should fail when required descriptor field is not present in CR", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"node": map[string]interface{}{
							"node": "val",
						},
					},
				}

				result = checkOwnedCSVStatusDescriptor(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.FailState))
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

				result = checkOwnedCSVSpecDescriptors(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.PassState))
			})
			It("should fail when required spec descriptor field is not present in CR", func() {
				cr := unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"status": "val",
						},
					},
				}

				result = checkOwnedCSVSpecDescriptors(cr, &csv, result)
				Expect(result.State).To(Equal(scapiv1alpha3.FailState))
			})
			It("should fail when CRs do not have spec field specified", func() {
				cr := []unstructured.Unstructured{
					unstructured.Unstructured{
						Object: map[string]interface{}{},
					},
				}
				result = checkSpec(cr, result)
				Expect(result.State).To(Equal(scapiv1alpha3.PassState))
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
				result = checkSpec(cr, result)
				Expect(result.State).To(Equal(scapiv1alpha3.PassState))
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

			result = isCRFromCRDApi(cr, crd, result)
			Expect(result.State).To(Equal(scapiv1alpha3.PassState))

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

			result = isCRFromCRDApi(cr, crd, result)
			Expect(result.State).To(Equal(scapiv1alpha3.FailState))

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

			result = isCRFromCRDApi(cr, crd, result)
			Expect(result.State).To(Equal(scapiv1alpha3.PassState))

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
			result = CheckResources(crd, result)
			Expect(result.State).To(Equal(scapiv1alpha3.PassState))
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
			result = CheckResources(crd, result)
			Expect(result.State).To(Equal(scapiv1alpha3.FailState))
		})
	})

})

func TestScorecard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scorecard Basic and OLM Tests")
}
