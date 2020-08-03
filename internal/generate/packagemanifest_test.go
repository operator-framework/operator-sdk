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

package generate_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/generate"
)

var _ = Describe("A package manifest generator", func() {
	Describe("GeneratePackageManifest", func() {
		var (
			g                                    generate.Generator
			buffer                               *bytes.Buffer
			pkgManDefault                        string
			pkgManOneChannel                     string
			pkgManUpdatedOneChannel              string
			pkgManUpdatedSecondChannel           string
			pkgManUpdatedSecondChannelNewDefault string
		)
		BeforeEach(func() {
			buffer = bytes.NewBuffer([]byte{})
			pkgManDefault = `channels:
- currentCSV: memcached-operator.v0.0.1
  name: alpha
defaultChannel: alpha
packageName: memcached-operator
`
			pkgManOneChannel = `channels:
- currentCSV: memcached-operator.v0.0.1
  name: stable
defaultChannel: stable
packageName: memcached-operator
`
			pkgManUpdatedOneChannel = `channels:
- currentCSV: memcached-operator.v0.0.2
  name: alpha
defaultChannel: alpha
packageName: memcached-operator
`
			pkgManUpdatedSecondChannel = `channels:
- currentCSV: memcached-operator.v0.0.1
  name: alpha
- currentCSV: memcached-operator.v0.0.2
  name: stable
defaultChannel: alpha
packageName: memcached-operator
`
			pkgManUpdatedSecondChannelNewDefault = `channels:
- currentCSV: memcached-operator.v0.0.1
  name: alpha
- currentCSV: memcached-operator.v0.0.2
  name: stable
defaultChannel: stable
packageName: memcached-operator
`
		})
		Context("when writing a new package manifest", func() {
			It("writes a package manifest", func() {
				opts := &generate.PkgOptions{
					OperatorName: "memcached-operator",
					OutputWriter: buffer,
					Version:      "0.0.1",
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buffer.String())).To(Equal(pkgManDefault))
			})
			It("writes a package manifest with a non-default channel", func() {
				opts := &generate.PkgOptions{
					OperatorName: "memcached-operator",
					OutputWriter: buffer,
					Version:      "0.0.1",
					ChannelName:  "stable",
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buffer.String())).To(Equal(pkgManOneChannel))
			})
		})
		Context("when updating an existing package manifest", func() {
			It("updates an existing package manifest with a updated channel", func() {
				opts := &generate.PkgOptions{
					OperatorName: "memcached-operator",
					OutputWriter: buffer,
					BaseDir:      "testdata",
					ChannelName:  "alpha",
					Version:      "0.0.2",
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buffer.String())).To(Equal(pkgManUpdatedOneChannel))
			})
			It("updates an existing package manifest with a new channel", func() {
				opts := &generate.PkgOptions{
					OperatorName: "memcached-operator",
					OutputWriter: buffer,
					BaseDir:      "testdata",
					ChannelName:  "stable",
					Version:      "0.0.2",
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buffer.String())).To(Equal(pkgManUpdatedSecondChannel))
			})
			It("updates an existing package manifest with a new channel and an updated default channel", func() {
				opts := &generate.PkgOptions{
					OperatorName:     "memcached-operator",
					OutputWriter:     buffer,
					BaseDir:          "testdata",
					ChannelName:      "stable",
					Version:          "0.0.2",
					IsDefaultChannel: true,
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(buffer.String())).To(Equal(pkgManUpdatedSecondChannelNewDefault))
			})
		})
		Context("when incorrect params are provided", func() {
			It("fails if provided nil ops", func() {
				err := g.GeneratePackageManifest(nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("generator options must be set"))
			})
			It("fails if no operator name is specified", func() {
				opts := &generate.PkgOptions{}

				err := g.GeneratePackageManifest(opts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("operator name must be set"))
			})
			It("fails if no output writer is set", func() {
				opts := &generate.PkgOptions{
					OperatorName: "memcached-operator",
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("output writer must be set"))
			})
			It("fails if no version is specified", func() {
				opts := &generate.PkgOptions{
					OperatorName: "memcached-operator",
					OutputWriter: buffer,
					BaseDir:      "testdata",
				}

				err := g.GeneratePackageManifest(opts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("version must be set"))
			})
		})
	})
})
