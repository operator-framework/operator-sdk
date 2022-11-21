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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testIndexImageTag = "some-image:v1.2.3"
const caSecretName = "foo-secret"

// newFakeClient() returns a fake controller runtime client
func newFakeClient() client.Client {
	return fakeclient.NewClientBuilder().Build()
}

func TestCreateRegistryPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Registry Pod Suite")
}

var _ = Describe("FBCRegistryPod", func() {

	var defaultBundleItems = []index.BundleItem{{
		ImageTag: "quay.io/example/example-operator-bundle:0.2.0",
		AddMode:  index.SemverBundleAddMode,
	}}

	Describe("creating registry pod", func() {
		Context("with valid registry pod values", func() {
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
				cfg = &operator.Configuration{
					Client:    newFakeClient(),
					Namespace: "test-default",
				}
				rp = &FBCRegistryPod{
					BundleItems: defaultBundleItems,
					IndexImage:  testIndexImageTag,
				}
				By("initializing the FBCRegistryPod")
				Expect(rp.init(cfg, cs)).To(Succeed())
			})

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
				Expect(err).To(BeNil())
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
				Expect(err).To(BeNil())
				Expect(output).Should(Equal(containerCommandFor(rp2.FBCIndexRootDir, rp2.GRPCPort)))
			})
		})

		Context("with invalid registry pod values", func() {
			var (
				cfg *operator.Configuration
				cs  *v1alpha1.CatalogSource
			)
			BeforeEach(func() {
				cs = &v1alpha1.CatalogSource{
					ObjectMeta: v1.ObjectMeta{
						Name: "test-catalogsource",
					},
				}
				cfg = &operator.Configuration{
					Client:    newFakeClient(),
					Namespace: "test-default",
				}
			})

			It("should error when bundle image is not provided", func() {
				expectedErr := "bundle image set cannot be empty"
				rp := &FBCRegistryPod{}
				err := rp.init(cfg, cs)
				Expect(err).NotTo(BeNil())
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
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})
		})
	})
})

// containerCommandFor returns the expected container command for a db path and set of bundle items.
func containerCommandFor(indexRootDir string, grpcPort int32) string { //nolint:unparam
	return fmt.Sprintf("opm serve %s -p %d", indexRootDir, grpcPort)
}
