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

package olm

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// newFakeClient() returns a clientset
func newFakeClient() kubernetes.Interface {
	return fake.NewSimpleClientset()
}
func TestCreateRegistryPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Registry Pod Suite")
}

var _ = Describe("RegistryPod", func() {

	Describe("creating registry pod", func() {

		Context("with valid registry pod values", func() {
			expectedPodName := "quay-io-example-example-operator-bundle-0-2-0"
			expectedOutput := "/bin/mkdir -p index.db &&" +
				"/bin/opm registry add -d index.db -b quay.io/example/example-operator-bundle:0.2.0 --mode=semver &&" +
				"/bin/opm registry serve -d index.db -p 50051"

			var rp *RegistryPod
			var err error

			BeforeEach(func() {
				rp, err = NewRegistryPod(newFakeClient(), "/database/index.db", "quay.io/example/example-operator-bundle:0.2.0", "default")
				Expect(err).To(BeNil())
			})

			It("should validate the RegistryPod successfully", func() {
				err := rp.validate()

				Expect(err).To(BeNil())
			})

			It("should create the RegistryPod successfully", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.pod.Name).To(Equal(expectedPodName))
				Expect(rp.pod.Namespace).To(Equal(rp.Namespace))
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
				Expect(rp.pod.Namespace).To(Equal(rp.Namespace))
				Expect(rp.pod.Spec.Containers[0].Name).To(Equal(defaultContainerName))
				if len(rp.pod.Spec.Containers) > 0 {
					if len(rp.pod.Spec.Containers[0].Ports) > 0 {
						Expect(rp.pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(rp.GRPCPort))
					}
				}
			})

			It("should create registry pod successfully", func() {
				err := rp.Create(context.Background())

				Expect(err).To(BeNil())
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

			It("should error when bundle image is not provided", func() {
				expectedErr := "bundle image cannot be empty"

				_, err := NewRegistryPod(newFakeClient(), "/database/index.db",
					"", "default")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not create a registry pod when namespace is not provided", func() {
				expectedErr := "namespace cannot be empty"

				_, err := NewRegistryPod(newFakeClient(), "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0", "")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not create a registry pod when database path is not provided", func() {
				expectedErr := "registry database path cannot be empty"

				_, err := NewRegistryPod(newFakeClient(), "",
					"quay.io/example/example-operator-bundle:0.2.0", "default")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not create a registry pod when bundle add mode is empty", func() {
				expectedErr := "bundle add mode cannot be empty"

				rp, _ := NewRegistryPod(newFakeClient(), "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0", "default")
				rp.BundleAddMode = ""

				err := rp.validate()
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not accept any other bundle add mode other than semver or replaces", func() {
				expectedErr := "invalid bundle mode"

				rp, _ := NewRegistryPod(newFakeClient(), "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0", "default")
				rp.BundleAddMode = "invalid"

				err := rp.validate()
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("checkPodStatus should return error when pod check is false and context is done", func() {
				rp, _ := NewRegistryPod(newFakeClient(), "/database/index.db",
					"quay.io/example/example-operator-bundle:0.2.0", "default")

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
				expectedErr := "internal error: uninitialized RegistryPod cannot be used"

				err := rp.Create(context.Background())

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not be able to get pod logs if pod is not initialized", func() {
				rp := RegistryPod{}
				expectedErr := "a registry pod must be created before getting pod logs"

				_, err := rp.GetLogs(context.Background())

				Expect(err).ToNot(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			// todo(rashmigottipati): add test to check VerifyPodRunning returning error
		})
	})
})
