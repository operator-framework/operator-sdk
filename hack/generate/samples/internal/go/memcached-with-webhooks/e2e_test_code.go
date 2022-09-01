package v3

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	log "github.com/sirupsen/logrus"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

// implementingE2ETests will add e2e test for the sample
// so that users are able to know how the can create their own e2e tests
func (mh *Memcached) implementingE2ETests() {
	log.Infof("implementing example e2e tests")

	// testdDir is testdata/go/v3/memcached-operator/test
	testDir := filepath.Join(mh.ctx.Dir, "test")
	// testE2eDir is testdata/go/v3/memcached-operator/test/e2e
	testE2eDir := filepath.Join(testDir, "e2e")
	// testE2eDir is testdata/go/v3/memcached-operator/test/utils
	testUtilsDir := filepath.Join(testDir, "utils")

	// Following we will create the directories
	// Create the golang files with a string replace inside onlu
	// Then, replace the string replace by the template contents
	mh.createDirs(testDir, testE2eDir, testUtilsDir)
	mh.createGoFiles(testE2eDir, testUtilsDir)
	mh.addContent(testE2eDir, testUtilsDir)

	// Add a target to run the tests into the Makefile
	mh.addTestE2eMaekefileTarget()
}

func (mh *Memcached) addTestE2eMaekefileTarget() {
	err := kbutil.ReplaceInFile(filepath.Join(mh.ctx.Dir, "Makefile"),
		`KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out`,
		`KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)"  go test $(go list ./... | grep -v /test/) -coverprofile cover.out`,
	)
	pkg.CheckError("replacing test target", err)

	err = kbutil.InsertCode(filepath.Join(mh.ctx.Dir, "Makefile"),
		`.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)"  go test $(go list ./... | grep -v /test/) -coverprofile cover.out
`,
		targetTemplate,
	)
	pkg.CheckError("insert e2e target", err)
}

func (mh *Memcached) addContent(testE2eDir string, testUtilsDir string) {
	err := kbutil.ReplaceInFile(filepath.Join(testE2eDir, "e2e_suite_test.go"),
		"replace",
		e2eSuiteTemplate,
	)
	pkg.CheckError("replacing e2eSuiteTemplate", err)

	err = kbutil.ReplaceInFile(filepath.Join(testE2eDir, "e2e_test.go"),
		"replace",
		e2eTemplate,
	)
	pkg.CheckError("replacing e2eTemplate tests", err)

	err = kbutil.ReplaceInFile(filepath.Join(testUtilsDir, "utils.go"),
		"replace",
		utilsTemplate,
	)
	pkg.CheckError("replacing utils", err)
}

func (mh *Memcached) createGoFiles(testE2eDir string, testUtilsDir string) {
	err := ioutil.WriteFile(filepath.Join(testE2eDir, "e2e_suite_test.go"), []byte("replace"), 0644)
	pkg.CheckError("error to create file to add e2e_suite_test.go", err)

	err = ioutil.WriteFile(filepath.Join(testE2eDir, "e2e_test.go"), []byte("replace"), 0644)
	pkg.CheckError("error to create file to add e2e_test.go", err)

	err = ioutil.WriteFile(filepath.Join(testUtilsDir, "utils.go"), []byte("replace"), 0644)
	pkg.CheckError("error to create file to add utils.go", err)
}

func (mh *Memcached) createDirs(testDir string, testE2eDir string, testUtilsDir string) {
	err := os.Mkdir(testDir, os.ModePerm)
	pkg.CheckError("error to create test dir", err)
	err = os.Mkdir(testE2eDir, os.ModePerm)
	pkg.CheckError("error to create test e2e dir", err)
	err = os.Mkdir(testUtilsDir, os.ModePerm)
	pkg.CheckError("error to create test utils dir", err)
}

const e2eSuiteTemplate = `/*
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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	fmt.Fprintf(GinkgoWriter, "Starting Memcached Operator suite\n")
	RunSpecs(t, "Memcached e2e suite")
}
`

const e2eTemplate = `/*
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
	. "github.com/onsi/ginkgo"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"github.com/example/memcached-operator/test/utils"
)

// namespace store the ns where the Operator and Operand will be executed
const namespace = "memcached-operator-system"

var _ = Describe("memcached", func() {

	Context("ensure that Operator and Operand(s) can run in restricted namespaces", func() {
		BeforeEach(func() {
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

		AfterEach(func() {
			By("uninstalling the Prometheus manager bundle")
			utils.UninstallPrometheusOperator()

			By("uninstalling the cert-manager bundle")
			utils.UninstallCertManager()

			By("removing manager namespace")
			cmd := exec.Command("kubectl", "create", "ns", namespace)
			_, _ = utils.Run(cmd)
		})

		It("should successfully run the Memcached Operator", func() {
			var controllerPodName string
			var err error
			projectDir, _ := utils.GetProjectDir()

			// operatorImage store the name of the imahe used in the example
			const operatorImage = "example.com/memcached-operator:v0.0.1"

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", "IMG=example.com/memcached-operator:v0.0.1")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName("example.com/memcached-operator:v0.0.1")
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
				cmd =  exec.Command("kubectl", "get", 
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
`

const utilsTemplate = `/*
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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo" //nolint:golint,revive
)

const (
	prometheusOperatorVersion = "0.51"
	prometheusOperatorURL     = "https://raw.githubusercontent.com/prometheus-operator/" +
		"prometheus-operator/release-%s/bundle.yaml"

	certmanagerVersion = "v1.5.3"
	certmanagerURLTmpl = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"
)

func warnError(err error) {
	fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	_, err := Run(cmd)
	return err
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	fmt.Fprintf(GinkgoWriter, "running dir: %s\n", cmd.Dir)

	// To allow make commands be executed from the project directory which is subdir on SDK repo
	// TODO:(user) You might does not need the following code
	if err := os.Chdir(cmd.Dir); err != nil {
		fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}
`

const targetTemplate = `
.PHONY: test-e2e # You will need to have a Kind cluster up in running to run this target
test-e2e:
	go test ./test/e2e/ -v -ginkgo.v`
