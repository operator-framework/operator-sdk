/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"github.com/example/memcached-operator/test/utils"
)

// constant parts of the file
const namespace = "memcached-operator-system"

var _ = Describe("memcached", Ordered, func() {
	BeforeAll(func() {
		// The prometheus and the certmanager are installed in this test
		// because the Memcached sample has this option enable and
		// when we try to apply the manifests both will be required to be installed
		By("installing prometheus operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed())

		By("installing the cert-manager")
		Expect(utils.InstallCertManager()).To(Succeed())

		// The namespace can be created when we run make install
		// However, in this test we want ensure that the solution
		// can run in a ns labeled as restricted. Therefore, we are
		// creating the namespace an lebeling it.
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)

		// Now, let's ensure that all namespaces can raise an Warn when we apply the manifests
		// and that the namespace where the Operator and Operand will run are enforced as
		// restricted so that we can ensure that both can be admitted and run with the enforcement
		By("labeling all namespaces to warn when we apply the manifest if would violate the PodStandards")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", "--all",
			"pod-security.kubernetes.io/audit=restricted",
			"pod-security.kubernetes.io/enforce-version=v1.24",
			"pod-security.kubernetes.io/warn=restricted")
		_, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("labeling enforce the namespace where the Operator and Operand(s) will run")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/audit=restricted",
			"pod-security.kubernetes.io/enforce-version=v1.24",
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).To(Not(HaveOccurred()))
	})

	AfterAll(func() {
		By("uninstalling the Prometheus manager bundle")
		utils.UninstallPrometheusOperator()

		By("uninstalling the cert-manager bundle")
		utils.UninstallCertManager()

		By("removing manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("Memcached Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error
			projectDir, _ := utils.GetProjectDir()

			// operatorImage stores the name of the image used in the example
			var operatorImage = "example.com/memcached-operator:v0.0.1"

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", operatorImage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(operatorImage)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", operatorImage))
			outputMake, err := utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that manager Pod/container(s) are restricted")
			ExpectWithOffset(1, outputMake).NotTo(ContainSubstring("Warning: would violate PodSecurity"))

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)
				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

			By("creating an instance of the Memcached Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"config/samples/cache_v1alpha1_memcached.yaml"), "-n", namespace)
				_, err = utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validating that pod(s) status.phase=Running")
			getMemcachedPodStatus := func() error {
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "app.kubernetes.io/name=Memcached",
					"-o", "jsonpath={.items[*].status}", "-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "\"phase\":\"Running\"") {
					return fmt.Errorf("memcached pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, getMemcachedPodStatus, time.Minute, time.Second).Should(Succeed())

			By("validating that the status of the custom resource created is updated or not")
			getStatus := func() error {
				cmd = exec.Command("kubectl", "get", "memcached",
					"memcached-sample", "-o", "jsonpath={.status.conditions}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if !strings.Contains(string(status), "Available") {
					return fmt.Errorf("status condition with type Available should be set")
				}
				return nil
			}
			Eventually(getStatus, time.Minute, time.Second).Should(Succeed())
		})
	})
})
