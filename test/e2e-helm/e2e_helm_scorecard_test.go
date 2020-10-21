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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
)

var _ = Describe("Testing Helm Projects with Scorecard", func() {
	Context("with operator-sdk", func() {
		const (
			OLMBundleValidationTest   = "olm-bundle-validation"
			OLMCRDsHaveValidationTest = "olm-crds-have-validation"
			OLMCRDsHaveResourcesTest  = "olm-crds-have-resources"
			OLMSpecDescriptorsTest    = "olm-spec-descriptors"
			OLMStatusDescriptorsTest  = "olm-status-descriptors"
		)

		It("should work successfully with scorecard", func() {
			By("running basic scorecard tests")
			var scorecardOutput v1alpha3.TestList
			runScorecardCmd := exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector=suite=basic",
				"--output=json",
				"--wait-time=60s")
			scorecardOutputBytes, err := tc.Run(runScorecardCmd)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal(scorecardOutputBytes, &scorecardOutput)
			Expect(err).NotTo(HaveOccurred())
			Expect(scorecardOutput.Items).To(HaveLen(1))
			Expect(scorecardOutput.Items[0].Status.Results[0].State).To(Equal(v1alpha3.PassState))

			By("running olm scorecard tests")
			runOLMScorecardCmd := exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector=suite=olm",
				"--output=json",
				"--wait-time=60s")
			scorecardOutputBytes, err = tc.Run(runOLMScorecardCmd)
			Expect(err).To(HaveOccurred())
			err = json.Unmarshal(scorecardOutputBytes, &scorecardOutput)
			Expect(err).NotTo(HaveOccurred())

			expected := make(map[string]v1alpha3.State)
			expected[OLMBundleValidationTest] = v1alpha3.PassState
			expected[OLMCRDsHaveResourcesTest] = v1alpha3.FailState
			expected[OLMCRDsHaveValidationTest] = v1alpha3.FailState
			expected[OLMSpecDescriptorsTest] = v1alpha3.FailState
			expected[OLMStatusDescriptorsTest] = v1alpha3.FailState

			Expect(len(scorecardOutput.Items)).To(Equal(len(expected)))
			for a := 0; a < len(scorecardOutput.Items); a++ {
				fmt.Println("    - Name: ", scorecardOutput.Items[a].Status.Results[0].Name)
				fmt.Println("      Expected: ", expected[scorecardOutput.Items[a].Status.Results[0].Name])
				fmt.Println("      Output: ", scorecardOutput.Items[a].Status.Results[0].State)
				Expect(scorecardOutput.Items[a].Status.Results[0].State).To(Equal(expected[scorecardOutput.Items[a].Status.Results[0].Name]))
			}
		})
	})
})
