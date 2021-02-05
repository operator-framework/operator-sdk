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

package index

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

// newFakeClient() returns a fake controller runtime client
func newFakeClient() client.Client {
	return fakeclient.NewFakeClient()
}

func TestCreateRegistryPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Registry Pod Suite")
}

var _ = Describe("RegistryPod", func() {

	var defaultBundleItem = BundleItem{
		ImageTag: "quay.io/example/example-operator-bundle:0.2.0",
		AddMode:  SemverBundleAddMode,
	}

	Describe("creating registry pod", func() {

		Context("with valid registry pod values", func() {

			var rp *RegistryPod
			var cfg *operator.Configuration

			BeforeEach(func() {
				cfg = &operator.Configuration{
					Client:    newFakeClient(),
					Namespace: "test-default",
				}
				rp = &RegistryPod{BundleItems: []BundleItem{defaultBundleItem}}
				By("initializing the RegistryPod")
				Expect(rp.init(cfg)).To(Succeed())
			})

			It("should create the RegistryPod successfully", func() {
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
				Expect(rp.DBPath).To(Equal("/database/index.db"))
			})

			It("should return a valid container command for one image", func() {
				output, err := rp.getContainerCmd()
				Expect(err).To(BeNil())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, []BundleItem{defaultBundleItem})))
			})

			It("should return a valid container command for three images", func() {
				bundleItems := []BundleItem{
					defaultBundleItem,
					{
						ImageTag: "quay.io/example/example-operator-bundle:0.3.0",
						AddMode:  ReplacesBundleAddMode,
					},
					{
						ImageTag: "quay.io/example/example-operator-bundle:1.0.1",
						AddMode:  SemverBundleAddMode,
					},
				}
				rp2 := RegistryPod{
					DBPath:      defaultDBPath,
					GRPCPort:    defaultGRPCPort,
					BundleItems: bundleItems,
				}
				output, err := rp2.getContainerCmd()
				Expect(err).To(BeNil())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, bundleItems)))
			})

			It("check pod status should return successfully when pod check is true", func() {
				mockGoodPodCheck := wait.ConditionFunc(func() (done bool, err error) {
					return true, nil
				})

				err := rp.checkPodStatus(context.Background(), mockGoodPodCheck)

				Expect(err).To(BeNil())
			})
		})

		Context("with invalid registry pod values", func() {
			var cfg *operator.Configuration
			BeforeEach(func() {
				cfg = &operator.Configuration{
					Client:    newFakeClient(),
					Namespace: "test-default",
				}
			})

			It("should error when bundle image is not provided", func() {
				expectedErr := "bundle image set cannot be empty"
				rp := &RegistryPod{}
				err := rp.init(cfg)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not accept any other bundle add mode other than semver or replaces", func() {
				expectedErr := "invalid bundle mode"
				rp := &RegistryPod{
					BundleItems: []BundleItem{{ImageTag: "quay.io/example/example-operator-bundle:0.2.0", AddMode: "invalid"}},
				}
				err := rp.init(cfg)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("checkPodStatus should return error when pod check is false and context is done", func() {
				rp := &RegistryPod{
					BundleItems: []BundleItem{defaultBundleItem},
				}
				Expect(rp.init(cfg)).To(Succeed())

				mockBadPodCheck := wait.ConditionFunc(func() (done bool, err error) {
					return false, fmt.Errorf("error waiting for registry pod")
				})

				expectedErr := "error waiting for registry pod"
				// create a new context with a deadline of 1 millisecond
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
				cancel()

				err := rp.checkPodStatus(ctx, mockBadPodCheck)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})
		})
	})
})

// containerCommandFor returns the expected container command for a db path and set of bundle items.
func containerCommandFor(dbPath string, items []BundleItem) string {
	additions := &strings.Builder{}
	for _, item := range items {
		additions.WriteString(fmt.Sprintf("/bin/opm registry add -d %s -b %s --mode=%s && \\\n", dbPath, item.ImageTag, item.AddMode))
	}
	return fmt.Sprintf("/bin/mkdir -p /database && \\\n%s/bin/opm registry serve -d /database/index.db -p 50051\n", additions.String())
}
