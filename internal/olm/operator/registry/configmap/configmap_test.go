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

package configmap

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	client_cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var _ = Describe("ConfigMap", func() {

	Describe("hashContents", func() {
		It("should return the hash of the byte array without any error", func() {
			val := []byte("Hello")
			h := sha256.New()
			_, _ = h.Write(val)
			enc := base32.StdEncoding.WithPadding(base32.NoPadding)
			ans := enc.EncodeToString(h.Sum(nil))

			Expect(hashContents(val)).Should(Equal(ans))
		})
	})

	Describe("getRegistryConfigMapName", func() {
		It("should return the registry configmap name", func() {
			val := "Test"
			name := k8sutil.FormatOperatorNameDNS1123(val)
			ans := fmt.Sprintf("%s-registry-manifests", name)

			Expect(getRegistryConfigMapName(val)).Should(Equal(ans))
		})
	})

	Describe("makeObjectFileName", func() {
		var (
			fileName  string
			testVal   []byte
			userInput []string
		)
		BeforeEach(func() {
			testVal = []byte("test")
			userInput = []string{"userInput", "userInput2"}
			fileName = hashContents(testVal) + "."
		})
		It("returns the filename with extra user given string", func() {
			for _, name := range userInput {
				fileName += strings.ToLower(name) + "."
			}
			fileName = fileName + "yaml"
			Expect(makeObjectFileName(testVal, userInput...)).Should(Equal(fileName))
		})
		It("returns the filename without extra user given string", func() {
			fileName = fileName + "yaml"
			Expect(makeObjectFileName(testVal)).Should(Equal(fileName))
		})
	})

	Describe("addObjectToBinaryData ", func() {
		It("should add given object to the given binaryData", func() {
			userInput := []string{"userInput", "userInput2"}

			b := make(map[string][]byte)
			obj := struct {
				val1 string
				val2 string
			}{
				val1: "val1",
				val2: "val2",
			}

			binaryData := make(map[string][]byte)
			expected, err := yaml.Marshal(obj)
			Expect(err).ShouldNot(HaveOccurred())
			binaryData[makeObjectFileName(expected, userInput...)] = expected
			// Test and verify function
			Expect(addObjectToBinaryData(b, obj, userInput...)).Should(Succeed())
			Expect(b).Should(Equal(binaryData))
		})

	})

	Describe("makeObjectBinaryData", func() {
		It("creates the binary data", func() {
			binaryData := make(map[string][]byte)
			obj := struct {
				val1 string
				val2 string
			}{
				val1: "val1",
				val2: "val2",
			}

			userInput := []string{"userInput", "userInput2"}
			b, e := makeObjectBinaryData(obj, userInput...)
			Expect(e).ShouldNot(HaveOccurred())
			// Test and verify function
			Expect(addObjectToBinaryData(binaryData, obj, userInput...)).Should(Succeed())
			Expect(b).Should(Equal(binaryData))

		})
	})

	Describe("makeBundleBinaryData", func() {
		It("should serialize bundle to binary data", func() {
			b := apimanifests.Bundle{
				Name: "testbundle",
				Objects: []*unstructured.Unstructured{
					{
						Object: map[string]interface{}{"val1": "val1"},
					},
					{
						Object: map[string]interface{}{"val2": "va2"},
					},
				},
			}

			binaryData, err := makeBundleBinaryData(&b)
			Expect(err).ShouldNot(HaveOccurred())
			val := make(map[string][]byte)
			for _, obj := range b.Objects {
				Expect(addObjectToBinaryData(val, obj, obj.GetName(), obj.GetKind())).Should(Succeed())
			}

			Expect(binaryData).Should(Equal(val))
		})
	})

	Describe("makeConfigMapsForPackageManifests", func() {
		var (
			p apimanifests.PackageManifest
			e error
			b []*apimanifests.Bundle
		)
		BeforeEach(func() {
			b = []*apimanifests.Bundle{
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
				{
					Name: "testbundle_2",
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
			}
			p = apimanifests.PackageManifest{
				PackageName: "test_package_manifest",
				Channels: []apimanifests.PackageChannel{
					{Name: "test_1",
						CurrentCSVName: "test_csv_1",
					},
					{Name: "test_2",
						CurrentCSVName: "test_csv_2",
					},
				},
				DefaultChannelName: "test_channel_name",
			}
		})
		It("should serialize packagemanifest to binary data", func() {
			binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(&p, b)
			Expect(err).ShouldNot(HaveOccurred())

			val := make(map[string]map[string][]byte)
			cmName := getRegistryConfigMapName(p.PackageName) + "-package"
			val[cmName], err = makeObjectBinaryData(p)
			Expect(err).ShouldNot(HaveOccurred())
			for _, bundle := range b {
				version := bundle.CSV.Spec.Version.String()
				cmName := getRegistryConfigMapName(p.PackageName) + "-" + k8sutil.FormatOperatorNameDNS1123(version)
				val[cmName], e = makeBundleBinaryData(bundle)
				Expect(e).ShouldNot(HaveOccurred())
			}

			Expect(binaryDataByConfigMap).Should(Equal(val))
		})

	})

	Describe("getRegistryConfigMaps", func() {
		var (
			rr   RegistryResources
			list corev1.ConfigMapList
		)
		BeforeEach(func() {
			fakeclient := fake.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testns",
						Labels:    makeRegistryLabels("test"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testns2",
						Labels:    makeRegistryLabels("test"),
					},
				},
			).Build()
			rr = RegistryResources{
				Client: &client.Client{
					KubeClient: fakeclient,
				},
				Pkg: &apimanifests.PackageManifest{
					PackageName: "test",
				},
				Bundles: rr.Bundles,
			}

			list = corev1.ConfigMapList{}
		})
		It("performs operations and returns all the configmaps", func() {
			opts := []client_cr.ListOption{
				client_cr.MatchingLabels(makeRegistryLabels(rr.Pkg.PackageName)),
				client_cr.InNamespace("testns"),
			}
			Expect(rr.Client.KubeClient.List(context.TODO(), &list, opts...)).Should(Succeed())
			configmaps, err := rr.getRegistryConfigMaps(context.TODO(), "testns")
			Expect(err).ShouldNot(HaveOccurred())

			Expect(configmaps).Should(Equal(list.Items))
		})

	})
})
