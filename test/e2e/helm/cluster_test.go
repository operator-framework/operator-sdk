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

package e2e_helm_test

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/test/common"
)

var _ = Describe("Running Helm projects", func() {
	var (
		controllerPodName, metricsClusterRoleBindingName string
		memcachedSampleFile                              string
	)

	Context("built with operator-sdk", func() {
		BeforeEach(func() {
			metricsClusterRoleBindingName = fmt.Sprintf("%s-metrics-reader", tc.ProjectName)
			memcachedSampleFile = filepath.Join(tc.Dir, "config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))

			By("deploying project on the cluster")
			Expect(tc.Make("deploy", "IMG="+tc.ImageName)).To(Succeed())
		})

		AfterEach(func() {
			By("deleting curl pod")
			testutils.WrapWarnOutput(tc.Kubectl.Delete(false, "pod", "curl"))

			By("deleting test CR instances")
			testutils.WrapWarnOutput(tc.Kubectl.Delete(false, "-f", memcachedSampleFile))

			By("cleaning up permissions")
			testutils.WrapWarnOutput(tc.Kubectl.Command("delete", "clusterrolebinding", metricsClusterRoleBindingName))

			By("undeploy project")
			testutils.WrapWarn(tc.Make("undeploy"))

			By("ensuring that the namespace was deleted")
			testutils.WrapWarnOutput(tc.Kubectl.Wait(false, "namespace", "foo", "--for", "delete", "--timeout", "2m"))
		})

		It("should run correctly in a cluster", func() {
			By("checking if the Operator project Pod is running")
			verifyControllerUp := func() error {
				// Get the controller-manager pod name
				podOutput, err := tc.Kubectl.Get(
					true,
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}")
				if err != nil {
					return fmt.Errorf("could not get pods: %v", err)
				}
				podNames := kbutil.GetNonEmptyLines(podOutput)
				if len(podNames) != 1 {
					return fmt.Errorf("expecting 1 pod, have %d", len(podNames))
				}
				controllerPodName = podNames[0]
				if !strings.Contains(controllerPodName, "controller-manager") {
					return fmt.Errorf("expecting pod name %q to contain %q", controllerPodName, "controller-manager")
				}

				// Ensure the controller-manager Pod is running.
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

			sampleFile := filepath.Join("config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))

			By("updating replicaCount to 1 in the CR manifest")
			err = kbutil.ReplaceInFile(filepath.Join(tc.Dir, sampleFile), "replicaCount: 3", "replicaCount: 1")
			Expect(err).NotTo(HaveOccurred())

			By("creating an instance of release(CR)")
			_, err = tc.Kubectl.Apply(false, "-f", sampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled and the release was Installed")
			managerContainerLogs := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Installed release"))

			By("getting the release name")
			releaseName, err := tc.Kubectl.Get(
				false,
				tc.Kind, "-o", "jsonpath={..status.deployedRelease.name}")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(releaseName)).NotTo(BeIdenticalTo(0))

			By("checking the release(CR) statefulset status")
			verifyReleaseUp := func() string {
				output, err := tc.Kubectl.Command(
					"rollout", "status", "statefulset", releaseName)
				Expect(err).NotTo(HaveOccurred())
				return output
			}
			Eventually(verifyReleaseUp, time.Minute, time.Second).Should(ContainSubstring("statefulset rolling update complete"))

			By("ensuring the created Service for the release(CR)")
			crServiceName, err := tc.Kubectl.Get(
				false,
				"Service", "-l", fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
				"-o", "jsonpath={..metadata.name}")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(crServiceName)).NotTo(BeIdenticalTo(0))

			By("scaling statefulset replicas to 2")
			_, err = tc.Kubectl.Command(
				"scale", "statefulset", releaseName, "--replicas", "2")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the statefulset automatically scales back down to 1")
			verifyRelease := func() error {
				replicas, err := tc.Kubectl.Get(
					false,
					"statefulset", releaseName, "-o", "jsonpath={..spec.replicas}")
				Expect(err).NotTo(HaveOccurred())
				if replicas != "1" {
					return fmt.Errorf("release(CR) statefulset with %s replicas", replicas)
				}
				return nil
			}
			Eventually(verifyRelease, time.Minute, time.Second).Should(Succeed())

			By("updating replicaCount to 2 in the CR manifest")
			err = kbutil.ReplaceInFile(filepath.Join(tc.Dir, sampleFile), "replicaCount: 1", "replicaCount: 2")
			Expect(err).NotTo(HaveOccurred())

			By("applying CR manifest with replicaCount: 2")
			_, err = tc.Kubectl.Apply(false, "-f", sampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled and the release was Upgraded")
			managerContainerLogsAfterUpdateCR := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterUpdateCR, time.Minute, time.Second).Should(
				ContainSubstring("Upgraded release"))

			By("checking StatefulSet replicas spec is equals 2")
			verifyReleaseUpgrade := func() error {
				replicas, err := tc.Kubectl.Get(
					false,
					"statefulset", releaseName, "-o", "jsonpath={..spec.replicas}")
				Expect(err).NotTo(HaveOccurred())
				if replicas != "2" {
					return fmt.Errorf("release(CR) statefulset with %s replicas", replicas)
				}
				return nil
			}
			Eventually(verifyReleaseUpgrade, time.Minute, time.Second).Should(Succeed())

			// As of Kubernetes 1.24 a ServiceAccount no longer has a ServiceAccount token secret autogenerated. We have to create it manually here
			By("Creating the ServiceAccount token")
			secretFile, err := common.GetSASecret(tc.Kubectl.ServiceAccount, tc.Dir)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				_, err = tc.Kubectl.Apply(true, "-f", secretFile)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("granting permissions to access the metrics and read the token")
			_, err = tc.Kubectl.Command("create", "clusterrolebinding", metricsClusterRoleBindingName,
				fmt.Sprintf("--clusterrole=%s-metrics-reader", tc.ProjectName),
				fmt.Sprintf("--serviceaccount=%s:%s", tc.Kubectl.Namespace, tc.Kubectl.ServiceAccount))
			Expect(err).NotTo(HaveOccurred())

			By("reading the metrics token")
			// Filter token query by service account in case more than one exists in a namespace.
			query := fmt.Sprintf(`{.items[?(@.metadata.annotations.kubernetes\.io/service-account\.name=="%s")].data.token}`,
				tc.Kubectl.ServiceAccount,
			)
			b64Token, err := tc.Kubectl.Get(true, "secrets", "-o=jsonpath="+query)
			Expect(err).NotTo(HaveOccurred())
			token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64Token))
			Expect(err).NotTo(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			By("creating a curl pod")
			cmdOpts := []string{
				"run", "curl", "--image=curlimages/curl:7.68.0", "--restart=OnFailure", "--",
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

			By("checking metrics endpoint serving as expected")
			getCurlLogs := func() string {
				logOutput, err := tc.Kubectl.Logs("curl")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(getCurlLogs, time.Minute, time.Second).Should(ContainSubstring("< HTTP/2 200"))

			By("getting the CR namespace token")
			crNamespace, err := tc.Kubectl.Get(
				false,
				tc.Kind,
				fmt.Sprintf("%s-sample", strings.ToLower(tc.Kind)),
				"-o=jsonpath={..metadata.namespace}")
			Expect(err).NotTo(HaveOccurred())
			Expect(crNamespace).NotTo(BeEmpty())

			By("ensuring the operator metrics contains a `resource_created_at` metric for the CR")
			metricExportedCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", tc.Group, tc.Domain),
				tc.Kind,
				strings.ToLower(tc.Kind),
				crNamespace,
				tc.Version)
			Eventually(getCurlLogs, time.Minute, time.Second).Should(ContainSubstring(metricExportedCR))

			By("annotate CR with uninstall-wait")
			cmdOpts = []string{
				"annotate", tc.Kind, fmt.Sprintf("%s-sample", strings.ToLower(tc.Kind)),
				"helm.sdk.operatorframework.io/uninstall-wait=true",
			}
			_, err = tc.Kubectl.Command(cmdOpts...)
			Expect(err).NotTo(HaveOccurred())

			By("adding a finalizer to statefulset")
			cmdOpts = []string{
				"patch", "statefulset", releaseName, "-p",
				"{\"metadata\":{\"finalizers\":[\"helm.sdk.operatorframework.io/fake-finalizer\"]}}",
				"--type=merge",
			}
			_, err = tc.Kubectl.Command(cmdOpts...)
			Expect(err).NotTo(HaveOccurred())

			By("deleting CR manifest")
			_, err = tc.Kubectl.Delete(false, "-f", sampleFile, "--wait=false")
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled and uninstall-wait is enabled")
			managerContainerLogsAfterDeleteCR := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring("Uninstall wait"))
			Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring("Waiting until all resources are deleted"))

			By("removing the finalizer from statefulset")
			cmdOpts = []string{
				"patch", "statefulset", releaseName, "-p",
				"{\"metadata\":{\"finalizers\":[]}}",
				"--type=merge",
			}
			_, err = tc.Kubectl.Command(cmdOpts...)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled and CR finalizer is removed")
			Eventually(managerContainerLogsAfterDeleteCR, 2*time.Minute, time.Second).Should(ContainSubstring("Removing finalizer"))

			By("ensuring the CR is deleted")
			verifyDeletedCR := func() error {
				_, err := tc.Kubectl.Get(
					true,
					tc.Kind, fmt.Sprintf("%s-sample", strings.ToLower(tc.Kind)))
				if err == nil {
					return fmt.Errorf("the %s CR is not deleted", tc.Kind)
				}
				return nil
			}
			Eventually(verifyDeletedCR, time.Minute, time.Second).Should(Succeed())

			By("creating an instance of release(CR) again after a delete with uninstall-wait")
			_, err = tc.Kubectl.Apply(false, "-f", sampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled and the release was Installed again")
			managerContainerLogs = func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Installed release"))

			By("deleting CR manifest again without uninstall-wait")
			_, err = tc.Kubectl.Delete(false, "-f", sampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled and the release was Uninstalled")
			managerContainerLogsAfterDeleteCR = func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring("Uninstalled release"))
			Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring("Removing finalizer"))
		})
	})
})
