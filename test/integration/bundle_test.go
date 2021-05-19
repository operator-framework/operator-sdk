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

package integration

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/testutils"
)

var _ = Describe("run bundle", func() {

	var (
		err    error
		output string
	)

	AfterEach(func() {
		By("cleaning up")
		_, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		output, err := tc.Kubectl.Wait(true, "pods", "--all", "--for=delete", "--timeout=2m")
		if err != nil && !strings.Contains(output, "no matching resources found") {
			Fail(err.Error())
		}
	})

	It("should handle existing operator deployments correctly", func() {
		output, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring(`package \"memcached-operator\" not found`))
		Expect(runBundle(&tc, "integration/memcached-operator-bundle:v0.0.1")).To(Succeed())
		Expect(runBundle(&tc, "integration/memcached-operator-bundle:v0.0.1")).NotTo(Succeed())
		_, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		output, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring(`package \"memcached-operator\" not found`))
	})

	It("should succeed with a single operator version in OwnNamespace mode", func() {
		Expect(runBundle(&tc, "integration/memcached-operator-bundle:v0.0.1", "--install-mode", "OwnNamespace")).To(Succeed())
	})

	It("should successfully deploy the second of two operator versions", func() {
		Skip("currently broken")
		By("building a v0.0.1 catalog and v0.2.0 bundle")
		Expect(tc.Make("catalog-build", "catalog-push", "IMAGE_TAG_BASE="+localImageBase)).To(Succeed())
		if onKind {
			inClusterCatalogImage := inClusterImageBase + "-catalog:v0.0.1"
			ExpectWithOffset(1, dockerTag(tc, localImageBase+"-catalog:v0.0.1", inClusterCatalogImage)).To(Succeed())
			ExpectWithOffset(1, tc.LoadImageToKindClusterWithName(inClusterCatalogImage)).To(Succeed())
		}

		inClusterBundleImage := inClusterImageBase + "-bundle:v0.2.0"
		Expect(tc.Make("bundle", "bundle-build", "bundle-push", "VERSION="+"0.2.0", "CHANNELS=stable", "IMG="+inClusterBundleImage, "IMAGE_TAG_BASE="+localImageBase)).To(Succeed())
		if onKind {
			ExpectWithOffset(1, dockerTag(tc, localImageBase+"-bundle:v0.2.0", inClusterBundleImage)).To(Succeed())
			ExpectWithOffset(1, tc.LoadImageToKindClusterWithName(inClusterBundleImage)).To(Succeed())
		}

		Expect(runBundle(&tc, "integration/memcached-operator-bundle:v0.2.0", "--index-image", inClusterImageBase+"-catalog:v0.0.1")).To(Succeed())
	})
})

func runBundle(tc *testutils.TestContext, args ...string) (err error) {
	args = append(args,
		inClusterRegistryHost+"/"+args[0],
		"--local-bundle", localRegistryHost+"/"+args[0],
		"--ca-secret-name", caSecretName,
	)
	return runCmd(tc, "bundle", args[1:]...)
}
