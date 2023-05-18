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
	"fmt"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	"github.com/operator-framework/api/pkg/manifests"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Registry", func() {

	Describe("makeRegistryLabels", func() {
		It("should return the registry label", func() {
			labels := map[string]string{
				"package-name": k8sutil.TrimDNS1123Label("pkgName"),
			}
			for k, v := range SDKLabels {
				labels[k] = v
			}

			Expect(makeRegistryLabels("pkgName")).Should(Equal(labels))
		})
	})

	Describe("GetRegistryServiceAddr", func() {
		It("should return a Service's DNS name + port for a given pkgName and namespace", func() {
			name := fmt.Sprintf("%s.%s.svc.cluster.local:%d", getRegistryServerName("pkgName"), "testns", registryGRPCPort)

			Expect(GetRegistryServiceAddr("pkgName", "testns")).Should(Equal(name))
		})
	})

	Describe("DeletePackageManifestsRegistry", func() {
		It("should delete the package manifest registry", func() {
			fakeclient := fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testns",
						Labels:    makeRegistryLabels("test"),
					},
				},
				newRegistryDeployment("pkgName", "testns"),
				newRegistryService("pkgName", "testns"),
			).Build()
			rr := RegistryResources{
				Pkg: &manifests.PackageManifest{
					PackageName: "pkgName",
					Channels: []manifests.PackageChannel{
						manifests.PackageChannel{
							Name: "pkgChannelTest",
						},
					},
				},
				Bundles: []*apimanifests.Bundle{
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
				Client: &client.Client{
					KubeClient: fakeclient,
				},
			}
			dep := appsv1.Deployment{}
			Expect(
				rr.Client.KubeClient.Get(context.TODO(), types.NamespacedName{Name: getRegistryServerName("pkgName"), Namespace: "testns"}, &dep),
			).Should(Succeed())
			Expect(rr.DeletePackageManifestsRegistry(context.TODO(), "testns")).Should(Succeed())

			err := rr.Client.KubeClient.Get(context.TODO(), types.NamespacedName{Name: "pkgName-registry-server", Namespace: "testns"}, &dep)
			Expect(apierrors.IsNotFound(err)).Should(BeTrue())
		})
	})

	Describe("IsRegistryExist", func() {
		var (
			testns string
			rr     RegistryResources
		)
		BeforeEach(func() {
			testns = "testns"
			fakeclient := fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testns,
						Labels:    makeRegistryLabels("test"),
					},
				},
				newRegistryDeployment("pkgName", testns),
				newRegistryService("pkgName", testns),
			).Build()
			rr = RegistryResources{
				Pkg: &manifests.PackageManifest{
					PackageName: "pkgName",
					Channels: []manifests.PackageChannel{
						manifests.PackageChannel{
							Name: "pkgChannelTest",
						},
					},
				},
				Bundles: []*apimanifests.Bundle{
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
				Client: &client.Client{
					KubeClient: fakeclient,
				},
			}
		})

		It("should return true if a deployment exitsts in the registry", func() {
			temp, err := rr.IsRegistryExist(context.TODO(), testns)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(temp).Should(BeTrue())
		})

		It("should return false if a deployment does not exitst in the registry", func() {
			var (
				err  error
				temp bool
			)

			Expect(rr.DeletePackageManifestsRegistry(context.TODO(), testns)).Should(Succeed())

			temp, err = rr.IsRegistryExist(context.TODO(), testns)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(temp).Should(BeFalse())
		})
	})

	Describe("IsRegistryDataStale", func() {
		var (
			testns string
			rr     RegistryResources
		)
		BeforeEach(func() {
			testns = "testns"
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
				newRegistryDeployment("pkgName", testns),
				newRegistryService("pkgName", testns),
			).Build()
			rr = RegistryResources{
				Pkg: &manifests.PackageManifest{
					PackageName: "pkgName",
					Channels: []manifests.PackageChannel{
						manifests.PackageChannel{
							Name: "pkgChannelTest",
						},
					},
				},
				Bundles: []*apimanifests.Bundle{
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
				Client: &client.Client{
					KubeClient: fakeclient,
				},
			}
		})

		It("should return true if there are no registry configmaps", func() {
			rr.Client.KubeClient = fake.NewClientBuilder().Build()
			temp, err := rr.IsRegistryDataStale(context.TODO(), testns)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(temp).Should(BeTrue())
		})

		It("should return true if the configmap does not exist", func() {
			temp, err := rr.IsRegistryDataStale(context.TODO(), testns)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(temp).Should(BeTrue())
		})

		It("should return true if the number of files to be added to the registry don't match the numberof files currently in the registry", func() {
			rr.Client.KubeClient = fake.NewClientBuilder().WithObjects(
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
				newRegistryDeployment("pkgName", testns),
				newRegistryService("pkgName", testns),
			).Build()
			temp, err := rr.IsRegistryDataStale(context.TODO(), testns)

			Expect(err).ShouldNot(HaveOccurred())
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
			rr.Client.KubeClient = fake.NewClientBuilder().WithObjects(
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
				newRegistryDeployment("pkgName", testns),
				newRegistryService("pkgName", testns),
			).Build()
			temp, err := rr.IsRegistryDataStale(context.TODO(), testns)

			Expect(err).ShouldNot(HaveOccurred())
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
			rr.Client.KubeClient = fake.NewClientBuilder().WithObjects(
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
				newRegistryDeployment("pkgName", testns),
				newRegistryService("pkgName", testns),
			).Build()
			temp, err := rr.IsRegistryDataStale(context.TODO(), testns)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(temp).Should(BeTrue())
		})
	})

	// TODO: Test CreatePackageManifestsRegistry and Test to make IsRegistryDataStale to return false

})
