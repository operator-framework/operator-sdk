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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		JustBeforeEach(func() {
			cs = &v1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{
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
				Expect(rp.pod.Spec.Containers).Should(HaveLen(1))
				Expect(rp.pod.Spec.Containers[0].Ports).Should(HaveLen(1))
				Expect(rp.pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(rp.GRPCPort))
				Expect(rp.pod.Spec.Containers[0].Command).Should(HaveLen(3))
				Expect(rp.pod.Spec.Containers[0].Command).Should(ContainElements("sh", "-c", containerCommandFor(rp.FBCIndexRootDir, rp.GRPCPort)))
				Expect(rp.pod.Spec.InitContainers).Should(HaveLen(1))
			})

			It("should create a registry pod when database path is not provided", func() {
				Expect(rp.FBCIndexRootDir).To(Equal(fmt.Sprintf("/%s-configs", cs.Name)))
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

		Context("creating a compressed ConfigMap", func() {
			It("cmWriter.makeBaseConfigMap() should return a basic ConfigMap manifest", func() {
				cm := rp.cmWriter.newConfigMap("test-cm")
				Expect(cm.Name).Should(Equal("test-cm"))
				Expect(cm.GetObjectKind().GroupVersionKind()).Should(Equal(corev1.SchemeGroupVersion.WithKind("ConfigMap")))
				Expect(cm.GetNamespace()).Should(Equal(cfg.Namespace))
				Expect(cm.Data).Should(BeNil())
				Expect(cm.BinaryData).ShouldNot(BeNil())
				Expect(cm.BinaryData).Should(BeEmpty())
			})

			It("partitionedConfigMaps() should return a single compressed ConfigMap", func() {
				rp.FBCContent = testYaml
				expectedYaml := strings.TrimPrefix(strings.TrimSpace(testYaml), "---\n")

				cms, err := rp.partitionedConfigMaps()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(cms).Should(HaveLen(1))
				Expect(cms[0].BinaryData).Should(HaveKey("extraFBC"))

				By("uncompressed the BinaryData")
				uncompressed := decompressCM(cms[0])
				Expect(uncompressed).Should(Equal(expectedYaml))
			})

			It("partitionedConfigMaps() should return a single compressed ConfigMap for large yaml", func() {
				largeYaml := strings.Builder{}
				for largeYaml.Len() < maxConfigMapSize {
					largeYaml.WriteString(testYaml)
				}
				rp.FBCContent = largeYaml.String()

				expectedYaml := strings.TrimPrefix(strings.TrimSpace(largeYaml.String()), "---\n")
				expectedYaml = regexp.MustCompile(`\n\n+`).ReplaceAllString(expectedYaml, "\n")

				cms, err := rp.partitionedConfigMaps()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(cms).Should(HaveLen(1))
				Expect(cms[0].BinaryData).Should(HaveKey("extraFBC"))

				actualBinaryData := cms[0].BinaryData["extraFBC"]
				Expect(len(actualBinaryData)).Should(BeNumerically("<", maxConfigMapSize))
				By("uncompress the BinaryData")
				uncompressed := decompressCM(cms[0])
				Expect(uncompressed).Should(Equal(expectedYaml))
			})

			It("partitionedConfigMaps() should return a multiple compressed ConfigMaps for a huge yaml", func() {
				// build completely random yamls. This is because gzip relies on duplications, and so repeated text is
				// compressed very well, so we'll need a really huge input to create more than one CM. When using random
				// input, gzip will create larger output, and we can get to multiple CM with much smaller input.
				largeYamlBuilder := strings.Builder{}
				for largeYamlBuilder.Len() < maxConfigMapSize*2 {
					largeYamlBuilder.WriteString(generateRandYaml())
				}
				largeYaml := largeYamlBuilder.String()
				rp.FBCContent = largeYaml

				expectedYaml := strings.TrimPrefix(strings.TrimSpace(largeYaml), "---\n")
				expectedYaml = regexp.MustCompile(`\n\n+`).ReplaceAllString(expectedYaml, "\n")

				cms, err := rp.partitionedConfigMaps()
				Expect(err).ShouldNot(HaveOccurred())

				Expect(cms).Should(HaveLen(2))
				Expect(cms[0].BinaryData).Should(HaveKey("extraFBC"))
				Expect(cms[1].BinaryData).Should(HaveKey("extraFBC"))
				decompressed1 := decompressCM(cms[1])
				decompressed0 := decompressCM(cms[0])
				Expect(decompressed0 + "\n---\n" + decompressed1).Should(Equal(expectedYaml))
			})

			It("createOrUpdateConfigMap() should create the compressed ConfigMap if it does not exist", func() {
				cm := rp.cmWriter.newConfigMap("test-cm")
				cm.BinaryData["test"] = compress("hello test world!")

				Expect(rp.createOrUpdateConfigMap(cm)).Should(Succeed())

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: cm.GetName()}, testCm)).Should(Succeed())
				Expect(testCm).Should(BeEquivalentTo(cm))
			})

			It("createOrUpdateConfigMap() should update the compressed ConfigMap if it already exists", func() {
				cm := rp.cmWriter.newConfigMap("test-cm")
				cm.BinaryData["test"] = compress("hello test world!")
				Expect(rp.cfg.Client.Create(context.TODO(), cm)).Should(Succeed())
				cm.BinaryData["test"] = compress("hello changed world!")
				cm.SetResourceVersion("2")

				Expect(rp.createOrUpdateConfigMap(cm)).Should(Succeed())

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: cm.GetName()}, testCm)).Should(Succeed())
				Expect(testCm).Should(BeEquivalentTo(cm))
			})

			It("createOrUpdateConfigMap() should update the uncompressed-old ConfigMap if it already exists", func() {
				origCM := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: corev1.SchemeGroupVersion.String(),
						Kind:       "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: rp.cfg.Namespace,
						Name:      "test-cm",
					},
					Data: map[string]string{"test": "hello test world!"},
				}

				Expect(rp.cfg.Client.Create(context.TODO(), origCM)).Should(Succeed())
				cm := rp.cmWriter.newConfigMap("test-cm")
				cm.BinaryData["test"] = compress("hello changed world!")
				cm.SetResourceVersion("2")

				Expect(rp.createOrUpdateConfigMap(cm)).Should(Succeed())

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: cm.GetName()}, testCm)).Should(Succeed())
				Expect(cm.Data).Should(BeNil())
				Expect(testCm.BinaryData).Should(BeEquivalentTo(cm.BinaryData))
			})

			It("createConfigMaps() should create a single compressed ConfigMap", func() {
				rp.FBCContent = testYaml

				expectedYaml := strings.TrimPrefix(strings.TrimSpace(testYaml), "---\n")
				expectedName := fmt.Sprintf("%s-configmap-partition-1", cs.GetName())

				cms, err := rp.createConfigMaps(cs)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(cms).Should(HaveLen(1))
				Expect(cms[0].GetNamespace()).Should(Equal(rp.cfg.Namespace))
				Expect(cms[0].GetName()).Should(Equal(expectedName))
				Expect(cms[0].Data).Should(BeNil())
				Expect(cms[0].BinaryData).Should(HaveKey("extraFBC"))
				uncompressed := decompressCM(cms[0])
				Expect(uncompressed).Should(Equal(expectedYaml))

				testCm := &corev1.ConfigMap{}
				Expect(rp.cfg.Client.Get(context.TODO(), types.NamespacedName{Namespace: rp.cfg.Namespace, Name: expectedName}, testCm)).Should(Succeed())
				Expect(testCm.BinaryData).Should(HaveKey("extraFBC"))
				Expect(testCm.Data).Should(BeNil())
				uncompressed = decompressCM(testCm)
				Expect(uncompressed).Should(Equal(expectedYaml))
				Expect(testCm.OwnerReferences).Should(HaveLen(1))
			})

			It("should create the compressed FBCRegistryPod successfully", func() {
				expectedPodName := "quay-io-example-example-operator-bundle-0-2-0"
				Expect(rp).NotTo(BeNil())
				Expect(rp.pod.Name).To(Equal(expectedPodName))
				Expect(rp.pod.Namespace).To(Equal(rp.cfg.Namespace))
				Expect(rp.pod.Spec.Containers[0].Name).To(Equal(defaultContainerName))
				Expect(rp.pod.Spec.Containers).Should(HaveLen(1))
				Expect(rp.pod.Spec.Containers[0].Ports).Should(HaveLen(1))
				Expect(rp.pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(rp.GRPCPort))
				Expect(rp.pod.Spec.Containers[0].Command).Should(HaveLen(3))
				Expect(rp.pod.Spec.Containers[0].Command).Should(ContainElements("sh", "-c", containerCommandFor(rp.FBCIndexRootDir, rp.GRPCPort)))
				Expect(rp.pod.Spec.InitContainers).Should(HaveLen(1))
				Expect(rp.pod.Spec.InitContainers[0].VolumeMounts).Should(HaveLen(2))
			})
		})
	})
})

