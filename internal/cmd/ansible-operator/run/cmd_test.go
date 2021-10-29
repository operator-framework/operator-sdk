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

package run

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running a verify config command", func() {

	Describe("verifyCfgURL", func() {
		It("verify valid URL", func() {
			err := verifyCfgURL("https://127.0.0.1:49810")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("verifyCfgURL", func() {
		It("verify valid URL with slash at the end", func() {
			err := verifyCfgURL("https://127.0.0.1:49810/")
			Expect(err).To(BeNil())
		})
	})

	Describe("verifyCfgURL", func() {
		It("Verify invalid URL and check if printed output contains path or not", func() {
			err := verifyCfgURL("https://127.0.0.1:49810/path")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("'https://127.0.0.1:49810/path' contains a path component, which the proxy server is currently unable to handle properly"))
		})
	})

	Describe("verifyCfgURL", func() {
		It("verify Empty URL", func() {
			err := verifyCfgURL("")
			Expect(err).To(BeNil())
		})
	})

})
