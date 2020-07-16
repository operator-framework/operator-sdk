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

package scorecard

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running the scorecard command", func() {
	Describe("NewCmd", func() {
		It("builds and returns a cobra command", func() {
			cmd := NewCmd()
			Expect(cmd).NotTo(BeNil())

			flag := cmd.Flags().Lookup("kubeconfig")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("selector")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("l"))

			flag = cmd.Flags().Lookup("config")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("c"))

			flag = cmd.Flags().Lookup("namespace")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("n"))
			Expect(flag.DefValue).To(Equal("default"))

			flag = cmd.Flags().Lookup("output")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("o"))
			Expect(flag.DefValue).To(Equal("text"))

			flag = cmd.Flags().Lookup("service-account")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("s"))
			Expect(flag.DefValue).To(Equal("default"))

			flag = cmd.Flags().Lookup("list")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("L"))
			Expect(flag.DefValue).To(Equal("false"))

			flag = cmd.Flags().Lookup("skip-cleanup")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("x"))
			Expect(flag.DefValue).To(Equal("false"))

			flag = cmd.Flags().Lookup("wait-time")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("w"))
			Expect(flag.DefValue).To(Equal("30s"))
		})
	})

	Describe("validate", func() {
		var cmd scorecardCmd
		BeforeEach(func() {
			cmd = scorecardCmd{}
		})
		It("fails if anything other than exactly one arg is provided", func() {
			err := cmd.validate([]string{})
			Expect(err).To(HaveOccurred())

			err = cmd.validate([]string{"apple", "banana"})
			Expect(err).To(HaveOccurred())
		})

		It("succeeds if exactly one arg is provided", func() {
			input := "cherry"
			err := cmd.validate([]string{input})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
