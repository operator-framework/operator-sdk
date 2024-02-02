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

package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("run packagemanifests", func() {

	var (
		err    error
		output string
	)

	AfterEach(func() {
		By("cleaning up")
		_, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle existing operator deployments correctly", func() {
		output, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring(`package \"memcached-operator\" not found`))
		Expect(runPackageManifests(&tc, "--version", "0.0.1")).To(Succeed())
		Expect(runPackageManifests(&tc, "--version", "0.0.1")).NotTo(Succeed())
		_, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		output, err = cleanup(&tc)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring(`package \"memcached-operator\" not found`))
	})

	It("should succeed with a single operator version in AllNamespaces mode", func() {
		Expect(runPackageManifests(&tc, "--install-mode", "AllNamespaces", "--version", "0.0.1")).To(Succeed())
	})
})
