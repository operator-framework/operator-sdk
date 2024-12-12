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

package packagemanifest_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest"
)

var _ = Describe("A package manifest generator", func() {
	var (
		testDataDir string
	)
	BeforeEach(func() {
		testDataDir = filepath.Join("..", "testdata")
	})
	Describe("Generate", func() {
		var (
			g                                    packagemanifest.Generator
			blankOpts                            packagemanifest.Options
			operatorName                         string
			outputDir                            string
			pkgManFilename                       string
			pkgManDefault                        string
			pkgManOneChannel                     string
			pkgManUpdatedOneChannel              string
			pkgManUpdatedSecondChannel           string
			pkgManUpdatedSecondChannelNewDefault string
		)
		BeforeEach(func() {
			g = packagemanifest.NewGenerator()
			operatorName = "memcached-operator"
			blankOpts = packagemanifest.Options{}
			pkgManFilename = operatorName + ".package.yaml"
			outputDir = os.TempDir()
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
				err := g.Generate(operatorName, "0.0.1", outputDir, blankOpts)
				Expect(err).NotTo(HaveOccurred())
				file, err := os.ReadFile(outputDir + string(os.PathSeparator) + pkgManFilename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(file)).To(Equal(pkgManDefault))
			})
			It("writes a package manifest with a non-default channel", func() {
				opts := packagemanifest.Options{
					ChannelName: "stable",
				}

				err := g.Generate(operatorName, "0.0.1", outputDir, opts)
				Expect(err).NotTo(HaveOccurred())
				file, err := os.ReadFile(outputDir + string(os.PathSeparator) + pkgManFilename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(file)).To(Equal(pkgManOneChannel))
			})
		})
		Context("when updating an existing package manifest", func() {
			It("creates a new package manifest if provided an existing packagemanifest that doesn't exist", func() {
				opts := packagemanifest.Options{
					BaseDir:     "testpotato",
					ChannelName: "stable",
				}

				err := g.Generate(operatorName, "0.0.1", outputDir, opts)
				Expect(err).NotTo(HaveOccurred())
				file, err := os.ReadFile(outputDir + string(os.PathSeparator) + pkgManFilename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(file)).To(Equal(pkgManOneChannel))
			})
			It("updates an existing package manifest with a updated channel", func() {
				opts := packagemanifest.Options{
					BaseDir:     testDataDir,
					ChannelName: "alpha",
				}

				err := g.Generate(operatorName, "0.0.2", outputDir, opts)
				Expect(err).NotTo(HaveOccurred())
				file, err := os.ReadFile(outputDir + string(os.PathSeparator) + pkgManFilename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(file)).To(Equal(pkgManUpdatedOneChannel))
			})
			It("updates an existing package manifest with a new channel", func() {
				opts := packagemanifest.Options{
					BaseDir:     testDataDir,
					ChannelName: "stable",
				}

				err := g.Generate(operatorName, "0.0.2", outputDir, opts)
				Expect(err).NotTo(HaveOccurred())
				file, err := os.ReadFile(outputDir + string(os.PathSeparator) + pkgManFilename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(file)).To(Equal(pkgManUpdatedSecondChannel))
			})
			It("updates an existing package manifest with a new channel and an updated default channel", func() {
				opts := packagemanifest.Options{
					BaseDir:          testDataDir,
					ChannelName:      "stable",
					IsDefaultChannel: true,
				}

				err := g.Generate(operatorName, "0.0.2", outputDir, opts)
				Expect(err).NotTo(HaveOccurred())
				file, err := os.ReadFile(outputDir + string(os.PathSeparator) + pkgManFilename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(file)).To(Equal(pkgManUpdatedSecondChannelNewDefault))
			})
		})
		Context("when incorrect params are provided", func() {
			It("fails if no operator name is specified", func() {
				err := g.Generate("", "", "", blankOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(packagemanifest.ErrNoOpName.Error()))
			})
			It("fails if no version is specified", func() {
				err := g.Generate(operatorName, "", "", blankOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(packagemanifest.ErrNoVersion.Error()))
			})
			It("fails if no output directory is set", func() {
				err := g.Generate(operatorName, "0.0.1", "", blankOpts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(packagemanifest.ErrNoOutputDir.Error()))
			})
		})
	})
	Describe("GetBase", func() {
		var (
			b packagemanifest.PackageManifest
		)
		BeforeEach(func() {
			b = packagemanifest.PackageManifest{}
		})
		It("returns a new blank packagemanifest", func() {
			b.PackageName = "sweetsop"

			pm, err := b.GetBase()
			Expect(err).NotTo(HaveOccurred())
			Expect(pm).NotTo(BeNil())
			Expect(pm.PackageName).To(Equal(b.PackageName))
		})
		It("reads an existing packagemanifest from disk", func() {
			b.BasePath = filepath.Join(testDataDir, "memcached-operator.package.yaml")

			pm, err := b.GetBase()
			Expect(err).NotTo(HaveOccurred())
			Expect(pm).NotTo(BeNil())
			Expect(pm.PackageName).To(Equal("memcached-operator"))
			Expect(pm.Channels).To(HaveLen(1))
			Expect(pm.Channels[0].Name).To(Equal("alpha"))
			Expect(pm.Channels[0].CurrentCSVName).To(Equal("memcached-operator.v0.0.1"))
			Expect(pm.DefaultChannelName).To(Equal("alpha"))
		})
		It("fails if provided a non-existent base path", func() {
			b.BasePath = "not-a-real-thing.yaml"

			pm, err := b.GetBase()
			Expect(pm).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error reading existing"))
		})
	})
})
