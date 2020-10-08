package configmap

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Testing ConfigMap.go", func() {

	Describe("hashContents", func() {
		It("should return the hash of the byte array without any error", func() {
			val := []byte("Hello")
			h := sha256.New()
			_, _ = h.Write(val)
			enc := base32.StdEncoding.WithPadding(base32.NoPadding)
			ans := enc.EncodeToString(h.Sum(nil))

			Expect(hashContents([]byte("Hello"))).Should(Equal(ans))
		})
	})

	Describe("getRegistryConfigMapName", func() {
		It("should return the registry configmap name", func() {
			val := "Test"
			name := k8sutil.FormatOperatorNameDNS1123(val)
			ans := fmt.Sprintf("%s-registry-manifests", name)

			Expect(getRegistryConfigMapName("Test")).Should(Equal(ans))
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
				if name != "" {
					fileName += strings.ToLower(name) + "."
				}
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
			type binaryData map[string][]byte
			userInput := []string{"userInput", "userInput2"}

			type test_struct struct {
				val1 string
				val2 string
			}
			b := make(binaryData)
			fmt.Printf("%+v", b)
			obj := test_struct{"val1", "val2"}

			binaryData_local := make(binaryData)
			b_local, err := yaml.Marshal(obj)

			binaryData_local[makeObjectFileName(b_local, userInput...)] = b_local

			Expect(err).Should(BeNil())
			Expect(addObjectToBinaryData(b, obj, userInput...)).Should(BeNil())
			Expect(b).Should(Equal(binaryData_local))
		})

	})

	Describe("makeObjectBinaryData", func() {
		It("creates the binary data", func() {

			binaryData := make(map[string][]byte)
			type return_struct struct {
				val1 map[string][]byte
				val2 error
			}
			type test_struct struct {
				val1 string
				val2 string
			}

			obj := test_struct{"val1", "val2"}
			userInput := []string{"userInput", "userInput2"}

			addObjectToBinaryData(binaryData, obj, userInput...)

			b := make(map[string][]byte)
			b, e := makeObjectBinaryData(obj, userInput...)

			Expect(e).Should(BeNil())
			Expect(b).Should(Equal(binaryData))

		})
	})

	Describe("makeBundleBinaryData", func() {
		It("should serialize bundle to binary data", func() {

			var e error
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

			val := make(map[string][]byte)
			for _, obj := range b.Objects {
				e = addObjectToBinaryData(val, obj, obj.GetName(), obj.GetKind())
			}

			Expect(binaryData).Should(Equal(val))
			Expect(e).Should(BeNil())
			Expect(err).Should(BeNil())

		})
	})

	Describe("makeConfigMapsForPackageManifests", func() {
		It("Test", func() {

			var e error
			b := []*apimanifests.Bundle{
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

			p := apimanifests.PackageManifest{
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

			binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(&p, b)

			val := make(map[string]map[string][]byte)

			cmName := getRegistryConfigMapName(p.PackageName) + "-package"
			val[cmName], err = makeObjectBinaryData(p)

			// Create Bundle ConfigMaps.
			for _, bundle := range b {
				version := bundle.CSV.Spec.Version.String()
				e = fmt.Errorf("bundle ClusterServiceVersion %s has no version", bundle.CSV.GetName())

				cmName := getRegistryConfigMapName(p.PackageName) + "-" + k8sutil.FormatOperatorNameDNS1123(version)
				val[cmName], e = makeBundleBinaryData(bundle)

			}

			Expect(e).Should(BeNil())
			Expect(err).Should(BeNil())
			Expect(binaryDataByConfigMap).Should(Equal(val))

		})

	})

	Describe("getRegistryConfigMaps", func() {
		It("performs operations and returns all the configmaps", func() {
			var fclient RegistryResources

			fakeclient := fake.NewFakeClient(
				&corev1.ConfigMapList{
					Items: []corev1.ConfigMap{
						corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "testns",
							},
						},
						corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "testns2",
							},
						},
					},
				},
			)

			fclient = RegistryResources{
				Client: &client.Client{
					KubeClient: fakeclient,
				},
				Pkg: &apimanifests.PackageManifest{
					PackageName: "test",
				},
				Bundles: fclient.Bundles,
			}

			configmaps, err := fclient.getRegistryConfigMaps(context.TODO(), "testns")

			fmt.Printf("\n\n%v", configmaps)
			fmt.Printf("\n\n%v", err)

		})

	})
})
