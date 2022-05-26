// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writifng, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e_ansible_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kbtutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/testutils/e2e/metrics"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
)

var _ = Describe("Running ansible projects", func() {

	var (
		controllerPodName, memcachedDeploymentName, metricsClusterRoleBindingName string
		fooSampleFile, memfinSampleFile, memcachedSampleFile                      string
		gvk                                                                       schema.GroupVersionKind
	)

	Context("built with operator-sdk", func() {
		BeforeEach(func() {
			gvk = ansibleSample.GVKs()[0]
			metricsClusterRoleBindingName = fmt.Sprintf("%s-metrics-reader", ansibleSample.Name())
			samplesDir := filepath.Join("config", "samples")
			fooSampleFile = filepath.Join(samplesDir, fmt.Sprintf("%s_%s_foo.yaml", gvk.Group, gvk.Version))
			memfinSampleFile = filepath.Join(samplesDir, fmt.Sprintf("%s_%s_memfin.yaml", gvk.Group, gvk.Version))
			memcachedSampleFile = filepath.Join(samplesDir,
				fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind)))

			By("installing the CRDs on the cluster")
			Expect(operator.InstallCRDs(ansibleSample)).To(Succeed())

			By("deploying project on the cluster")
			Expect(operator.DeployOperator(ansibleSample, image)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up metrics")
			Expect(metrics.CleanUpMetrics(kctl, metricsClusterRoleBindingName)).To(Succeed())

			By("cleaning up created API objects during test process")
			Expect(operator.UndeployOperator(ansibleSample)).To(Succeed())

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
				fmt.Sprintf("%s-controller-manager-metrics-monitor", ansibleSample.Name()))
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the created metrics Service for the manager")
			_, err = kctl.Get(
				true,
				"Service",
				fmt.Sprintf("%s-controller-manager-metrics-service", ansibleSample.Name()))
			Expect(err).NotTo(HaveOccurred())

			// TODO(everettraven): this could be simplifed once all the GVKs are made to be part of the sample implementation
			// -----------------------------------------------
			By("create custom resource (Memcached CR)")
			_, err = kctl.Apply(false, "-f", memcachedSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("create custom resource (Foo CR)")
			_, err = kctl.Apply(false, "-f", fooSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("create custom resource (Memfin CR)")
			_, err = kctl.Apply(false, "-f", memfinSampleFile)
			Expect(err).NotTo(HaveOccurred())
			// -----------------------------------------------

			By("ensuring the CR gets reconciled")
			managerContainerLogs := func() string {
				logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring(
				"Ansible-runner exited successfully"))
			Eventually(managerContainerLogs, time.Minute, time.Second).ShouldNot(ContainSubstring("failed=1"))
			Eventually(managerContainerLogs, time.Minute, time.Second).ShouldNot(ContainSubstring("[Gathering Facts]"))

			By("ensuring no liveness probe fail events")
			verifyControllerProbe := func() string {
				By("getting the controller-manager events")
				eventsOutput, err := kctl.Get(
					true,
					"events", "--field-selector", fmt.Sprintf("involvedObject.name=%s", controllerPodName))
				Expect(err).NotTo(HaveOccurred())
				return eventsOutput
			}
			Eventually(verifyControllerProbe, time.Minute, time.Second).ShouldNot(ContainSubstring("Killing"))

			By("getting memcached deploy by labels")
			getMemcachedDeploymentName := func() string {
				memcachedDeploymentName, err = kctl.Get(
					false, "deployment",
					"-l", "app=memcached", "-o", "jsonpath={..metadata.name}")
				Expect(err).NotTo(HaveOccurred())
				return memcachedDeploymentName
			}
			Eventually(getMemcachedDeploymentName, 2*time.Minute, time.Second).ShouldNot(BeEmpty())

			By("checking the Memcached CR deployment status")
			verifyCRUp := func() string {
				output, err := kctl.Command(
					"rollout", "status", "deployment", memcachedDeploymentName)
				Expect(err).NotTo(HaveOccurred())
				return output
			}
			Eventually(verifyCRUp, time.Minute, time.Second).Should(ContainSubstring("successfully rolled out"))

			By("ensuring the created Service for the Memcached CR")
			crServiceName, err := kctl.Get(
				false,
				"Service", "-l", "app=memcached")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(crServiceName)).NotTo(BeIdenticalTo(0))

			By("Verifying that a config map owned by the CR has been created")
			verifyConfigMap := func() error {
				_, err = kctl.Get(
					false,
					"configmap", "test-blacklist-watches")
				return err
			}
			Eventually(verifyConfigMap, time.Minute*2, time.Second).Should(Succeed())

			By("Ensuring that config map requests skip the cache.")
			checkSkipCache := func() string {
				logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(checkSkipCache, time.Minute, time.Second).Should(ContainSubstring("\"Skipping cache lookup" +
				"\",\"resource\":{\"IsResourceRequest\":true," +
				"\"Path\":\"/api/v1/namespaces/default/configmaps/test-blacklist-watches\""))

			By("scaling deployment replicas to 2")
			_, err = kctl.Command(
				"scale", "deployment", memcachedDeploymentName, "--replicas", "2")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the deployment automatically scales back down to 1")
			verifyMemcachedScalesBack := func() error {
				replicas, err := kctl.Get(
					false,
					"deployment", memcachedDeploymentName, "-o", "jsonpath={..spec.replicas}")
				Expect(err).NotTo(HaveOccurred())
				if replicas != "1" {
					return fmt.Errorf("memcached(CR) deployment with %s replicas", replicas)
				}
				return nil
			}
			Eventually(verifyMemcachedScalesBack, time.Minute, time.Second).Should(Succeed())

			By("updating size to 2 in the CR manifest")
			err = kbtutil.ReplaceInFile(filepath.Join(ansibleSample.Dir(), memcachedSampleFile), "size: 1", "size: 2")
			Expect(err).NotTo(HaveOccurred())

			By("applying CR manifest with size: 2")
			_, err = kctl.Apply(false, "-f", memcachedSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled after patching it")
			managerContainerLogsAfterUpdateCR := func() string {
				logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterUpdateCR, time.Minute, time.Second).Should(
				ContainSubstring("Ansible-runner exited successfully"))
			Eventually(managerContainerLogs, time.Minute, time.Second).ShouldNot(ContainSubstring("failed=1"))

			By("checking Deployment replicas spec is equals 2")
			verifyMemcachedPatch := func() error {
				replicas, err := kctl.Get(
					false,
					"deployment", memcachedDeploymentName, "-o", "jsonpath={..spec.replicas}")
				Expect(err).NotTo(HaveOccurred())
				if replicas != "2" {
					return fmt.Errorf("memcached(CR) deployment with %s replicas", replicas)
				}
				return nil
			}
			Eventually(verifyMemcachedPatch, time.Minute, time.Second).Should(Succeed())

			metricInfo := metrics.GetMetrics(ansibleSample, kctl, metricsClusterRoleBindingName)

			By("getting the CR namespace token")
			crNamespace, err := kctl.Get(
				false,
				gvk.Kind,
				fmt.Sprintf("%s-sample", strings.ToLower(gvk.Kind)),
				"-o=jsonpath={..metadata.namespace}")
			Expect(err).NotTo(HaveOccurred())
			Expect(crNamespace).NotTo(HaveLen(0))

			By("ensuring the operator metrics contains a `resource_created_at` metric for the Memcached CR")
			metricExportedMemcachedCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", gvk.Group, ansibleSample.Domain()),
				gvk.Kind,
				strings.ToLower(gvk.Kind),
				crNamespace,
				gvk.Version)
			Expect(metricInfo).Should(ContainSubstring(metricExportedMemcachedCR))

			By("ensuring the operator metrics contains a `resource_created_at` metric for the Foo CR")
			metricExportedFooCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", gvk.Group, ansibleSample.Domain()),
				"Foo",
				strings.ToLower("Foo"),
				crNamespace,
				gvk.Version)
			Expect(metricInfo).Should(ContainSubstring(metricExportedFooCR))

			By("ensuring the operator metrics contains a `resource_created_at` metric for the Memfin CR")
			metricExportedMemfinCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", gvk.Group, ansibleSample.Domain()),
				"Memfin",
				strings.ToLower("Memfin"),
				crNamespace,
				gvk.Version)
			Expect(metricInfo).Should(ContainSubstring(metricExportedMemfinCR))

			By("creating a configmap that the finalizer should remove")
			_, err = kctl.Command("create", "configmap", "deleteme")
			Expect(err).NotTo(HaveOccurred())

			By("deleting Memcached CR manifest")
			_, err = kctl.Delete(false, "-f", memcachedSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled successfully")
			managerContainerLogsAfterDeleteCR := func() string {
				logOutput, err := kctl.Logs(true, controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring(
				"Ansible-runner exited successfully"))
			Eventually(managerContainerLogsAfterDeleteCR).ShouldNot(ContainSubstring("error"))

			By("ensuring that Memchaced Deployment was removed")
			getMemcachedDeployment := func() error {
				_, err := kctl.Get(
					false, "deployment",
					memcachedDeploymentName)
				return err
			}
			Eventually(getMemcachedDeployment, time.Minute*2, time.Second).ShouldNot(Succeed())
		})
	})
})
