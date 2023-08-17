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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
const (
	namespace                                      = "memcached-operator-system"
	memcachedDeploymentSizeUndesiredCountTotalName = "memcached_deployment_size_undesired_count_total"
	tokenRequestRawString                          = "{\"apiVersion\": \"authentication.k8s.io/v1\", \"kind\": \"TokenRequest\"}"
)

// tokenRequest is a trimmed down version of the authentication.k8s.io/v1/TokenRequest Type
// that we want to use for extracting the token.
type tokenRequest struct {
	Status struct {
		Token string "json:\"token\""
	} "json:\"status\""
}

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

	Context("Memcached Operator metrics", Ordered, func() {
		BeforeAll(func() {
			By("granting permissions to access the metrics")
			cmd := exec.Command("kubectl",
				"create", "clusterrolebinding", "metrics-memcached-operator",
				"--clusterrole=memcached-operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:memcached-operator-controller-manager", namespace))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			By("removing permissions to access the metrics")
			cmd := exec.Command("kubectl", "delete",
				"clusterrolebinding", "metrics-memcached-operator")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("MemcachedDeploymentSizeUndesiredCountTotal should be increased when scaling the Memcached deployment", func() {
			initialMetricValue := getMetricValue(memcachedDeploymentSizeUndesiredCountTotalName)

			numberOfScales := 5
			By(fmt.Sprintf("scaling memcached-samle deployment %d times", numberOfScales))
			scaleMemcachedSampleDeployment(numberOfScales)

			By(fmt.Sprintf("validating MemcachedDeploymentSizeUndesiredCountTotal has increased by %d", numberOfScales))
			finalMetricValue := getMetricValue(memcachedDeploymentSizeUndesiredCountTotalName)
			Expect(finalMetricValue).Should(BeNumerically(">=", initialMetricValue+numberOfScales))
		})
	})
})

// getMetricValue will reach the Memcached operator metrics endpoint, validate the metric and extract its value
func getMetricValue(metricName string) int {
	// reach the metrics endpoint and validate the metric exists
	metricsEndpoint := curlMetrics()
	ExpectWithOffset(1, metricsEndpoint).Should(ContainSubstring(metricName))

	// extract the metric value
	metricValue, err := strconv.Atoi(parseMetricValue(metricsEndpoint, metricName))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return metricValue
}

// curlMetrics curl's the /metrics endpoint, returning all logs once a 200 status is returned.
func curlMetrics() string {
	By("reading the metrics token")
	// Filter token query by service account in case more than one exists in a namespace.
	token, err := serviceAccountToken()
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	ExpectWithOffset(2, len(token)).To(BeNumerically(">", 0))

	By("creating a curl pod")
	cmd := exec.Command("kubectl", "run", "curl", "--image=curlimages/curl:7.68.0",
		"--restart=OnFailure", "-n", "default", "--", "curl", "-v", "-k", "-H",
		fmt.Sprintf("Authorization: Bearer %s", strings.TrimSpace(token)),
		fmt.Sprintf("https://memcached-operator-controller-manager-metrics-service.%s.svc:8443/metrics", namespace))
	_, err = utils.Run(cmd)
	ExpectWithOffset(2, err).NotTo(HaveOccurred())

	By("validating that the curl pod is running as expected")
	verifyCurlUp := func() error {
		// Validate pod status
		cmd := exec.Command("kubectl", "get", "pods", "curl",
			"-o", "jsonpath={.status.phase}", "-n", "default")
		statusOutput, err := utils.Run(cmd)
		status := string(statusOutput)
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		if status != "Completed" && status != "Succeeded" {
			return fmt.Errorf("curl pod in %s status", status)
		}
		return nil
	}
	EventuallyWithOffset(2, verifyCurlUp, 240*time.Second, time.Second).Should(Succeed())

	By("validating that the metrics endpoint is serving as expected")
	var metricsEndpoint string
	getCurlLogs := func() string {
		cmd = exec.Command("kubectl", "logs", "curl", "-n", "default")
		metricsEndpointOutput, err := utils.Run(cmd)
		ExpectWithOffset(3, err).NotTo(HaveOccurred())
		metricsEndpoint = string(metricsEndpointOutput)
		return metricsEndpoint
	}
	EventuallyWithOffset(2, getCurlLogs, 10*time.Second, time.Second).Should(ContainSubstring("< HTTP/2 200"))

	By("cleaning up the curl pod")
	cmd = exec.Command("kubectl", "delete",
		"pods/curl", "-n", "default")
	_, err = utils.Run(cmd)
	ExpectWithOffset(3, err).NotTo(HaveOccurred())

	return metricsEndpoint
}

// serviceAccountToken provides a helper function that can provide you with a service account
// token that you can use to interact with the service. This function leverages the k8s'
// TokenRequest API in raw format in order to make it generic for all version of the k8s that
// is currently being supported in kubebuilder test infra.
// TokenRequest API returns the token in raw JWT format itself. There is no conversion required.
func serviceAccountToken() (out string, err error) {
	By("Creating the ServiceAccount token")
	secretName := "memcached-operator-controller-manager-token-request"
	projectDir, _ := utils.GetProjectDir()
	tokenRequestFile := filepath.Join(projectDir, "/test/e2e/", secretName)
	err = os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o755))
	if err != nil {
		return out, err
	}
	var rawJson string
	Eventually(func() error {
		// Output of this is already a valid JWT token. No need to covert this from base64 to string format
		cmd := exec.Command("kubectl", "create", "--raw",
			fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/memcached-operator-controller-manager/token", namespace),
			"-f", tokenRequestFile,
		)
		rawJsonOutput, err := utils.Run(cmd)
		rawJson = string(rawJsonOutput)
		if err != nil {
			return err
		}
		var token tokenRequest
		err = json.Unmarshal([]byte(rawJson), &token)
		if err != nil {
			return err
		}
		out = token.Status.Token
		return nil
	}, time.Minute, time.Second).Should(Succeed())

	return out, err
}

// parseMetricValue will parse the metric value from the metrics endpoint
func parseMetricValue(metricsEndpoint string, metricName string) string {
	r := strings.NewReader(metricsEndpoint)
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		metricLine := scan.Text()
		if strings.HasPrefix(metricLine, metricName) {
			split := strings.Split(metricLine, " ")
			return split[1]
		}
	}
	return ""
}

// scaleMemcachedSampleDeployment will scale memcached-sample deployment 'numberOfScales' times
func scaleMemcachedSampleDeployment(numberOfScales int) {
	for i := 1; i <= numberOfScales; i++ {
		cmd := exec.Command("kubectl", "scale", "--replicas=3",
			"deployment", "memcached-sample", "-n", namespace)
		_, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)
	}
}

const monitoringImportFragment = "\"github.com/example/memcached-operator/monitoring\""

const incMemcachedDeploymentSizeUndesiredCountTotalFragment = "monitoring.MemcachedDeploymentSizeUndesiredCountTotal.Inc()"

const registerMetricsFragment = "monitoring.RegisterMetrics()"
