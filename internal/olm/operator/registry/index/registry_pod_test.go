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
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	Describe("creating registry pod", func() {

		Context("with valid registry pod values", func() {
			expectedPodName := "quay-io-example-example-operator-bundle-0-2-0"
			expectedOutput := "/bin/mkdir -p /database &&" +
				"/bin/opm registry add -d /database/index.db -b quay.io/example/example-operator-bundle:0.2.0 --mode=semver &&" +
				"/bin/opm registry serve -d /database/index.db -p 50051"

			var rp *RegistryPod
			var cfg *operator.Configuration
			var err error

			BeforeEach(func() {
				cfg = &operator.Configuration{
					Client:    newFakeClient(),
					Namespace: "test-default",
				}
				rp, err = NewRegistryPod(cfg, "/database/index.db", "quay.io/example/example-operator-bundle:0.2.0")
				Expect(err).To(BeNil())
			})

			It("should validate the RegistryPod successfully", func() {
				err := rp.validate()

				Expect(err).To(BeNil())
			})

			It("should create the RegistryPod successfully", func() {
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

			It("should return a valid container command", func() {
				output, err := rp.getContainerCmd()

				Expect(err).To(BeNil())
				Expect(output).Should(Equal(expectedOutput))
			})

			It("should return a pod definition successfully", func() {
				rp.pod, err = rp.podForBundleRegistry()

				Expect(rp.pod).NotTo(BeNil())
				Expect(rp.pod.Name).To(Equal(expectedPodName))
				Expect(rp.pod.Namespace).To(Equal(rp.cfg.Namespace))
				Expect(rp.pod.Spec.Containers[0].Name).To(Equal(defaultContainerName))
				if len(rp.pod.Spec.Containers) > 0 {
					if len(rp.pod.Spec.Containers[0].Ports) > 0 {
						Expect(rp.pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(rp.GRPCPort))
					}
				}
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
				expectedErr := "bundle image cannot be empty"

				_, err := NewRegistryPod(cfg, "/database/index.db", "")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not create a registry pod when database path is not provided", func() {
				expectedErr := "registry database path cannot be empty"

				_, err := NewRegistryPod(cfg, "",
					"quay.io/example/example-operator-bundle:0.2.0")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not create a registry pod when bundle add mode is empty", func() {
				expectedErr := "bundle add mode cannot be empty"

				rp, _ := NewRegistryPod(cfg, "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0")
				rp.BundleAddMode = ""

				err := rp.validate()
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not accept any other bundle add mode other than semver or replaces", func() {
				expectedErr := "invalid bundle mode"

				rp, _ := NewRegistryPod(cfg, "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0")
				rp.BundleAddMode = "invalid"

				err := rp.validate()
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("checkPodStatus should return error when pod check is false and context is done", func() {
				rp, _ := NewRegistryPod(cfg, "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0")

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

			It("Create should fail when registry pod is not initialized", func() {
				rp := RegistryPod{}
				cs := &v1alpha1.CatalogSource{}
				pod, err := rp.Create(context.Background(), cs)

				Expect(err).NotTo(BeNil())
				Expect(pod).To(BeNil())
				Expect(err).To(MatchError(errPodNotInit))
			})

		})
	})
})
