package configmap

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
	"testing"

	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestConfigmap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configmap Suite")
}

var _ = Describe("Testing configmap.go", func() {

	Describe("Hash function", func() {
		It("tests the hash function", func() {
			val := []byte("Hello")
			h := sha256.New()
			_, _ = h.Write(val)
			enc := base32.StdEncoding.WithPadding(base32.NoPadding)
			ans := enc.EncodeToString(h.Sum(nil))

			Expect(hashContents([]byte("Hello"))).Should(Equal(ans))
		})
	})

	Describe("Return the registry name for the configmap", func() {
		It("returns the registry configmap name", func() {
			val := "Test"
			name := k8sutil.FormatOperatorNameDNS1123(val)
			ans := fmt.Sprintf("%s-registry-manifests", name)

			Expect(getRegistryConfigMapName("Test")).Should(Equal(ans))
		})
	})

	Describe("Return a unique yaml filename", func() {

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

		It("returns the filename with extra  usergiven string", func() {

			for _, name := range userInput {
				if name != "" {
					fileName += strings.ToLower(name) + "."
				}
			}
			fileName = fileName + "yaml"
			Expect(makeObjectFileName(testVal, userInput...)).Should(Equal(fileName))

		})

		It("returns the filename without extra  usergiven string", func() {
			fileName = fileName + "yaml"
			Expect(makeObjectFileName(testVal)).Should(Equal(fileName))
		})
	})

	Describe("Adding object to binary data ", func() {
		It("Test", func() {
			type binaryData map[string][]byte
			userInput := []string{"userInput", "userInput2"}

			type test_struct struct {
				val1 string
				val2 string
			}
			b := make(binaryData)
			obj := test_struct{"val1", "val2"}

			Expect(addObjectToBinaryData(b, obj, userInput...)).Should(BeNil())
		})

	})

	Describe("Creating the binary data", func() {
		It("Test", func() {

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

	Describe("BUNDLE", func() {
		It("Test", func() {

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

			Expect(e).Should(BeNil())
			Expect(err).Should(BeNil())
			Expect(binaryData).Should(Equal(val))

		})
	})

	Describe("Package manifest", func() {
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

			fmt.Printf("%+v", e)
			Expect(e).Should(BeNil())
			Expect(err).Should(BeNil())
			Expect(binaryDataByConfigMap).Should(Equal(val))

		})

	})
})
