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

package packagemanifests

import (
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest"
	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest/packagemanifestfakes"
)

var _ = Describe("Running a generate packagemanifests command", func() {
	var (
		c            packagemanifestsCmd
		crdsDir      string
		deployDir    string
		inputDir     string
		kustomizeDir string
		versionOne   string

		originalDir string
		testDataDir string
	)
	BeforeEach(func() {
		c = packagemanifestsCmd{}
		crdsDir = "crds"
		deployDir = "deploy"
		inputDir = "input"
		kustomizeDir = "kustomize"
		versionOne = "1.0.0"

		originalDir = filepath.Join("..", "..", "..", "..", "internal", "cmd", "operator-sdk", "generate", "packagemanifests")
		testDataDir = filepath.Join("..", "..", "..", "..", "..", "testdata", "go", "v4", "memcached-operator")
	})
	Describe("validate", func() {
		It("fails if no version is provided", func() {
			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version must be set"))
		})
		It("fails if a non-parsable version is provided", func() {
			c.version = "potato"

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("potato is not a valid semantic version"))
		})
		It("fails if an a non-parsable from-version is provided", func() {
			c.version = versionOne
			c.fromVersion = "1.0.a"

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("1.0.a is not a valid semantic version"))
		})
		It("fails if an input-dir is not provided", func() {
			c.version = versionOne

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("input-dir must be set"))
		})
		It("fails if a deploy-dir is not provided while not reading from stdin", func() {
			c.version = versionOne
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deploy-dir must be set if not reading from stdin"))
		})
		It("fails if a crds-dir is not provided while not reading from stdin", func() {
			c.version = versionOne
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir
			c.deployDir = deployDir

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be set if not reading from stdin"))
		})
		It("allows deply-dir and crds-dir to not be set if reading from a pipe such as stdin", func() {
			c.version = versionOne
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir
			r, _, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			origStdin := os.Stdin
			defer func() { os.Stdin = origStdin }()
			os.Stdin = r

			err = c.validate()
			Expect(err).NotTo(HaveOccurred())
		})
		It("fails if an output-dir is set while set to write to stdout", func() {
			c.version = versionOne
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir
			c.deployDir = deployDir
			c.crdsDir = crdsDir
			c.stdout = true
			c.outputDir = "output/"

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("output-dir cannot be set if writing to stdout"))
		})
		It("fails if default-channel is set but channel is not provided", func() {
			c.version = versionOne
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir
			c.deployDir = deployDir
			c.crdsDir = crdsDir
			c.isDefaultChannel = true

			err := c.validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("default-channel can only be set if --channel is set"))
		})
		It("validates successfully", func() {
			c.version = versionOne
			c.fromVersion = "0.1.2"
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir
			c.deployDir = deployDir
			c.crdsDir = "crds/"

			err := c.validate()
			Expect(err).NotTo(HaveOccurred())
		})
		It("succeeds if both default-channel and channel are set", func() {
			c.version = versionOne
			c.inputDir = inputDir
			c.kustomizeDir = kustomizeDir
			c.deployDir = deployDir
			c.crdsDir = crdsDir
			c.isDefaultChannel = true
			c.channelName = "alpha"

			err := c.validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("setDefaults", func() {
		Context("no project file is present", func() {
			It("fails if no correct operator name can be found", func() {
				err := c.setDefaults()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("--package <name> must be set if PROJECT config file is not present"))
			})
			It("sets fields on the command to default values", func() {
				c.packageName = "apricot"

				err := c.setDefaults()
				Expect(err).NotTo(HaveOccurred())
				Expect(c.inputDir).To(Equal(""))
				Expect(c.outputDir).To(Equal(defaultRootDir))
				Expect(c.generator).ToNot(BeNil())
				Expect(c.layout).To(Equal("unknown"))
			})
			It("does not set outputDir if stdout has been set", func() {
				c.packageName = "banana"
				c.stdout = true

				err := c.setDefaults()
				Expect(err).NotTo(HaveOccurred())
				Expect(c.outputDir).To(Equal(""))
			})
		})
		Context("a valid project file is present", func() {
			BeforeEach(func() {
				err := os.Chdir(testDataDir)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				err := os.Chdir(originalDir)
				Expect(err).NotTo(HaveOccurred())
			})
			It("reads the data from the project file", func() {
				err := c.setDefaults()
				Expect(err).NotTo(HaveOccurred())
				Expect(c.packageName).To(Equal("memcached-operator"))
				Expect(c.layout).To(Equal("go.kubebuilder.io/v4"))
			})
			It("doesn't overwrite the package name if it's already set", func() {
				c.packageName = "cherry"

				err := c.setDefaults()
				Expect(err).NotTo(HaveOccurred())
				Expect(c.packageName).To(Equal("cherry"))
				Expect(c.layout).To(Equal("go.kubebuilder.io/v4"))
			})
		})
		Context("an invalid project file is present", func() {
			BeforeEach(func() {
				err := os.Chdir("testdata")
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				err := os.Chdir("..")
				Expect(err).NotTo(HaveOccurred())
			})
			It("reads the data from the project file", func() {
				err := c.setDefaults()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("did not find expected key"))
			})
		})
	})
	Describe("generatePackageManifest", func() {
		var fakeGen packagemanifestfakes.FakeGenerator
		BeforeEach(func() {
			c.channelName = "apple"
			c.generator = &fakeGen
			c.inputDir = "banana/"
			c.isDefaultChannel = true
			c.outputDir = os.TempDir()
			c.packageName = "cherry"
			c.version = "1.2.3"
		})
		It("calls the package manifest generator with the correct params", func() {
			err := c.generatePackageManifest()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeGen.GenerateCallCount()).To(Equal(1))
			paramName, paramVersion, paramOutputDir, paramOpt := fakeGen.GenerateArgsForCall(0)
			Expect(paramName).To(Equal(c.packageName))
			Expect(paramVersion).To(Equal(c.version))
			Expect(paramOutputDir).To(Equal(c.outputDir))
			Expect((paramOpt)).To(Equal(packagemanifest.Options{
				BaseDir:          c.inputDir,
				ChannelName:      c.channelName,
				IsDefaultChannel: c.isDefaultChannel,
			}))
		})
		It("bubbles up errors from the generator", func() {
			potatoErr := errors.New("potato error")
			fakeGen.GenerateReturns(potatoErr)

			err := c.generatePackageManifest()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(potatoErr.Error()))
		})
	})
})
