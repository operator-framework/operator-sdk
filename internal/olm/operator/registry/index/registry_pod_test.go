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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	corev1 "k8s.io/api/core/v1"
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

var _ = Describe("SQLiteRegistryPod", func() {

	var defaultBundleItems = []BundleItem{{
		ImageTag: "quay.io/example/example-operator-bundle:0.2.0",
		AddMode:  SemverBundleAddMode,
	}}

	Describe("creating registry pod", func() {

		Context("with valid registry pod values", func() {

			var (
				rp  *SQLiteRegistryPod
				cfg *operator.Configuration
				pod *corev1.Pod
				err error
			)

			BeforeEach(func() {
				cfg = &operator.Configuration{
					Client:    newFakeClient(),
					Namespace: "test-default",
				}
				rp = &SQLiteRegistryPod{
					BundleItems: defaultBundleItems,
					IndexImage:  testIndexImageTag,
				}
				By("initializing the SQLiteRegistryPod")
				Expect(rp.init(cfg)).To(Succeed())
			})

			It("should create the SQLiteRegistryPod successfully", func() {
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
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, defaultBundleItems, false, rp.SkipTLSVerify, false)))
			})

			It("should return a container command with --ca-file", func() {
				rp.CASecretName = caSecretName
				output, err := rp.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, defaultBundleItems, true, rp.SkipTLSVerify, false)))
			})

			It("should return a container command for image with --skip-tls-verify", func() {
				if len(defaultBundleItems) > 0 {
					bundles := []BundleItem{defaultBundleItems[0]}
					rp.BundleItems = bundles
					rp.SkipTLSVerify = true
					output, err := rp.getContainerCmd()
					Expect(err).ToNot(HaveOccurred())
					Expect(output).Should(Equal(containerCommandFor(defaultDBPath, bundles, false, rp.SkipTLSVerify, false)))
				}
			})

			It("should return a valid container command for three images", func() {
				bundleItems := append(defaultBundleItems,
					BundleItem{
						ImageTag: "quay.io/example/example-operator-bundle:0.3.0",
						AddMode:  ReplacesBundleAddMode,
					},
					BundleItem{
						ImageTag: "quay.io/example/example-operator-bundle:1.0.1",
						AddMode:  SemverBundleAddMode,
					},
					BundleItem{
						ImageTag: "localhost/example-operator-bundle:1.0.1",
						AddMode:  SemverBundleAddMode,
					},
				)
				rp2 := SQLiteRegistryPod{
					DBPath:        defaultDBPath,
					GRPCPort:      defaultGRPCPort,
					BundleItems:   bundleItems,
					SkipTLSVerify: true,
				}
				output, err := rp2.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, bundleItems, false, rp2.SkipTLSVerify, false)))
			})

			It("should return a valid container command for one image", func() {
				output, err := rp.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, defaultBundleItems, false, false, rp.UseHTTP)))
			})

			It("should return a container command with --ca-file", func() {
				rp.CASecretName = caSecretName
				output, err := rp.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, defaultBundleItems, true, false, rp.UseHTTP)))
			})

			It("should return a container command for image with --use-http", func() {
				if len(defaultBundleItems) > 0 {
					bundles := []BundleItem{defaultBundleItems[0]}
					rp.BundleItems = bundles
					rp.UseHTTP = true
					output, err := rp.getContainerCmd()
					Expect(err).ToNot(HaveOccurred())
					Expect(output).Should(Equal(containerCommandFor(defaultDBPath, bundles, false, false, rp.UseHTTP)))
				}
			})

			It("should return a valid container command for three images", func() {
				bundleItems := append(defaultBundleItems,
					BundleItem{
						ImageTag: "quay.io/example/example-operator-bundle:0.3.0",
						AddMode:  ReplacesBundleAddMode,
					},
					BundleItem{
						ImageTag: "quay.io/example/example-operator-bundle:1.0.1",
						AddMode:  SemverBundleAddMode,
					},
					BundleItem{
						ImageTag: "localhost/example-operator-bundle:1.0.1",
						AddMode:  SemverBundleAddMode,
					},
				)
				rp2 := SQLiteRegistryPod{
					DBPath:      defaultDBPath,
					GRPCPort:    defaultGRPCPort,
					BundleItems: bundleItems,
					UseHTTP:     true,
				}
				output, err := rp2.getContainerCmd()
				Expect(err).ToNot(HaveOccurred())
				Expect(output).Should(Equal(containerCommandFor(defaultDBPath, bundleItems, false, false, rp2.UseHTTP)))
			})

			It("check pod status should return successfully when pod check is true", func() {
				mockGoodPodCheck := wait.ConditionFunc(func() (done bool, err error) {
					return true, nil
				})

				Expect(rp.checkPodStatus(context.Background(), mockGoodPodCheck)).To(Succeed())
			})

			It("adds secrets and a service account to the pod", func() {
				cfg.ServiceAccount = "foo"
				rp.SecretName = caSecretName

				pod, err = rp.podForBundleRegistry()
				Expect(err).NotTo((HaveOccurred()))
				Expect(pod.Spec.ServiceAccountName).To(Equal(cfg.ServiceAccount))
				Expect(pod.Spec.Volumes).To(Equal([]corev1.Volume{
					{
						Name: "foo-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "foo-secret",
								DefaultMode: newInt32(0400),
								Optional:    newBool(false),
								Items: []corev1.KeyToPath{
									{Key: ".dockerconfigjson", Path: ".docker/config.json"},
								},
							},
						},
					},
				}))
				for _, container := range pod.Spec.Containers {
					Expect(container.VolumeMounts).To(Equal([]corev1.VolumeMount{
						{Name: "foo-secret", ReadOnly: true, MountPath: "/root"},
					}))
				}
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
				rp := &SQLiteRegistryPod{}
				err := rp.init(cfg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("should not accept any other bundle add mode other than semver or replaces", func() {
				expectedErr := `bundle add mode "invalid" does not exist`
				rp := &SQLiteRegistryPod{
					BundleItems: []BundleItem{{ImageTag: "quay.io/example/example-operator-bundle:0.2.0", AddMode: "invalid"}},
				}
				err := rp.init(cfg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})

			It("checkPodStatus should return error when pod check is false and context is done", func() {
				rp := &SQLiteRegistryPod{
					BundleItems: defaultBundleItems,
					IndexImage:  testIndexImageTag,
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
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(expectedErr))
			})
		})
	})
})

// containerCommandFor returns the expected container command for a db path and set of bundle items.
func containerCommandFor(dbPath string, items []BundleItem, hasCA, skipTLSVerify bool, useHTTP bool) string { //nolint:unparam
	var caFlag string
	if hasCA {
		caFlag = " --ca-file=/certs/cert.pem"
	}
	additions := &strings.Builder{}
	for _, item := range items {
		additions.WriteString(fmt.Sprintf("opm registry add -d /tmp/tmp.db -b %s --mode=%s%s --skip-tls-verify=%v --use-http=%v && \\\n", item.ImageTag, item.AddMode, caFlag, skipTLSVerify, useHTTP))
	}

	return fmt.Sprintf("[[ -f %s ]] && cp %s /tmp/tmp.db; \\\n%sopm registry serve -d /tmp/tmp.db -p 50051\n", dbPath, dbPath, additions.String())
}
