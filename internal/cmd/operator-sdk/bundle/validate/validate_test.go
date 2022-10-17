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

package validate

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/validate"
)

var _ = Describe("Running a bundle validate command", func() {
	Describe("NewCmd", func() {
		var (
			cmd  = NewCmd()
			flag *pflag.Flag
		)

		It("builds and returns a cobra command", func() {
			Expect(cmd).NotTo(BeNil())

			flag = cmd.Flags().Lookup("image-builder")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("b"))
			Expect(flag.DefValue).To(Equal("docker"))

			flag = cmd.Flags().Lookup("select-optional")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("list-optional")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("output")
			Expect(flag).NotTo(BeNil())
			Expect(flag.Shorthand).To(Equal("o"))
			Expect(flag.DefValue).To(Equal(validate.TextOutput))
		})
	})

	Describe("Creating a logger", func() {
		It("that is Info Level when not verbose", func() {
			verbose := false
			logger := createLogger(verbose)
			Expect(logger.Logger.GetLevel()).To(Equal(log.InfoLevel))
		})
		It("that is Debug level if verbose", func() {
			verbose := true
			logger := createLogger(verbose)
			Expect(logger.Logger.GetLevel()).To(Equal(log.DebugLevel))
		})
	})

	Describe("validate", func() {
		var cmd bundleValidateCmd
		BeforeEach(func() {
			cmd = bundleValidateCmd{}
		})

		It("fails with no args", func() {
			err := cmd.validate([]string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("an image tag or directory is a required argument"))
		})
		It("fails with more than one arg", func() {
			err := cmd.validate([]string{"a", "b"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("an image tag or directory is a required argument"))
		})

		It("fails if the output format isnt text or json-alpha1", func() {
			wrongArg := "json-alpha2"
			cmd.outputFormat = wrongArg
			err := cmd.validate([]string{"quay.io/person/example"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid value for output flag: " + wrongArg))
		})

		It("succeeds if the arg is text or json-alpha1", func() {
			cmd.outputFormat = "text"
			err := cmd.validate([]string{"quay.io/person/example"})
			Expect(err).NotTo(HaveOccurred())

			cmd.outputFormat = "json-alpha1"
			err = cmd.validate([]string{"quay.io/person/example"})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
