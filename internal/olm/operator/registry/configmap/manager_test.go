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

package configmap

import (
	"context"

	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	"github.com/operator-framework/api/pkg/manifests"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Registry", func() {

	const testns = "testns"

	Describe("DeleteRegistry", func() {
		It("should delete the package manifest registry", func() {
			fakeclient := fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testns,
						Labels:    makeRegistryLabels("test"),
					},
				},
				newRegistryPod("pkgName", testns),
			).Build()
			m := Manager{
				pkg: &manifests.PackageManifest{
					PackageName: "pkgName",
					Channels: []manifests.PackageChannel{
						manifests.PackageChannel{
							Name: "pkgChannelTest",
						},
					},
				},
				bundles: []*apimanifests.Bundle{
					{
						Name: "testbundle",
						Objects: []*unstructured.Unstructured{
							{
								Object: map[string]interface{}{"val1": "val1"},
							},
							{
								Object: map[string]interface{}{"val2": "va2"},
							},
						},
						CSV: &v1alpha1.ClusterServiceVersion{
							Spec: v1alpha1.ClusterServiceVersionSpec{
								Version: version.OperatorVersion{
									Version: semver.SpecVersion,
								},
							},
						},
					},
				},
				cfg: &operator.Configuration{
					Namespace: testns,
					Client:    fakeclient,
				},
			}
			pod := corev1.Pod{}
			err := m.cfg.Client.Get(context.TODO(), types.NamespacedName{Name: getRegistryPodName("pkgName"), Namespace: testns}, &pod)
			Expect(err).Should(BeNil())

			cs := &v1alpha1.CatalogSource{}
			cs.SetName("foo")
			err = m.DeleteRegistry(context.TODO(), cs)
			Expect(err).Should(BeNil())

			err = m.cfg.Client.Get(context.TODO(), types.NamespacedName{Name: "pkgName-registry-server", Namespace: testns}, &pod)
			Expect(apierrors.IsNotFound(err)).Should(BeTrue())
		})
	})

	Describe("IsRegistryExist", func() {
		var (
			testns string
			m      Manager
		)
		BeforeEach(func() {
			fakeclient := fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testns,
						Labels:    makeRegistryLabels("test"),
					},
				},
				newRegistryPod("pkgName", testns),
			).Build()
			m = Manager{
				pkg: &manifests.PackageManifest{
					PackageName: "pkgName",
					Channels: []manifests.PackageChannel{
						manifests.PackageChannel{
							Name: "pkgChannelTest",
						},
					},
				},
				bundles: []*apimanifests.Bundle{
					{
						Name: "testbundle",
						Objects: []*unstructured.Unstructured{
							{
								Object: map[string]interface{}{"val1": "val1"},
							},
							{
								Object: map[string]interface{}{"val2": "va2"},
							},
						},
						CSV: &v1alpha1.ClusterServiceVersion{
							Spec: v1alpha1.ClusterServiceVersionSpec{
								Version: version.OperatorVersion{
									Version: semver.SpecVersion,
								},
							},
						},
					},
				},
				cfg: &operator.Configuration{
					Namespace: testns,
					Client:    fakeclient,
				},
			}
		})

		It("should return true if a deployment exitsts in the registry", func() {
			temp, err := m.IsRegistryExist(context.TODO())
			Expect(err).Should(BeNil())
			Expect(temp).Should(BeTrue())
		})

		It("should return false if a deployment does not exitst in the registry", func() {
			var (
				err  error
				temp bool
			)

			cs := &v1alpha1.CatalogSource{}
			cs.SetName("foo")
			err = m.DeleteRegistry(context.TODO(), cs)
			Expect(err).Should(BeNil())

			temp, err = m.IsRegistryExist(context.TODO())
			Expect(err).Should(BeNil())
			Expect(temp).Should(BeFalse())
		})
	})

	Describe("IsRegistryDataStale", func() {
		var (
			testns string
			m      Manager
		)
		BeforeEach(func() {
			fakeclient := fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config1",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config2",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				newRegistryPod("pkgName", testns),
			).Build()
			m = Manager{
				pkg: &manifests.PackageManifest{
					PackageName: "pkgName",
					Channels: []manifests.PackageChannel{
						manifests.PackageChannel{
							Name: "pkgChannelTest",
						},
					},
				},
				bundles: []*apimanifests.Bundle{
					{
						Package: "pkgName",
						Name:    "testbundle",
						Objects: []*unstructured.Unstructured{
							{
								Object: map[string]interface{}{"val1": "val1"},
							},
							{
								Object: map[string]interface{}{"val2": "va2"},
							},
						},
						CSV: &v1alpha1.ClusterServiceVersion{
							Spec: v1alpha1.ClusterServiceVersionSpec{
								Version: version.OperatorVersion{
									Version: semver.SpecVersion,
								},
							},
						},
					},
				},
				cfg: &operator.Configuration{
					Namespace: testns,
					Client:    fakeclient,
				},
			}
		})

		It("should return true if there are no registry configmaps", func() {
			m.cfg.Client = fake.NewClientBuilder().Build()
			temp, err := m.IsRegistryDataStale(context.TODO())

			Expect(err).Should(BeNil())
			Expect(temp).Should(BeTrue())
		})

		It("should return true if the configmap does not exist", func() {
			temp, err := m.IsRegistryDataStale(context.TODO())

			Expect(err).Should(BeNil())
			Expect(temp).Should(BeTrue())
		})

		It("should return true if the number of files to be added to the registry don't match the numberof files currently in the registry", func() {
			m.cfg.Client = fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      getRegistryConfigMapName("pkgName") + "-package",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config2",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				newRegistryPod("pkgName", testns),
			).Build()
			temp, err := m.IsRegistryDataStale(context.TODO())

			Expect(err).Should(BeNil())
			Expect(temp).Should(BeTrue())
		})

		It("should return true if the binary data does not have a filekey", func() {
			binarydata, _ := makeObjectBinaryData(struct {
				val1 string
				val2 string
			}{
				val1: "val1",
				val2: "val2",
			}, "userInput")
			m.cfg.Client = fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					BinaryData: binarydata,
					ObjectMeta: metav1.ObjectMeta{
						Name:      getRegistryConfigMapName("pkgName") + "-package",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config2",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				newRegistryPod("pkgName", testns),
			).Build()
			temp, err := m.IsRegistryDataStale(context.TODO())

			Expect(err).Should(BeNil())
			Expect(temp).Should(BeTrue())
		})

		It("should return true if it fails in the next iteration", func() {
			binarydata, _ := makeObjectBinaryData(&manifests.PackageManifest{
				PackageName: "pkgName",
				Channels: []manifests.PackageChannel{
					manifests.PackageChannel{
						Name: "pkgChannelTest",
					},
				},
			})
			m.cfg.Client = fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					BinaryData: binarydata,
					ObjectMeta: metav1.ObjectMeta{
						Name:      getRegistryConfigMapName("pkgName") + "-package",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      getRegistryConfigMapName("pkgName2") + "-package",
						Namespace: testns,
						Labels:    makeRegistryLabels("pkgName"),
					},
				},
				newRegistryPod("pkgName", testns),
			).Build()
			temp, err := m.IsRegistryDataStale(context.TODO())

			Expect(err).Should(BeNil())
			Expect(temp).Should(BeTrue())
		})
	})

	// TODO: Test CreatePackageManifestsRegistry and Test to make IsRegistryDataStale to return false

})
