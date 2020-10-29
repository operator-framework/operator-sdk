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
)

var _ = Describe("Integrating ansible Projects with OLM", func() {
	Context("with operator-sdk", func() {
		const operatorVersion = "0.0.1"

		It("should generate and run a valid OLM bundle and packagemanifests", func() {
			By("building the operator bundle image")
			err := tc.Make("bundle-build", "BUNDLE_IMG="+tc.BundleImageName)
			Expect(err).NotTo(HaveOccurred())

			if tc.IsRunningOnKind() {
				By("loading the bundle image into Kind cluster")
				err := tc.LoadImageToKindClusterWithName(tc.BundleImageName)
				Expect(err).NotTo(HaveOccurred())
			}

			By("adding the 'packagemanifests' rule to the Makefile")
			err = tc.AddPackagemanifestsTarget()
			Expect(err).NotTo(HaveOccurred())

			By("generating the operator package manifests")
			err = tc.Make("packagemanifests", "IMG="+tc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			By("running the package")
			runPkgManCmd := exec.Command(tc.BinaryName, "run", "packagemanifests",
				"--install-mode", "AllNamespaces",
				"--version", operatorVersion,
				"--timeout", "4m")
			_, err = tc.Run(runPkgManCmd)
			Expect(err).NotTo(HaveOccurred())

			By("destroying the deployed package manifests-formatted operator")
			cleanupPkgManCmd := exec.Command(tc.BinaryName, "cleanup", tc.ProjectName,
				"--timeout", "4m")
			_, err = tc.Run(cleanupPkgManCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
