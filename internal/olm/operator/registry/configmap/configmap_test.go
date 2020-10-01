package configmap

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
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
			// Expect(func() return_struct {
			// 	b, e := makeObjectBinaryData(obj, userInput...)
			// 	return return_struct{b, e}
			// }).Should(BeEquivalentTo(return_struct{binaryData, err}))
			Expect(b).Should(Equal(binaryData))
			Expect(e).Should(BeNil())
		})
	})

	Describe("BUNDLE", func() {
		It("Test", func() {

			// binaryData := make(map[string][]byte)
			// type return_struct struct {
			// 	val1 map[string][]byte
			// 	val2 error
			// }
			// type test_struct struct {
			// 	val1 string
			// 	val2 string
			// }

			// obj := test_struct{"val1", "val2"}
			// userInput := []string{"userInput", "userInput2"}

			// addObjectToBinaryData(binaryData, obj, userInput...)

			// b := make(map[string][]byte)
			// b, e := makeObjectBinaryData(obj, userInput...)
			// // Expect(func() return_struct {
			// // 	b, e := makeObjectBinaryData(obj, userInput...)
			// // 	return return_struct{b, e}
			// // }).Should(BeEquivalentTo(return_struct{binaryData, err}))
			// Expect(b).Should(Equal(binaryData))
			// Expect(e).Should(BeNil())

			b := apimanifests.Bundle{
				Name: "testbundle",
			}

			binaryData, err := makeBundleBinaryData(&b)

			fmt.Printf("%+v\n", binaryData)
			fmt.Printf("%+v\n", err)
		})
	})

})

// Bundle{
// 	Name: "testbundle",
// 	CSV: &operatorsv1alpha1.ClusterServiceVersion{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "testbundle",
// 			Namespace: "default",
// 		}
// 		Spec: {}
// 	},
// }
