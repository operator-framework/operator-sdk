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

package packagemanifests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running a run packagemanifests command", func() {
	Describe("NewCmd", func() {
		It("builds a cobra command", func() {
			cmd := NewCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(BeNil())
			Expect(cmd.Short).NotTo(BeNil())
			Expect(cmd.Long).NotTo(BeNil())
			aliases := cmd.Aliases
			Expect(len(aliases)).To(Equal(1))
			Expect(aliases[0]).To(Equal("pm"))
		})
	})
	Describe("validate", func() {
		var (
			c   packagemanifestsCmd
			err error
		)
		BeforeEach(func() {
			c = packagemanifestsCmd{}
		})
		It("fails if provided more than 1 arg", func() {
			err = c.validate([]string{"foo", "bar"})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("exactly one argument is required"))
		})
		It("succeeds and if exactly 1 arg is provided", func() {
			arg := "baz"
			err = c.validate([]string{arg})
			Expect(err).To(BeNil())
		})
	})
	Describe("setDefaults", func() {
		var (
			c packagemanifestsCmd
		)
		BeforeEach(func() {
			c = packagemanifestsCmd{}
		})
		It("defaults to 'packagemanifests' if no args are provided", func() {
			c.setDefaults([]string{})
			Expect(c.ManifestsDir).To(Equal("packagemanifests"))
		})
		It("sets ManifestDir to the first arg if provided more than 0", func() {
			c.setDefaults([]string{"config/potato"})
			Expect(c.ManifestsDir).To(Equal("config/potato"))
		})
	})
})
