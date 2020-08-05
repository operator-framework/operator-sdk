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

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"

	testutils "github.com/operator-framework/operator-sdk/test/internal"
)

const (
	OLMBundleValidationTest   = "olm-bundle-validation"
	OLMCRDsHaveValidationTest = "olm-crds-have-validation"
	OLMCRDsHaveResourcesTest  = "olm-crds-have-resources"
	OLMSpecDescriptorsTest    = "olm-spec-descriptors"
	OLMStatusDescriptorsTest  = "olm-status-descriptors"
)

var _ = Describe("operator-sdk", func() {
	Context("with the new project layout", func() {
		var (
			tc              testutils.TestContext
			projectName     string
			operatorVersion = "0.0.1"
		)

		BeforeEach(func() {
			By("creating a new test context")
			var err error
			tc, err = testutils.NewTestContext("GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(tc.Prepare()).To(Succeed())
			projectName = filepath.Base(tc.Dir)

			By("installing OLM")
			Expect(tc.InstallOLM()).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up created API objects during test process")
			tc.CleanupManifests(filepath.Join("config", "default"))

			By("removing container image and work dir")
			tc.Destroy()

			By("uninstalling OLM")
			tc.UninstallOLM()
		})

		It("should generate a runnable project", func() {
			var controllerPodName string
			By("initializing a project")
			err := tc.Init(
				"--project-version", "3-alpha",
				"--repo", path.Join("github.com", "example", projectName),
				"--domain", tc.Domain,
				"--fetch-deps=false")
			Expect(err).Should(Succeed())

			By("creating an API definition")
			err = tc.CreateAPI(
				"--group", tc.Group,
				"--version", tc.Version,
				"--kind", tc.Kind,
				"--namespaced",
				"--resource",
				"--controller",
				"--make=false")
			Expect(err).Should(Succeed())

			By("implementing the API")
			Expect(kbtestutils.InsertCode(
				filepath.Join(tc.Dir, "api", tc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(tc.Kind))),
				fmt.Sprintf(`type %sSpec struct {
`, tc.Kind),
				`	// +optional
	Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())

			By("building the operator image")
			err = tc.Make("docker-build", "IMG="+tc.ImageName)
			Expect(err).Should(Succeed())

			kubectx, err := tc.Kubectl.Command("config", "current-context")
			Expect(err).Should(Succeed())

			if strings.Contains(kubectx, "kind") {
				By("loading the operator image into the test cluster")
				err = tc.LoadImageToKindCluster()
				Expect(err).Should(Succeed())
			}

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
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Successfully Reconciled"))

			By("cleaning up the operator and resources")
			defaultOutput, err := tc.KustomizeBuild(filepath.Join("config", "default"))
			Expect(err).NotTo(HaveOccurred())
			_, err = tc.Kubectl.WithInput(string(defaultOutput)).Command("delete", "-f", "-")
			Expect(err).NotTo(HaveOccurred())

			By("generating the operator bundle")
			// Turn off interactive prompts for all generation tasks.
			replace := "operator-sdk generate kustomize manifests"
			testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
			err = tc.Make("bundle", "IMG="+tc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			By("building the operator bundle image")
			// Use the existing image tag but with a "-bundle" suffix.
			imageSplit := strings.SplitN(tc.ImageName, ":", 2)
			bundleImage := path.Join("quay.io", imageSplit[0]+"-bundle")
			if len(imageSplit) == 2 {
				bundleImage += ":" + imageSplit[1]
			}
			err = tc.Make("bundle-build", "BUNDLE_IMG="+bundleImage)
			Expect(err).NotTo(HaveOccurred())

			By("generating the operator package manifests")
			Expect(tc.Make("manifests")).Should(Succeed())
			genKustomizeCmd := exec.Command(tc.BinaryName, "generate", "kustomize", "manifests")
			_, err = tc.Run(genKustomizeCmd)
			Expect(err).NotTo(HaveOccurred())
			manifestsOutput, err := tc.KustomizeBuild(filepath.Join("config", "manifests"))
			Expect(err).NotTo(HaveOccurred())
			genPkgManCmd := exec.Command(tc.BinaryName, "generate", "packagemanifests", "--version", "0.0.1")
			tc.Stdin = bytes.NewBuffer(manifestsOutput)
			_, err = tc.Run(genPkgManCmd)
			Expect(err).NotTo(HaveOccurred())

			By("running the package manifests-formatted operator")
			_, err = tc.Kubectl.Command("create", "namespace", tc.Kubectl.Namespace)
			Expect(err).NotTo(HaveOccurred())
			runPkgManCmd := exec.Command(tc.BinaryName, "run", "packagemanifests",
				"--install-mode", "AllNamespaces",
				"--namespace", tc.Kubectl.Namespace,
				"--version", operatorVersion,
				"--timeout", "4m")
			_, err = tc.Run(runPkgManCmd)
			Expect(err).NotTo(HaveOccurred())

			By("running basic scorecard tests")
			var scorecardOutput v1alpha3.TestList
			runScorecardCmd := exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector=suite=basic",
				"--output=json",
				"--wait-time=40s")
			scorecardOutputBytes, err := tc.Run(runScorecardCmd)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal(scorecardOutputBytes, &scorecardOutput)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(scorecardOutput.Items)).To(Equal(1))
			Expect(scorecardOutput.Items[0].Status.Results[0].State).To(Equal(v1alpha3.PassState))

			By("running olm scorecard tests")
			runOLMScorecardCmd := exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector=suite=olm",
				"--output=json",
				"--wait-time=40s")
			scorecardOutputBytes, err = tc.Run(runOLMScorecardCmd)
			Expect(err).To(HaveOccurred())
			err = json.Unmarshal(scorecardOutputBytes, &scorecardOutput)
			Expect(err).NotTo(HaveOccurred())

			resultTable := make(map[string]v1alpha3.State)
			resultTable[OLMStatusDescriptorsTest] = v1alpha3.FailState
			resultTable[OLMCRDsHaveResourcesTest] = v1alpha3.FailState
			resultTable[OLMBundleValidationTest] = v1alpha3.PassState
			resultTable[OLMSpecDescriptorsTest] = v1alpha3.FailState
			resultTable[OLMCRDsHaveValidationTest] = v1alpha3.PassState

			Expect(len(scorecardOutput.Items)).To(Equal(len(resultTable)))
			for a := 0; a < len(scorecardOutput.Items); a++ {
				Expect(scorecardOutput.Items[a].Status.Results[0].State).To(Equal(resultTable[scorecardOutput.Items[a].Status.Results[0].Name]))
			}

			By("destroying the deployed package manifests-formatted operator")
			cleanupPkgManCmd := exec.Command(tc.BinaryName, "cleanup", projectName,
				"--namespace", tc.Kubectl.Namespace,
				"--timeout", "4m")
			_, err = tc.Run(cleanupPkgManCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
