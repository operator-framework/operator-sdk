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
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
)

var _ = Describe("Running Helm projects", func() {
	Context("built with operator-sdk", func() {
		BeforeEach(func() {
			By("Installing CRD's")
			err := operator.InstallCRDs(helmSampleValidKubeConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("Uninstalling CRD's")
			err := operator.UninstallCRDs(helmSampleValidKubeConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should run correctly when run locally", func() {
			By("Running the project")
			cmd := exec.Command("make", "run")
			cmd.Dir = helmSample.Dir()
			err := cmd.Start()
			Expect(err).NotTo(HaveOccurred())

			By("Killing the project")
			err = cmd.Process.Kill()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
