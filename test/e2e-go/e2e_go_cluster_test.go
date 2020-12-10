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

package e2e_go_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kbtestutils "sigs.k8s.io/kubebuilder/v2/test/e2e/utils"
)

var _ = Describe("operator-sdk", func() {
	var controllerPodName string
	var metricsClusterRoleBindingName string

	Context("built with operator-sdk", func() {
		BeforeEach(func() {
			metricsClusterRoleBindingName = fmt.Sprintf("%s-metrics-reader", tc.ProjectName)
			By("deploying project on the cluster")
			err := tc.Make("deploy", "IMG="+tc.ImageName)
			Expect(err).NotTo(HaveOccurred())

		})
		AfterEach(func() {
			By("deleting Curl Pod created")
			_, _ = tc.Kubectl.Delete(false, "pod", "curl")

			By("cleaning up permissions")
			_, _ = tc.Kubectl.Command("delete", "clusterrolebinding",
				metricsClusterRoleBindingName)

			By("cleaning up created API objects during test process")
			tc.CleanupManifests(filepath.Join("config", "default"))

			By("ensuring that the namespace was deleted")
			verifyNamespaceDeleted := func() error {
				_, err := tc.Kubectl.Command("get", "namespace", tc.Kubectl.Namespace)
				if strings.Contains(err.Error(), "(NotFound): namespaces") {
					return err
				}
				return nil
			}
			Eventually(verifyNamespaceDeleted, 2*time.Minute, time.Second).ShouldNot(Succeed())
		})

		It("should run correctly in a cluster", func() {
			By("checking if the Operator project Pod is running")
			verifyControllerUp := func() error {
				By("getting the controller-manager pod name")
				podOutput, err := tc.Kubectl.Get(
					true,
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}")
				if err != nil {
					return fmt.Errorf("could not get pods: %v", err)
				}

				By("ensuring the created controller-manager Pod")
				podNames := kbtestutils.GetNonEmptyLines(podOutput)
				if len(podNames) != 1 {
					return fmt.Errorf("expecting 1 pod, have %d", len(podNames))
				}
				controllerPodName = podNames[0]
				if !strings.Contains(controllerPodName, "controller-manager") {
					return fmt.Errorf("expecting pod name %q to contain %q", controllerPodName, "controller-manager")
				}

				By("checking the controller-manager Pod is running")
				status, err := tc.Kubectl.Get(
					true,
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
				if err != nil {
					return fmt.Errorf("failed to get pod status for %q: %v", controllerPodName, err)
				}
				if status != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())

			By("ensuring the created ServiceMonitor for the manager")
			_, err := tc.Kubectl.Get(
				true,
				"ServiceMonitor",
				fmt.Sprintf("%s-controller-manager-metrics-monitor", tc.ProjectName))
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the created metrics Service for the manager")
			_, err = tc.Kubectl.Get(
				true,
				"Service",
				fmt.Sprintf("%s-controller-manager-metrics-service", tc.ProjectName))
			Expect(err).NotTo(HaveOccurred())

			By("creating an instance of CR")
			// currently controller-runtime doesn't provide a readiness probe, we retry a few times
			// we can change it to probe the readiness endpoint after CR supports it.
			sampleFile := filepath.Join("config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))
			Eventually(func() error {
				_, err = tc.Kubectl.Apply(true, "-f", sampleFile)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("granting permissions to access the metrics and read the token")
			_, err = tc.Kubectl.Command(
				"create",
				"clusterrolebinding", metricsClusterRoleBindingName,
				fmt.Sprintf("--clusterrole=%s-metrics-reader", tc.ProjectName),
				fmt.Sprintf("--serviceaccount=%s:default", tc.Kubectl.Namespace))
			Expect(err).NotTo(HaveOccurred())

			By("reading the token")
			b64Token, err := tc.Kubectl.Get(
				true,
				"secrets",
				"-o=jsonpath={.items[0].data.token}")
			Expect(err).NotTo(HaveOccurred())
			token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64Token))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(token)).To(BeNumerically(">", 0))

			By("creating a curl pod")
			// TODO: the flag --generator=run-pod/v1 is deprecated, however, shows that besides
			// it should not make any difference and work locally successfully when the flag is removed
			// the test will fail and the curl pod is not found when the flag is not used
			cmdOpts := []string{
				"run", "--generator=run-pod/v1", "curl", "--image=curlimages/curl:7.68.0", "--restart=OnFailure", "--",
				"curl", "-v", "-k", "-H", fmt.Sprintf(`Authorization: Bearer %s`, token),
				fmt.Sprintf("https://%s-controller-manager-metrics-service.%s.svc:8443/metrics", tc.ProjectName, tc.Kubectl.Namespace),
			}
			_, err = tc.Kubectl.CommandInNamespace(cmdOpts...)
			Expect(err).NotTo(HaveOccurred())

			By("validating that the curl pod is running as expected")
			verifyCurlUp := func() error {
				// Validate pod status
				status, err := tc.Kubectl.Get(
					true,
					"pods", "curl", "-o", "jsonpath={.status.phase}")
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				if status != "Completed" && status != "Succeeded" {
					return fmt.Errorf("curl pod in %s status", status)
				}
				return nil
			}
			Eventually(verifyCurlUp, 2*time.Minute, time.Second).Should(Succeed())

			By("validating that the metrics endpoint is serving as expected")
			var metricsOutput string
			getCurlLogs := func() string {
				metricsOutput, err = tc.Kubectl.Logs("curl")
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				return metricsOutput
			}
			Eventually(getCurlLogs, time.Minute, time.Second).Should(ContainSubstring("< HTTP/2 200"))

			// The controller updates memcacheds' status.nodes with a list of pods it is replicated across
			// on a successful reconcile.
			By("validating that the created resource object gets reconciled in the controller")
			status, err := tc.Kubectl.Get(true, "memcacheds", "memcached-sample", "-o", "jsonpath={.status.nodes}")
			Expect(err).NotTo(HaveOccurred())
			var nodes []string
			Expect(json.Unmarshal([]byte(status), &nodes)).To(Succeed())
			Expect(len(nodes)).To(BeNumerically(">", 0))
		})
	})
})