func decompressCM(cm *corev1.ConfigMap) string {
	actualBinaryData := cm.BinaryData["extraFBC"]
	ExpectWithOffset(1, len(actualBinaryData)).Should(BeNumerically("<", maxConfigMapSize))
	By("uncompress the BinaryData")
	compressed := bytes.NewBuffer(actualBinaryData)
	reader, err := gzip.NewReader(compressed)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	var uncompressed bytes.Buffer
	ExpectWithOffset(1, reader.Close()).Should(Succeed())
	_, err = io.Copy(&uncompressed, reader)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	return uncompressed.String()
}

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
const charTbl = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+=*&^%$#@!,.;~/\\|"

var rnd = rand.New(rand.NewSource(time.Now().UnixMilli()))

func randField() string {

	fieldNameLength := rnd.Intn(15) + 5
	fieldName := make([]byte, fieldNameLength)
	for i := 0; i < fieldNameLength; i++ {
		fieldName[i] = charTbl[rnd.Intn('z'-'a'+1)]
	}

	// random field name between 5 and 45
	size := rnd.Intn(40) + 5

	value := make([]byte, size)
	for i := 0; i < size; i++ {
		value[i] = charTbl[rnd.Intn(len(charTbl))]
	}
	return fmt.Sprintf("%s: %q\n", fieldName, value)
}

func generateRandYaml() string {
	numLines := rnd.Intn(45) + 5

	b := strings.Builder{}
	b.WriteString("---\n")
	for i := 0; i < numLines; i++ {
		b.WriteString(randField())
	}
	return b.String()
}

var (
	compressBuff = &bytes.Buffer{}
	compressor   = gzip.NewWriter(compressBuff)
)

func compress(s string) []byte {
	compressBuff.Reset()
	compressor.Reset(compressBuff)

	input := bytes.NewBufferString(s)
	_, err := io.Copy(compressor, input)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	Expect(compressor.Flush()).Should(Succeed())
	Expect(compressor.Close()).Should(Succeed())

	return compressBuff.Bytes()
}
