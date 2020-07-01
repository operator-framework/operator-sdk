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

package bundle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Checking operator-sdk bundle create command", func() {
	Describe("newCreateCmd", func() {
		It("builds and returns a cobra command", func() {
			cmd := newCreateCmd()
			Expect(cmd).NotTo(BeNil())

			flag := cmd.Flags().Lookup("directory")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("d"))

			flag = cmd.Flags().Lookup("output-dir")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("tag")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("t"))

			flag = cmd.Flags().Lookup("package")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("channels")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("c"))
			Expect(flag.DefValue).To(Equal("stable"))

			flag = cmd.Flags().Lookup("generate-only")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal("false"))

			flag = cmd.Flags().Lookup("overwrite")
			Expect(flag).NotTo(BeNil())
			Expect(flag.DefValue).To(Equal("false"))

			flag = cmd.Flags().Lookup("image-builder")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("b"))
			Expect(flag.DefValue).To(Equal("docker"))

			flag = cmd.Flags().Lookup("default-channel")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("e"))
		})
	})

	Describe("validate", func() {
		var cmd bundleCreateCmd
		BeforeEach(func() {
			cmd = bundleCreateCmd{}
		})

		It("fails if directory is not set", func() {
			err := cmd.validate([]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("--directory must be set"))
		})

		It("fails if package is not set", func() {
			cmd.directory = "apple"

			err := cmd.validate([]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("--package must be set"))
		})

		It("fails if default channel is not set", func() {
			cmd.directory = "banana"
			cmd.packageName = "cherry"

			err := cmd.validate([]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("--default-channel must be set"))
		})

		It("fails if GenerateOnly is false but a bundle image tag is not provided", func() {
			cmd.directory = "durian"
			cmd.packageName = "elderberry"
			cmd.defaultChannel = "fig"

			err := cmd.validate([]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("a bundle image tag is a required argument if --generate-only=true"))
		})

		It("fails if GenerateOnly is false and more than one arg is provided", func() {
			cmd.directory = "grapefruit"
			cmd.packageName = "honeydew"
			cmd.defaultChannel = "imbe"

			err := cmd.validate([]string{"aaa", "bbb"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("a bundle image tag is a required argument if --generate-only=true"))
		})

		It("succeeds if GenerateOnly is false and a bundle image tag is provided", func() {
			cmd.directory = "jackfruit"
			cmd.packageName = "kiwi"
			cmd.defaultChannel = "lime"

			err := cmd.validate([]string{"aaa"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails if GenerateOnly is true and any args are provided", func() {
			cmd.directory = "mango"
			cmd.packageName = "	nectarine"
			cmd.defaultChannel = "orange"
			cmd.generateOnly = true

			err := cmd.validate([]string{"aaa"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("the command does not accept any arguments if --generate-only=true"))
		})

		It("succeeds if GenerateOnly is true and no args are provided", func() {
			cmd.directory = "pineapple"
			cmd.packageName = "quince"
			cmd.defaultChannel = "raspberry"
			cmd.generateOnly = true

			err := cmd.validate([]string{})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
