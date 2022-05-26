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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/testutils/e2e/metrics"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
)

var _ = Describe("Running Helm projects", func() {
	var (
		controllerPodName, metricsClusterRoleBindingName string
	)

	Context("built with operator-sdk", func() {
		BeforeEach(func() {
			metricsClusterRoleBindingName = fmt.Sprintf("%s-metrics-reader", helmSample.Name())

			By("installing the CRDs on the cluster")
			Expect(operator.InstallCRDs(helmSampleValidKubeConfig)).To(Succeed())

			By("deploying project on the cluster")
			Expect(operator.DeployOperator(helmSampleValidKubeConfig, image)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up metrics")
			Expect(metrics.CleanUpMetrics(kctl, metricsClusterRoleBindingName)).To(Succeed())

			By("cleaning up created API objects during test process")
			// TODO(estroz): go/v2 does not have this target, so generalize once tests are refactored.
			Expect(operator.UndeployOperator(helmSampleValidKubeConfig)).To(Succeed())

			By("ensuring that the namespace was deleted")
			testutils.WrapWarnOutput(kctl.Wait(false, "namespace", "foo", "--for", "delete", "--timeout", "2m"))
		})

		It("should run correctly in a cluster", func() {
			By("checking if the Operator project Pod is running")
			verifyControllerUp := func() error {
				var err error
				controllerPodName, err = operator.EnsureOperatorRunning(kctl, 1, "controller-manager", "controller-manager")
				return err
			}
			Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())

			By("ensuring the created ServiceMonitor for the manager")
			_, err := kctl.Get(
				true,
				"ServiceMonitor",
				fmt.Sprintf("%s-controller-manager-metrics-monitor", helmSample.Name()))
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the created metrics Service for the manager")
			_, err = kctl.Get(
				true,
				"Service",
				fmt.Sprintf("%s-controller-manager-metrics-service", helmSample.Name()))
			Expect(err).NotTo(HaveOccurred())

			for _, gvk := range helmSample.GVKs() {
				sampleFile := filepath.Join("config", "samples",
					fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind)))

				By("updating replicaCount to 1 in the CR manifest")
				err = kbutil.ReplaceInFile(filepath.Join(helmSample.Dir(), sampleFile), "replicaCount: 3", "replicaCount: 1")
				Expect(err).NotTo(HaveOccurred())

				By("creating an instance of release(CR)")
				_, err = kctl.Apply(false, "-f", sampleFile)
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the CR gets reconciled and the release was Installed")
				managerContainerLogs := func() string {
					logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
					Expect(err).NotTo(HaveOccurred())
					return logOutput
				}
				Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Installed release"))

				By("getting the release name")
				releaseName, err := kctl.Get(
					false,
					gvk.Kind, "-o", "jsonpath={..status.deployedRelease.name}")
				Expect(err).NotTo(HaveOccurred())
				Expect(len(releaseName)).NotTo(BeIdenticalTo(0))

				By("checking the release(CR) statefulset status")
				verifyReleaseUp := func() string {
					output, err := kctl.Command(
						"rollout", "status", "statefulset", releaseName)
					Expect(err).NotTo(HaveOccurred())
					return output
				}
				Eventually(verifyReleaseUp, time.Minute, time.Second).Should(ContainSubstring("statefulset rolling update complete"))

				By("ensuring the created Service for the release(CR)")
				crServiceName, err := kctl.Get(
					false,
					"Service", "-l", fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
					"-o", "jsonpath={..metadata.name}")
				Expect(err).NotTo(HaveOccurred())
				Expect(len(crServiceName)).NotTo(BeIdenticalTo(0))

				By("scaling statefulset replicas to 2")
				_, err = kctl.Command(
					"scale", "statefulset", releaseName, "--replicas", "2")
				Expect(err).NotTo(HaveOccurred())

				By("verifying the statefulset automatically scales back down to 1")
				verifyRelease := func() error {
					replicas, err := kctl.Get(
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
				err = kbutil.ReplaceInFile(filepath.Join(helmSample.Dir(), sampleFile), "replicaCount: 1", "replicaCount: 2")
				Expect(err).NotTo(HaveOccurred())

				By("applying CR manifest with replicaCount: 2")
				_, err = kctl.Apply(false, "-f", sampleFile)
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the CR gets reconciled and the release was Upgraded")
				managerContainerLogsAfterUpdateCR := func() string {
					logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
					Expect(err).NotTo(HaveOccurred())
					return logOutput
				}
				Eventually(managerContainerLogsAfterUpdateCR, time.Minute, time.Second).Should(
					ContainSubstring("Upgraded release"))

				By("checking StatefulSet replicas spec is equals 2")
				verifyReleaseUpgrade := func() error {
					replicas, err := kctl.Get(
						false,
						"statefulset", releaseName, "-o", "jsonpath={..spec.replicas}")
					Expect(err).NotTo(HaveOccurred())
					if replicas != "2" {
						return fmt.Errorf("release(CR) statefulset with %s replicas", replicas)
					}
					return nil
				}
				Eventually(verifyReleaseUpgrade, time.Minute, time.Second).Should(Succeed())

				metricInfo := metrics.GetMetrics(helmSample, kctl, metricsClusterRoleBindingName)

				By("getting the CR namespace token")
				crNamespace, err := kctl.Get(
					false,
					gvk.Kind,
					fmt.Sprintf("%s-sample", strings.ToLower(gvk.Kind)),
					"-o=jsonpath={..metadata.namespace}")
				Expect(err).NotTo(HaveOccurred())
				Expect(crNamespace).NotTo(HaveLen(0))

				By("ensuring the operator metrics contains a `resource_created_at` metric for the CR")
				metricExportedCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
					"kind=\"%s\","+
					"name=\"%s-sample\","+
					"namespace=\"%s\","+
					"version=\"%s\"}",
					fmt.Sprintf("%s.%s", gvk.Group, helmSample.Domain()),
					gvk.Kind,
					strings.ToLower(gvk.Kind),
					crNamespace,
					gvk.Version)
				Expect(metricInfo).Should(ContainSubstring(metricExportedCR))

				By("annotate CR with uninstall-wait")
				cmdOpts := []string{
					"annotate", gvk.Kind, fmt.Sprintf("%s-sample", strings.ToLower(gvk.Kind)),
					"helm.sdk.operatorframework.io/uninstall-wait=true",
				}
				_, err = kctl.Command(cmdOpts...)
				Expect(err).NotTo(HaveOccurred())

				By("adding a finalizer to statefulset")
				cmdOpts = []string{
					"patch", "statefulset", releaseName, "-p",
					"{\"metadata\":{\"finalizers\":[\"helm.sdk.operatorframework.io/fake-finalizer\"]}}",
					"--type=merge",
				}
				_, err = kctl.Command(cmdOpts...)
				Expect(err).NotTo(HaveOccurred())

				By("deleting CR manifest")
				_, err = kctl.Delete(false, "-f", sampleFile, "--wait=false")
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the CR gets reconciled and uninstall-wait is enabled")
				managerContainerLogsAfterDeleteCR := func() string {
					logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
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
				_, err = kctl.Command(cmdOpts...)
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the CR gets reconciled and CR finalizer is removed")
				Eventually(managerContainerLogsAfterDeleteCR, 2*time.Minute, time.Second).Should(ContainSubstring("Removing finalizer"))

				By("ensuring the CR is deleted")
				verifyDeletedCR := func() error {
					_, err := kctl.Get(
						true,
						gvk.Kind, fmt.Sprintf("%s-sample", strings.ToLower(gvk.Kind)))
					if err == nil {
						return fmt.Errorf("the %s CR is not deleted", gvk.Kind)
					}
					return nil
				}
				Eventually(verifyDeletedCR, time.Minute, time.Second).Should(Succeed())

				By("creating an instance of release(CR) again after a delete with uninstall-wait")
				_, err = kctl.Apply(false, "-f", sampleFile)
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the CR gets reconciled and the release was Installed again")
				managerContainerLogs = func() string {
					logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
					Expect(err).NotTo(HaveOccurred())
					return logOutput
				}
				Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Installed release"))

				By("deleting CR manifest again without uninstall-wait")
				_, err = kctl.Delete(false, "-f", sampleFile)
				Expect(err).NotTo(HaveOccurred())

				By("ensuring the CR gets reconciled and the release was Uninstalled")
				managerContainerLogsAfterDeleteCR = func() string {
					logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
					Expect(err).NotTo(HaveOccurred())
					return logOutput
				}
				Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring("Uninstalled release"))
				Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring("Removing finalizer"))
			}

		})
	})
})
