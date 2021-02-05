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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
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
			Expect(err).To(HaveOccurred())
			Expect(json.Unmarshal(outputBytes, &output)).To(Succeed())

			expected := map[string]v1alpha3.State{
				// Basic suite.
				"basic-check-spec": v1alpha3.PassState,
				// OLM suite.
				"olm-bundle-validation":    v1alpha3.PassState,
				"olm-crds-have-validation": v1alpha3.FailState,
				"olm-crds-have-resources":  v1alpha3.FailState,
				"olm-spec-descriptors":     v1alpha3.FailState,
				"olm-status-descriptors":   v1alpha3.FailState,
			}
			if strings.ToLower(operatorType) == "go" {
				// Go projects have generated CRD validation.
				expected["olm-crds-have-validation"] = v1alpha3.PassState
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

		It("should run a kuttl test suite successfully", func() {
			// Add kuttl tests to the scorecard's config.
			enableKuttlTests(tc)
			// Clean up after kuttl tests to avoid conflicts with other tests.
			defer func() {
				By("cleaning up CRs created by kuttl")
				// If multiple groups are being tested, append output of the following command for that group.
				crdNames, err := tc.Kubectl.Command("api-resources", "--api-group", "cache.example.com", "--output", "name")
				Expect(err).NotTo(HaveOccurred())
				crdNames = strings.Join(strings.Split(strings.TrimSpace(crdNames), "\n"), ",")
				kuttlCreatedCRsDeleted := func() error {
					output, err := tc.Kubectl.Get(true, crdNames, "-l", "test=kuttl")
					if err != nil {
						return fmt.Errorf("error getting all resources: %v", err)
					}
					if output = strings.TrimSpace(output); !strings.Contains(output, "No resources found") {
						return fmt.Errorf("kuttl resources still present: %s", output)
					}
					return nil
				}
				Eventually(kuttlCreatedCRsDeleted, 30*time.Second, time.Second).Should(Succeed())

				testutils.WrapWarn(tc.Make("undeploy"))
			}()

			// Run `operator-sdk scorecard` as it has been set up in `make test-scorecard`,
			// but with the 'suite=kuttl' selector, a longer timeout, and JSON output for marshaling purposes.
			cmd = exec.Command(tc.BinaryName, "scorecard", "testbundle",
				"--selector", "suite=kuttl",
				// Set namespace so kuttl runs with an appropriately-privileged service account.
				"--namespace", tc.Kubectl.Namespace,
				"--output", "json",
				"--wait-time", "2m")
			outputBytes, err = tc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(outputBytes, &output)).To(Succeed())

			Expect(output.Items).To(HaveLen(1))
			results := output.Items[0].Status.Results
			Expect(results).To(HaveLen(1))
			// Test name will be the test directory name, which is version in a non-multigroup scenario.
			Expect(results[0].Name).To(Equal(tc.Version))
			Expect(results[0].State).To(Equal(v1alpha3.PassState))
		})
	}
}

// enableKuttlTests exactly replicates the setup steps prior to and included in
// `make test-scorecard IMG=<operator image>`, but with a `kubectl wait` for the operator's deployment
// to be ready since this process is likely slow in test environments.
func enableKuttlTests(tc *testutils.TestContext) {
	var err error

	// Uncomment the kuttl config.
	ExpectWithOffset(1, testutils.UncommentCode(
		filepath.Join(tc.Dir, "config", "scorecard-testbundle", "kustomization.yaml"),
		`#- path: patches/kuttl.config.yaml
#  target:
#    group: scorecard.operatorframework.io
#    version: v1alpha3
#    kind: Configuration
#    name: config`, "#")).To(Succeed())

	// Regenerate the bundle to get the updated scorecard config.
	ExpectWithOffset(1, tc.Make("bundle", "deploy", "IMG="+tc.ImageName)).To(Succeed())

	// Wait for the operator to become ready.
	_, err = tc.Kubectl.Wait(true, "deployment.apps/memcached-operator-controller-manager",
		"--for", "condition=Available",
		"--timeout", "5m")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	// Create the test bundle.
	_, err = tc.Run(exec.Command("cp", "-r", "bundle", "testbundle"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	testBundleScorecardDir := filepath.Join(tc.Dir, "testbundle", "tests", "scorecard")
	ExpectWithOffset(1, os.MkdirAll(testBundleScorecardDir, 0755)).To(Succeed())

	// Build the testbundle scorecard config.
	out, err := tc.Run(exec.Command(filepath.Join(tc.Dir, "bin", "kustomize"), "build", filepath.Join("config", "scorecard-testbundle")))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, ioutil.WriteFile(filepath.Join(testBundleScorecardDir, "config.yaml"), out, 0666)).To(Succeed())

	// Copy all test cases to the bundle so `scorecard` can run them.
	_, err = tc.Run(exec.Command("cp", "-r", filepath.Join("test", "kuttl"), filepath.Join("testbundle", "tests", "scorecard")))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}
