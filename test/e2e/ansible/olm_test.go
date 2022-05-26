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

package e2e_ansible_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

var _ = Describe("Integrating Ansible Projects with OLM", func() {
	Context("with operator-sdk", func() {
		var sample sample.Sample

		BeforeEach(func() {
			sample = ansibleSample
		})

		const operatorVersion = "0.0.1"

		It("should generate and run a valid OLM bundle and packagemanifests", func() {
			By("building the operator bundle image")
			err := olm.BuildBundleImage(sample, "bundle-"+image)
			Expect(err).NotTo(HaveOccurred())

			By("adding the 'packagemanifests' rule to the Makefile")
			err = olm.AddPackagemanifestsTarget(sample, projutil.OperatorTypeHelm)
			Expect(err).NotTo(HaveOccurred())

			By("generating the operator package manifests")
			err = olm.GeneratePackageManifests(sample, image)
			Expect(err).NotTo(HaveOccurred())

			By("running the package")
			runPkgManCmd := exec.Command(sample.Binary(), "run", "packagemanifests",
				"--install-mode", "AllNamespaces",
				"--version", operatorVersion,
				"--timeout", "4m")
			_, err = sample.CommandContext().Run(runPkgManCmd, sample.Name())
			Expect(err).NotTo(HaveOccurred())

			By("destroying the deployed package manifests-formatted operator")
			cleanupPkgManCmd := exec.Command(sample.Binary(), "cleanup", sample.Name(),
				"--timeout", "4m")
			_, err = sample.CommandContext().Run(cleanupPkgManCmd, sample.Name())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
