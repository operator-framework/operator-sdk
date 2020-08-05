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
	"path"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/operator-framework/operator-sdk/test/internal"
)

var _ = Describe("Integrating Helm Projects with OLM", func() {
	Context("with operator-sdk", func() {
		const operatorVersion = "0.0.1"

		BeforeEach(func() {
			By("turning off interactive prompts for all generation tasks.")
			replace := "operator-sdk generate kustomize manifests"
			testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
		})

		It("should generate and run a valid OLM bundle and packagemanifests", func() {
			By("building the bundle")
			err := tc.Make("bundle", "IMG="+tc.ImageName)
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
			cleanupPkgManCmd := exec.Command(tc.BinaryName, "cleanup", projectName,
				"--timeout", "4m")
			_, err = tc.Run(cleanupPkgManCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
