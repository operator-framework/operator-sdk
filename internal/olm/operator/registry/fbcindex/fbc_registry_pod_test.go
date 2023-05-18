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

package fbcindex

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testIndexImageTag = "some-image:v1.2.3"

func TestCreateRegistryPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Registry Pod Suite")
}

// newFakeClient() returns a fake controller runtime client
func newFakeClient() client.Client {
	return fakeclient.NewClientBuilder().Build()
}

var _ = Describe("FBCRegistryPod", func() {

	var defaultBundleItems = []index.BundleItem{{
		ImageTag: "quay.io/example/example-operator-bundle:0.2.0",
		AddMode:  index.SemverBundleAddMode,
	}}

	Describe("creating registry pod", func() {
		var (
			rp  *FBCRegistryPod
			cfg *operator.Configuration
			cs  *v1alpha1.CatalogSource
		)

		BeforeEach(func() {
			cs = &v1alpha1.CatalogSource{
				ObjectMeta: v1.ObjectMeta{
					Name: "test-catalogsource",
				},
			}

			schm := runtime.NewScheme()
			Expect(v1alpha1.AddToScheme(schm)).ShouldNot(HaveOccurred())

			cfg = &operator.Configuration{
				Client:    newFakeClient(),
				Namespace: "test-default",
				Scheme:    schm,
			}
			rp = &FBCRegistryPod{
				BundleItems: defaultBundleItems,
				IndexImage:  testIndexImageTag,
			}
			By("initializing the FBCRegistryPod")
			Expect(rp.init(cfg, cs)).To(Succeed())
		})

		Context("with valid registry pod values", func() {
			It("should create the FBCRegistryPod successfully", func() {
				expectedPodName := "quay-io-example-example-operator-bundle-0-2-0"
				Expect(rp).NotTo(BeNil())
				Expect(rp.pod.Name).To(Equal(expectedPodName))
				Expect(rp.pod.Namespace).To(Equal(rp.cfg.Namespace))
				Expect(rp.pod.Spec.Containers[0].Name).To(Equal(defaultContainerName))
				if len(rp.pod.Spec.Containers) > 0 {
					if len(rp.pod.Spec.Containers[0].Ports) > 0 {
						Expect(rp.pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(rp.GRPCPort))
					}
				}
			})

			It("should create a registry pod when database path is not provided", func() {
				Expect(rp.FBCIndexRootDir).To(Equal(fmt.Sprintf("/%s-configs", cs.Name)))
			})

			It("should return a valid container command for one image", func() {
				output, err := rp.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(rp.FBCIndexRootDir, rp.GRPCPort)))
			})

			It("should return a valid container command for three images", func() {
				bundleItems := append(defaultBundleItems,
					index.BundleItem{
						ImageTag: "quay.io/example/example-operator-bundle:0.3.0",
						AddMode:  index.ReplacesBundleAddMode,
					},
					index.BundleItem{
						ImageTag: "quay.io/example/example-operator-bundle:1.0.1",
						AddMode:  index.SemverBundleAddMode,
					},
					index.BundleItem{
						ImageTag: "localhost/example-operator-bundle:1.0.1",
						AddMode:  index.SemverBundleAddMode,
					},
				)
				rp2 := FBCRegistryPod{
					GRPCPort:    defaultGRPCPort,
					BundleItems: bundleItems,
				}
				output, err := rp2.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(rp2.FBCIndexRootDir, rp2.GRPCPort)))
			})
		})

		Context("with invalid registry pod values", func() {
			It("should error when bundle image is not provided", func() {
				expectedErr := "bundle image set cannot be empty"
				rp := &FBCRegistryPod{}
				err := rp.init(cfg, cs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("checkPodStatus should return error when pod check is false and context is done", func() {
				rp := &FBCRegistryPod{
					BundleItems: defaultBundleItems,
					IndexImage:  testIndexImageTag,
				}
				Expect(rp.init(cfg, cs)).To(Succeed())

				mockBadPodCheck := wait.ConditionFunc(func() (done bool, err error) {
					return false, fmt.Errorf("error waiting for registry pod")
				})

				expectedErr := "error waiting for registry pod"
				// create a new context with a deadline of 1 millisecond
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
				cancel()

				err := rp.checkPodStatus(ctx, mockBadPodCheck)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})
		})

		Context("creating a ConfigMap", func() {
			It("makeBaseConfigMap() should return a basic ConfigMap manifest", func() {
				cm := rp.makeBaseConfigMap()
				Expect(cm.GetObjectKind().GroupVersionKind()).Should(Equal(corev1.SchemeGroupVersion.WithKind("ConfigMap")))
				Expect(cm.GetNamespace()).Should(Equal(cfg.Namespace))
				Expect(cm.Data).ShouldNot(BeNil())
				Expect(cm.Data).Should(BeEmpty())
			})

			It("partitionedConfigMaps() should return a single ConfigMap", func() {
				rp.FBCContent = testYaml
				expectedYaml := ""
				for i, yaml := range strings.Split(testYaml, "---")[1:] {
					if i != 0 {
						expectedYaml += "\n---\n"
					}

					expectedYaml += yaml
				}
				cms := rp.partitionedConfigMaps()
				Expect(cms).Should(HaveLen(1))
				Expect(cms[0].Data).Should(HaveKey("extraFBC"))
				Expect(cms[0].Data["extraFBC"]).Should(Equal(expectedYaml))
			})

			It("partitionedConfigMaps() should return multiple ConfigMaps", func() {
				// Create a large yaml manifest
				largeYaml := ""
				for i := len([]byte(largeYaml)); i < maxConfigMapSize; {
					largeYaml += testYaml
					i = len([]byte(largeYaml))
				}

				rp.FBCContent = largeYaml

				cms := rp.partitionedConfigMaps()
				Expect(cms).Should(HaveLen(2))
				Expect(cms[0].Data).Should(HaveKey("extraFBC"))
				Expect(cms[0].Data["extraFBC"]).ShouldNot(BeEmpty())
				Expect(cms[1].Data).Should(HaveKey("extraFBC"))
				Expect(cms[1].Data["extraFBC"]).ShouldNot(BeEmpty())
			})

			It("createOrUpdateConfigMap() should create the ConfigMap if it does not exist", func() {
				cm := rp.makeBaseConfigMap()
				cm.SetName("test-cm")
				cm.Data["test"] = "hello test world!"

				Expect(rp.createOrUpdateConfigMap(cm)).Should(Succeed())

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: cm.GetName()}, testCm)).Should(Succeed())
				Expect(testCm).Should(BeEquivalentTo(cm))
			})

			It("createOrUpdateConfigMap() should update the ConfigMap if it already exists", func() {
				cm := rp.makeBaseConfigMap()
				cm.SetName("test-cm")
				cm.Data["test"] = "hello test world!"
				Expect(rp.cfg.Client.Create(context.TODO(), cm)).Should(Succeed())
				cm.Data["test"] = "hello changed world!"
				cm.SetResourceVersion("2")

				Expect(rp.createOrUpdateConfigMap(cm)).Should(Succeed())

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: cm.GetName()}, testCm)).Should(Succeed())
				Expect(testCm).Should(BeEquivalentTo(cm))
			})

			It("createConfigMaps() should create a single ConfigMap", func() {
				rp.FBCContent = testYaml
				expectedYaml := ""
				for i, yaml := range strings.Split(testYaml, "---")[1:] {
					if i != 0 {
						expectedYaml += "\n---\n"
					}

					expectedYaml += yaml
				}

				expectedName := fmt.Sprintf("%s-configmap-partition-1", cs.GetName())

				cms, err := rp.createConfigMaps(cs)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(cms).Should(HaveLen(1))
				Expect(cms[0].GetNamespace()).Should(Equal(rp.cfg.Namespace))
				Expect(cms[0].GetName()).Should(Equal(expectedName))
				Expect(cms[0].Data).Should(HaveKey("extraFBC"))
				Expect(cms[0].Data["extraFBC"]).Should(Equal(expectedYaml))

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: expectedName}, testCm)).Should(Succeed())
				Expect(testCm.Data).Should(HaveKey("extraFBC"))
				Expect(testCm.Data["extraFBC"]).Should(Equal(expectedYaml))
				Expect(testCm.OwnerReferences).Should(HaveLen(1))
			})

			It("createConfigMaps() should create multiple ConfigMaps", func() {
				largeYaml := ""
				for i := len([]byte(largeYaml)); i < maxConfigMapSize; {
					largeYaml += testYaml
					i = len([]byte(largeYaml))
				}
				rp.FBCContent = largeYaml

				cms, err := rp.createConfigMaps(cs)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(cms).Should(HaveLen(2))

				for i, cm := range cms {
					expectedName := fmt.Sprintf("%s-configmap-partition-%d", cs.GetName(), i+1)
					Expect(cm.Data).Should(HaveKey("extraFBC"))
					Expect(cm.Data["extraFBC"]).ShouldNot(BeEmpty())
					Expect(cm.GetNamespace()).Should(Equal(rp.cfg.Namespace))
					Expect(cm.GetName()).Should(Equal(expectedName))

					testCm := &corev1.ConfigMap{}
					Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: expectedName}, testCm)).Should(Succeed())
					Expect(testCm.Data).Should(HaveKey("extraFBC"))
					Expect(testCm.Data["extraFBC"]).Should(Equal(cm.Data["extraFBC"]))
					Expect(testCm.OwnerReferences).Should(HaveLen(1))
				}
			})
		})
	})
})

