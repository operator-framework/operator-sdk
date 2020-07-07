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

// Modified from https://github.com/kubernetes-sigs/kubebuilder/tree/39224f0/test/e2e/v3

package e2e

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint
	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"

	testutils "github.com/operator-framework/operator-sdk/test/internal"
	"github.com/operator-framework/operator-sdk/version"
)

var _ = Describe("operator-sdk", func() {
	Context("with the new project layout", func() {
		var (
			tc testutils.TestContext
			//operatorVersion = "0.0.1"
		)

		BeforeEach(func() {
			By("creating a new test context")
			var err error
			tc, err = testutils.NewTestContext("GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(tc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("deleting the CR instance")
			// currently controller-runtime doesn't provide a readiness probe, we retry a few times
			// we can change it to probe the readiness endpoint after CR supports it.
			sampleFile := filepath.Join("config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))
			Eventually(func() error {
				_, err := tc.Kubectl.Delete(true, "-f", sampleFile)
				return err
			}, time.Minute, time.Second).Should(Succeed())
			By("cleaning up created API objects during test process")
			tc.CleanupManifests(filepath.Join("config", "default"))

			By("removing container image and work dir")
			tc.Destroy()
		})

		It("should generate a runnable project", func() {
			var controllerPodName string
			By("initializing a project")
			err := tc.Init(
				"--plugins", "helm.operator-sdk.io/v1",
				"--project-version", "3-alpha",
				"--domain", tc.Domain)
			Expect(err).Should(Succeed())

			By("creating an API definition")
			err = tc.CreateAPI(
				"--group", tc.Group,
				"--version", tc.Version,
				"--kind", tc.Kind)
			Expect(err).Should(Succeed())

			By("replace Dockerfile to use the dev tag")
			version := strings.TrimSuffix(version.Version, "+git")
			testutils.ReplaceInFile(filepath.Join(tc.Dir, "Dockerfile"), version, "dev")

			By("building the operator image")
			err = tc.Make("docker-build", "IMG="+tc.ImageName)
			Expect(err).Should(Succeed())

			By("loading the operator image into the test cluster")
			err = tc.LoadImageToKindCluster()
			Expect(err).Should(Succeed())

			By("install CRDs")
			err = tc.Make("install")
			Expect(err).Should(Succeed())

			By("deploying the controller manager")
			err = tc.Make("deploy", "IMG="+tc.ImageName)
			Expect(err).Should(Succeed())

			By("ensuring the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				podOutput, err := tc.Kubectl.Get(
					true,
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}")
				Expect(err).NotTo(HaveOccurred())
				podNames := kbtestutils.GetNonEmptyLines(podOutput)
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				Expect(controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				status, err := tc.Kubectl.Get(
					true,
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
				Expect(err).NotTo(HaveOccurred())
				if status != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			Eventually(verifyControllerUp, time.Minute, time.Second).Should(Succeed())

			By("creating an instance of CR")
			// currently controller-runtime doesn't provide a readiness probe, we retry a few times
			// we can change it to probe the readiness endpoint after CR supports it.
			sampleFile := filepath.Join("config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))
			Eventually(func() error {
				_, err = tc.Kubectl.Apply(true, "-f", sampleFile)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("ensuring the created resource object gets reconciled in controller")
			managerContainerLogs := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Reconciled release"))

			By("generating the operator bundle")
			// Turn off interactive prompts for all generation tasks.
			replace := "operator-sdk generate kustomize manifests"
			testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
		})
	})
})
