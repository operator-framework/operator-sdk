// Copyright 2021 The Operator-SDK Authors
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

package common

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// ScorecardSpec runs a set of scorecard tests common to all operator types.
func ScorecardSpec(tc *testutils.TestContext, operatorType string) func() {
	return func() {
		var (
			err         error
			cmd         *exec.Cmd
			outputBytes []byte
			output      v1alpha3.TestList
		)

		It("should run a single scorecard test successfully", func() {
			cmd = exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector", "suite=basic",
				"--output", "json",
				"--wait-time", "2m")
			outputBytes, err = tc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(outputBytes, &output)).To(Succeed())

			Expect(output.Items).To(HaveLen(1))
			results := output.Items[0].Status.Results
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("basic-check-spec"))
			Expect(results[0].State).To(Equal(v1alpha3.PassState))
		})

		It("should run all enabled scorecard tests successfully", func() {
			cmd = exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--output", "json",
				"--wait-time", "4m")
			outputBytes, err = tc.Run(cmd)
			// Some tests are expected to fail, which results in scorecard exiting 1.
			// Go tests no longer expect to fail
			if strings.ToLower(operatorType) != "go" {
				Expect(err).To(HaveOccurred())
			}
			Expect(json.Unmarshal(outputBytes, &output)).To(Succeed())

			expected := map[string]v1alpha3.State{
				// Basic suite.
				"basic-check-spec": v1alpha3.PassState,
				// OLM suite.
				"olm-bundle-validation":    v1alpha3.PassState,
				"olm-crds-have-validation": v1alpha3.FailState,
				"olm-crds-have-resources":  v1alpha3.FailState,
				"olm-spec-descriptors":     v1alpha3.FailState,
				// For Ansible/Helm should PASS with a Suggestion
				// For Golang should pass because we have status spec and descriptions
				"olm-status-descriptors": v1alpha3.PassState,
			}
			if strings.ToLower(operatorType) == "go" {
				// Go projects have generated CRD validation.
				expected["olm-crds-have-validation"] = v1alpha3.PassState
				// Go generated test operator now has CSV markers
				// that allows these validations to pass
				expected["olm-crds-have-resources"] = v1alpha3.PassState
				expected["olm-spec-descriptors"] = v1alpha3.PassState
				expected["olm-status-descriptors"] = v1alpha3.PassState
				// The Go sample project tests a custom suite.
				expected["customtest1"] = v1alpha3.PassState
				expected["customtest2"] = v1alpha3.PassState
			}

			Expect(output.Items).To(HaveLen(len(expected)))
			for i := 0; i < len(output.Items); i++ {
				results := output.Items[i].Status.Results
				Expect(results).To(HaveLen(1))
				Expect(results[0].Name).NotTo(BeEmpty())
				fmt.Fprintln(GinkgoWriter, "    - Name: ", results[0].Name)
				fmt.Fprintln(GinkgoWriter, "      Expected: ", expected[results[0].Name])
				fmt.Fprintln(GinkgoWriter, "      Output: ", results[0].State)
				Expect(results[0].State).To(Equal(expected[results[0].Name]))
			}
		})

		It("should configure scorecard storage successfully", func() {
			cmd = exec.Command(tc.BinaryName, "scorecard", "bundle",
				"--selector", "suite=basic",
				"--output", "json",
				"--test-output", "/testdata",
				"--wait-time", "4m")
			outputBytes, err = tc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(outputBytes, &output)).To(Succeed())

			Expect(output.Items).To(HaveLen(1))
			results := output.Items[0].Status.Results
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("basic-check-spec"))
			Expect(results[0].State).To(Equal(v1alpha3.PassState))
		})
	}
}
