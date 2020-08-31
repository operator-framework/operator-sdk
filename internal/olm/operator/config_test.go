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

package operator_test

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

const (
	defaultHost       = "http://localhost:8080"
	defaultNamespace  = "my-default"
	customHost        = "https://custom:8443"
	customNamespace   = "my-custom"
	overrideNamespace = "my-override"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		var (
			c           operator.Configuration
			defaultFile *os.File
		)
		BeforeEach(func() {
			c = operator.Configuration{}

			var err error
			defaultFile, err = ioutil.TempFile("", "operator-sdk-test-kubeconfig")
			Expect(err).To(BeNil())
			Expect(ioutil.WriteFile(defaultFile.Name(), genKubeconfig(defaultHost, defaultNamespace), 0644)).To(Succeed())
			Expect(os.Unsetenv("KUBECONFIG")).To(Succeed())
		})

		AfterEach(func() {
			os.Remove(defaultFile.Name())
		})

		Context("with no kubeconfig override path or environment variable", func() {
			It("should fail with non-existent file", func() {
				clientcmd.RecommendedHomeFile = "/tmp/operator-sdk-kubeconfig-does-not-exist"
				Expect(c.Load()).To(HaveOccurred())
			})
			It("should load the default kubeconfig", func() {
				clientcmd.RecommendedHomeFile = defaultFile.Name()
				Expect(c.Load()).To(Succeed())
				verifyKubeconfig(c, defaultNamespace, defaultHost)
			})
			It("should override the namespace", func() {
				clientcmd.RecommendedHomeFile = defaultFile.Name()
				c.Namespace = overrideNamespace
				Expect(c.Load()).To(Succeed())
				verifyKubeconfig(c, overrideNamespace, defaultHost)
			})
		})

		Context("with kubeconfig override", func() {
			var customFile *os.File

			BeforeEach(func() {
				var err error
				customFile, err = ioutil.TempFile("", "operator-sdk-test-kubeconfig")
				Expect(err).To(BeNil())
				Expect(ioutil.WriteFile(customFile.Name(), genKubeconfig(customHost, customNamespace), 0644)).To(Succeed())
				Expect(os.Unsetenv("KUBECONFIG")).To(Succeed())
				clientcmd.RecommendedHomeFile = defaultFile.Name()

				c = operator.Configuration{}
			})

			AfterEach(func() {
				os.Remove(customFile.Name())
			})

			Context("path", func() {
				It("should fail with non-existent file", func() {
					c.KubeconfigPath = "/tmp/operator-sdk-kubeconfig-does-not-exist"
					Expect(c.Load()).To(HaveOccurred())
				})
				It("should load the custom kubeconfig", func() {
					c.KubeconfigPath = customFile.Name()
					Expect(c.Load()).To(Succeed())
					Expect(c.Namespace).To(Equal(customNamespace))
					verifyKubeconfig(c, customNamespace, customHost)
				})
				It("should override the namespace", func() {
					c.KubeconfigPath = customFile.Name()
					c.Namespace = overrideNamespace
					Expect(c.Load()).To(Succeed())
					verifyKubeconfig(c, overrideNamespace, customHost)
				})
			})
			Context("environment variable", func() {
				It("should fail with non-existent file", func() {
					Expect(os.Setenv("KUBECONFIG", "/tmp/operator-sdk-kubeconfig-does-not-exist")).To(Succeed())
					Expect(c.Load()).To(HaveOccurred())
				})
				It("should load the custom kubeconfig", func() {
					Expect(os.Setenv("KUBECONFIG", customFile.Name())).To(Succeed())
					Expect(c.Load()).To(Succeed())
					Expect(c.Namespace).To(Equal(customNamespace))
					verifyKubeconfig(c, customNamespace, customHost)
				})
				It("should override the namespace", func() {
					Expect(os.Setenv("KUBECONFIG", customFile.Name())).To(Succeed())
					c.Namespace = overrideNamespace
					Expect(c.Load()).To(Succeed())
					verifyKubeconfig(c, overrideNamespace, customHost)
				})
			})
		})
	})
})

func verifyKubeconfig(c operator.Configuration, expectedNs, expectedHost string) {
	Expect(c.Client).NotTo(BeNil())
	Expect(c.RESTConfig).NotTo(BeNil())
	Expect(c.RESTConfig.Host).To(Equal(expectedHost))
	Expect(c.Namespace).To(Equal(expectedNs))
}

func genKubeconfig(host, namespace string) []byte {
	return []byte(fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    server: %s
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    namespace: %s
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
`, host, namespace))
}
