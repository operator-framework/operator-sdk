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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kbtutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/test/common"
)

var _ = Describe("Running ansible projects", func() {

	var (
		controllerPodName, memcachedDeploymentName, metricsClusterRoleBindingName string
		fooSampleFile, memfinSampleFile, memcachedSampleFile                      string
	)

	Context("built with operator-sdk", func() {
		BeforeEach(func() {
			metricsClusterRoleBindingName = fmt.Sprintf("%s-metrics-reader", tc.ProjectName)
			samplesDir := filepath.Join(tc.Dir, "config", "samples")
			fooSampleFile = filepath.Join(samplesDir, fmt.Sprintf("%s_%s_foo.yaml", tc.Group, tc.Version))
			memfinSampleFile = filepath.Join(samplesDir, fmt.Sprintf("%s_%s_memfin.yaml", tc.Group, tc.Version))
			memcachedSampleFile = filepath.Join(samplesDir,
				fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))

			By("deploying project on the cluster")
			Expect(tc.Make("deploy", "IMG="+tc.ImageName)).To(Succeed())
		})

		AfterEach(func() {
			By("deleting curl pod")
			testutils.WrapWarnOutput(tc.Kubectl.Delete(false, "pod", "curl"))

			By("deleting test CR instances")
			for _, sample := range []string{memcachedSampleFile, fooSampleFile, memfinSampleFile} {
				testutils.WrapWarnOutput(tc.Kubectl.Delete(false, "-f", sample))
			}

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
				podNames := kbtutil.GetNonEmptyLines(podOutput)
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

			By("create custom resource (Memcached CR)")
			_, err = tc.Kubectl.Apply(false, "-f", memcachedSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("create custom resource (Foo CR)")
			_, err = tc.Kubectl.Apply(false, "-f", fooSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("create custom resource (Memfin CR)")
			_, err = tc.Kubectl.Apply(false, "-f", memfinSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled")
			managerContainerLogs := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
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
				eventsOutput, err := tc.Kubectl.Get(
					true,
					"events", "--field-selector", fmt.Sprintf("involvedObject.name=%s", controllerPodName))
				Expect(err).NotTo(HaveOccurred())
				return eventsOutput
			}
			Eventually(verifyControllerProbe, time.Minute, time.Second).ShouldNot(ContainSubstring("Killing"))

			By("getting memcached deploy by labels")
			getMemcachedDeploymentName := func() string {
				memcachedDeploymentName, err = tc.Kubectl.Get(
					false, "deployment",
					"-l", "app=memcached", "-o", "jsonpath={..metadata.name}")
				Expect(err).NotTo(HaveOccurred())
				return memcachedDeploymentName
			}
			Eventually(getMemcachedDeploymentName, 2*time.Minute, time.Second).ShouldNot(BeEmpty())

			By("checking the Memcached CR deployment status")
			verifyCRUp := func() string {
				output, err := tc.Kubectl.Command(
					"rollout", "status", "deployment", memcachedDeploymentName)
				Expect(err).NotTo(HaveOccurred())
				return output
			}
			Eventually(verifyCRUp, time.Minute, time.Second).Should(ContainSubstring("successfully rolled out"))

			By("ensuring the created Service for the Memcached CR")
			crServiceName, err := tc.Kubectl.Get(
				false,
				"Service", "-l", "app=memcached")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(crServiceName)).NotTo(BeIdenticalTo(0))

			By("Verifying that a config map owned by the CR has been created")
			verifyConfigMap := func() error {
				_, err = tc.Kubectl.Get(
					false,
					"configmap", "test-blacklist-watches")
				return err
			}
			Eventually(verifyConfigMap, time.Minute*2, time.Second).Should(Succeed())

			By("Ensuring that config map requests skip the cache.")
			checkSkipCache := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(checkSkipCache, time.Minute, time.Second).Should(ContainSubstring("\"Skipping cache lookup" +
				"\",\"resource\":{\"IsResourceRequest\":true," +
				"\"Path\":\"/api/v1/namespaces/default/configmaps/test-blacklist-watches\""))

			By("scaling deployment replicas to 2")
			_, err = tc.Kubectl.Command(
				"scale", "deployment", memcachedDeploymentName, "--replicas", "2")
			Expect(err).NotTo(HaveOccurred())

			By("verifying the deployment automatically scales back down to 1")
			verifyMemcachedScalesBack := func() error {
				replicas, err := tc.Kubectl.Get(
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
			err = kbtutil.ReplaceInFile(memcachedSampleFile, "size: 1", "size: 2")
			Expect(err).NotTo(HaveOccurred())

			By("applying CR manifest with size: 2")
			_, err = tc.Kubectl.Apply(false, "-f", memcachedSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled after patching it")
			managerContainerLogsAfterUpdateCR := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterUpdateCR, time.Minute, time.Second).Should(
				ContainSubstring("Ansible-runner exited successfully"))
			Eventually(managerContainerLogs, time.Minute, time.Second).ShouldNot(ContainSubstring("failed=1"))

			By("checking Deployment replicas spec is equals 2")
			verifyMemcachedPatch := func() error {
				replicas, err := tc.Kubectl.Get(
					false,
					"deployment", memcachedDeploymentName, "-o", "jsonpath={..spec.replicas}")
				Expect(err).NotTo(HaveOccurred())
				if replicas != "2" {
					return fmt.Errorf("memcached(CR) deployment with %s replicas", replicas)
				}
				return nil
			}
			Eventually(verifyMemcachedPatch, time.Minute, time.Second).Should(Succeed())

			// As of Kubernetes 1.24 a ServiceAccount no longer has a ServiceAccount token secret autogenerated. We have to create it manually here
			By("Creating the ServiceAccount token")
			secretFile, err := common.GetSASecret(tc.Kubectl.ServiceAccount, tc.Dir)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				_, err = tc.Kubectl.Apply(true, "-f", secretFile)
				return err
			}, time.Minute, time.Second).Should(Succeed())
			By("annotating the CR")
			_, err = tc.Kubectl.Command(
				"annotate", "foo", "foo-sample", "test-annotation='step2'")
			Expect(err).NotTo(HaveOccurred())

			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring(
				"Ansible-runner exited successfully"))
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring(
				"test-annotation found : 'step2'"))
			Eventually(managerContainerLogs, time.Minute, time.Second).ShouldNot(ContainSubstring("failed=1"))
			Eventually(managerContainerLogs, time.Minute, time.Second).ShouldNot(ContainSubstring("[Gathering Facts]"))

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
				if err != nil {
					return err
				}
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

			By("ensuring the operator metrics contains a `resource_created_at` metric for the Memcached CR")
			metricExportedMemcachedCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", tc.Group, tc.Domain),
				tc.Kind,
				strings.ToLower(tc.Kind),
				crNamespace,
				tc.Version)
			Eventually(getCurlLogs, time.Minute, time.Second).Should(ContainSubstring(metricExportedMemcachedCR))

			By("ensuring the operator metrics contains a `resource_created_at` metric for the Foo CR")
			metricExportedFooCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", tc.Group, tc.Domain),
				"Foo",
				strings.ToLower("Foo"),
				crNamespace,
				tc.Version)
			Eventually(getCurlLogs, time.Minute, time.Second).Should(ContainSubstring(metricExportedFooCR))

			By("ensuring the operator metrics contains a `resource_created_at` metric for the Memfin CR")
			metricExportedMemfinCR := fmt.Sprintf("resource_created_at_seconds{group=\"%s\","+
				"kind=\"%s\","+
				"name=\"%s-sample\","+
				"namespace=\"%s\","+
				"version=\"%s\"}",
				fmt.Sprintf("%s.%s", tc.Group, tc.Domain),
				"Memfin",
				strings.ToLower("Memfin"),
				crNamespace,
				tc.Version)
			Eventually(getCurlLogs, time.Minute, time.Second).Should(ContainSubstring(metricExportedMemfinCR))

			By("creating a configmap that the finalizer should remove")
			_, err = tc.Kubectl.Command("create", "configmap", "deleteme")
			Expect(err).NotTo(HaveOccurred())

			By("deleting Memcached CR manifest")
			_, err = tc.Kubectl.Delete(false, "-f", memcachedSampleFile)
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the CR gets reconciled successfully")
			managerContainerLogsAfterDeleteCR := func() string {
				logOutput, err := tc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogsAfterDeleteCR, time.Minute, time.Second).Should(ContainSubstring(
				"Ansible-runner exited successfully"))
			Eventually(managerContainerLogsAfterDeleteCR).ShouldNot(ContainSubstring("error"))

			By("ensuring that Memchaced Deployment was removed")
			getMemcachedDeployment := func() error {
				_, err := tc.Kubectl.Get(
					false, "deployment",
					memcachedDeploymentName)
				return err
			}
			Eventually(getMemcachedDeployment, time.Minute*2, time.Second).ShouldNot(Succeed())
		})
	})
})
