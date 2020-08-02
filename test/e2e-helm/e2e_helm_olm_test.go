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
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint
	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"

	testutils "github.com/operator-framework/operator-sdk/test/internal"
)

var _ = Describe("Integrating Helm Projects with OLM", func() {
	Context("with operator-sdk", func() {
		const operatorVersion = "0.0.1"

		const (
			OLMBundleValidationTest   = "olm-bundle-validation"
			OLMCRDsHaveValidationTest = "olm-crds-have-validation"
			OLMCRDsHaveResourcesTest  = "olm-crds-have-resources"
			OLMSpecDescriptorsTest    = "olm-spec-descriptors"
			OLMStatusDescriptorsTest  = "olm-status-descriptors"
		)

		BeforeEach(func() {
			By("turning off interactive prompts for all generation tasks.")
			replace := "operator-sdk generate kustomize manifests"
			testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
		})

		AfterEach(func() {
			By("destroying the deployed package manifests-formatted operator")
			cleanupPkgManCmd := exec.Command(tc.BinaryName, "cleanup", "packagemanifests",
				"--version", operatorVersion,
				"--timeout", "4m")
			_, _ = tc.Run(cleanupPkgManCmd)

			By("uninstalling CRD's")
			_ = tc.Make("uninstall")
		})

		It("should generate and run a valid OLM bundle and packagemanifests", func() {
			By("building the bundle")
			err := tc.Make("bundle")
			Expect(err).NotTo(HaveOccurred())

			By("validating the bundle")
			bundleValidateCmd := exec.Command(tc.BinaryName, "bundle", "validate", "bundle")
			_, err = tc.Run(bundleValidateCmd)
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

			By("loading the project image into Kind cluster")
			err = tc.LoadImageToKindClusterWithName(bundleImage)
			Expect(err).Should(Succeed())

			By("adding the 'packagemanifests' rule to the Makefile")
			err = tc.AddPackagemanifestsTarget()
			Expect(err).Should(Succeed())

			By("generating the operator package manifests")
			err = tc.Make("packagemanifests")
			Expect(err).NotTo(HaveOccurred())

			By("updating clusterserviceversion with the manager image")
			testutils.ReplaceInFile(
				filepath.Join(tc.Dir, "packagemanifests", operatorVersion,
					fmt.Sprintf("e2e-%s.clusterserviceversion.yaml", tc.TestSuffix)),
				"controller:latest", tc.ImageName)

			By("installing crds to run packagemanifests")
			err = tc.Make("install")
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

			By("running the package")
			runPkgManCmd := exec.Command(tc.BinaryName, "run", "packagemanifests",
				"--install-mode", "AllNamespaces",
				"--version", operatorVersion,
				"--timeout", "4m")
			_, err = tc.Run(runPkgManCmd)
			Expect(err).NotTo(HaveOccurred())

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
		})
	})
})