// containerCommandFor returns the expected container command for a db path and set of bundle items.
func containerCommandFor(indexRootDir string, grpcPort int32) string { //nolint:unparam
	return fmt.Sprintf("opm serve %s -p %d", indexRootDir, grpcPort)
}

const testYaml = `
---
name: 'Vada O''Connell'
email: braun.leta@hirthe.biz
phone: 815.290.6848
description: 'Sit velit accusantium repellat itaque quisquam dolorem. Necessitatibus et provident explicabo. Animi officia enim omnis unde odio odio. Inventore autem repellendus ducimus et et et iure.'
address:
    streetName: 'Terence Garden'
    streetAddress: '644 Ward Ranch'
    city: 'North Newtonhaven'
    postcode: 37068-6948
    country: 'Saudi Arabia'
---
name: 'Miss Rita Gulgowski'
email: oran02@gmail.com
phone: '+13346133601'
description: 'Inventore recusandae ducimus nemo consequatur. Dolorum vel voluptas sint tempore iste maiores. Voluptatem nisi incidunt sit. Vel et officiis eum enim dolores dolor.'
address:
    streetName: 'Skylar Gateway'
    streetAddress: '8717 Karley Creek Suite 375'
    city: Kuhlmanshire
    postcode: '59539'
    country: Bolivia
---
name: 'Prof. Laverna Stanton'
email: nicklaus.turner@gmail.com
phone: 928.205.3796
description: 'Fugiat quos aspernatur iste fugit provident fugit aut. Optio rem exercitationem quas esse et nesciunt velit excepturi. Doloremque aliquid iure aut quaerat id repellat.'
address:
    streetName: 'Kamron Roads'
    streetAddress: '956 Lemke Camp'
    city: Malikatown
    postcode: '89393'
    country: 'French Southern Territories'
`
