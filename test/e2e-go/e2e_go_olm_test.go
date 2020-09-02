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

package e2e_go_test

import (
	"encoding/json"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
)

var _ = Describe("Integrating Go Projects with OLM", func() {
	Context("with operator-sdk", func() {
		const operatorVersion = "0.0.1"

		const (
			OLMBundleValidationTest   = "olm-bundle-validation"
			OLMCRDsHaveValidationTest = "olm-crds-have-validation"
			OLMCRDsHaveResourcesTest  = "olm-crds-have-resources"
			OLMSpecDescriptorsTest    = "olm-spec-descriptors"
			OLMStatusDescriptorsTest  = "olm-status-descriptors"
		)

		It("should generate and run a valid OLM bundle and packagemanifests", func() {
			By("building the bundle")
			err := tc.Make("bundle", "IMG="+tc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			By("building the operator bundle image")
			err = tc.Make("bundle-build", "BUNDLE_IMG="+tc.BundleImageName)
			Expect(err).NotTo(HaveOccurred())

			if isRunningOnKind() {
				By("loading the bundle image into Kind cluster")
				err = tc.LoadImageToKindClusterWithName(tc.BundleImageName)
				Expect(err).NotTo(HaveOccurred())
			}

			By("adding the 'packagemanifests' rule to the Makefile")
			err = tc.AddPackagemanifestsTarget()
			Expect(err).NotTo(HaveOccurred())

			By("generating the operator package manifests")
			err = tc.Make("packagemanifests", "IMG="+tc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			By("running the package manifests-formatted operator")
			Expect(err).NotTo(HaveOccurred())
			runPkgManCmd := exec.Command(tc.BinaryName, "run", "packagemanifests",
				"--install-mode", "AllNamespaces",
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

			By("running custom scorecard tests")
			runScorecardCmd = exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector=suite=custom",
				"--output=json",
				"--wait-time=40s")
			scorecardOutputBytes, err = tc.Run(runScorecardCmd)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal(scorecardOutputBytes, &scorecardOutput)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(scorecardOutput.Items)).To(Equal(2))

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
			cleanupPkgManCmd := exec.Command(tc.BinaryName, "cleanup", tc.ProjectName,
				"--timeout", "4m")
			_, err = tc.Run(cleanupPkgManCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
