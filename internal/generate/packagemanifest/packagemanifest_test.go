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

package packagemanifest

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"sigs.k8s.io/yaml"

	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
)

func TestGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generator Suite")
}

const (
	pkgManDefaultContent = `channels:
  - currentCSV: memcached-operator.v0.0.1
    name: alpha
defaultChannel: alpha
packageName: memcached-operator
`

	pkgManSingleChannelContent = `channels:
  - currentCSV: memcached-operator.v0.0.1
    name: stable
defaultChannel: stable
packageName: memcached-operator
`
)

var (
	testDataDir = filepath.Join("..", "testdata")

	pkgManDefault, pkgManSingleChannel *apimanifests.PackageManifest
)

var _ = BeforeSuite(func() {
	initTestPackageManifestsHelper()
})

var _ = Describe("Generating a PackageManifest", func() {
	format.UseStringerRepresentation = true

	var (
		g            Generator
		buf          *bytes.Buffer
		operatorName = "memcached-operator"
		version      = "0.0.1"
	)

	BeforeEach(func() {
		buf = &bytes.Buffer{}
	})

	Describe("for the new Go project layout", func() {

		Context("with correct Options", func() {

			var (
				tmp string
				err error
			)

			BeforeEach(func() {
				tmp, err = ioutil.TempDir(".", "")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if tmp != "" {
					os.RemoveAll(tmp)
				}
			})

			It("should write a PackageManifest to an io.Writer", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      version,
				}
				opts := []Option{
					WithBase(testDataDir),
					WithWriter(buf),
				}
				Expect(g.Generate(opts...)).ToNot(HaveOccurred())
				Expect(buf.String()).To(MatchYAML(pkgManDefaultContent))
			})
			It("should write a PackageManifest to disk", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      version,
				}
				opts := []Option{
					WithBase(testDataDir),
					WithFileWriter(tmp),
				}
				Expect(g.Generate(opts...)).ToNot(HaveOccurred())
				outputFile := filepath.Join(tmp, makePkgManFileName(operatorName))
				Expect(outputFile).To(BeAnExistingFile())
				Expect(string(readFileHelper(outputFile))).To(MatchYAML(pkgManDefaultContent))
			})
		})

		Context("with incorrect configuration", func() {

			BeforeEach(func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      version,
				}
			})

			It("should return an error without any Options", func() {
				opts := []Option{}
				Expect(g.Generate(opts...)).To(MatchError(errNoGetWriter))
			})
			It("should return an error without a getWriter", func() {
				opts := []Option{
					WithBase(testDataDir),
				}
				Expect(g.Generate(opts...)).To(MatchError(errNoGetWriter))
			})
			It("should return an error without a getBase", func() {
				opts := []Option{
					WithWriter(&bytes.Buffer{}),
				}
				Expect(g.Generate(opts...)).To(MatchError(errNoGetBase))
			})
			It("should return an error without a Version", func() {
				g.Version = ""
				opts := []Option{
					WithBase(testDataDir),
					WithWriter(&bytes.Buffer{}),
				}
				Expect(g.Generate(opts...)).To(MatchError(errNoVersion))
			})
		})

		Context("to create a new PackageManifest", func() {
			It("should return the default file", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      version,
					getBase:      makeBaseGetter(pkgManDefaultContent),
				}
				pkg, err := g.generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(pkg).To(Equal(pkgManDefault))
			})
			It("should return a PackageManifest with a non-default channel", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      version,
					ChannelName:  "stable",
					getBase:      makeBaseGetter(pkgManSingleChannelContent),
				}
				pkg, err := g.generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(pkg).To(Equal(pkgManSingleChannel))
			})
		})

		Context("to update an existing PackageManifest file", func() {
			It("should return a PackageManifest with one updated channel CSV name", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      "0.0.2",
					ChannelName:  "alpha",
					getBase:      makeBaseGetter(pkgManDefaultContent),
				}
				pkg, err := g.generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(pkg).To(Equal(&apimanifests.PackageManifest{
					Channels: []apimanifests.PackageChannel{
						{Name: "alpha", CurrentCSVName: genutil.MakeCSVName(operatorName, "0.0.2")},
					},
					DefaultChannelName: "alpha",
					PackageName:        operatorName,
				}))
			})
			It("should return a PackageManifest with two channels", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      "0.0.2",
					ChannelName:  "stable",
					getBase:      makeBaseGetter(pkgManDefaultContent),
				}
				pkg, err := g.generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(pkg).To(Equal(&apimanifests.PackageManifest{
					Channels: []apimanifests.PackageChannel{
						{Name: "alpha", CurrentCSVName: genutil.MakeCSVName(operatorName, version)},
						{Name: "stable", CurrentCSVName: genutil.MakeCSVName(operatorName, "0.0.2")},
					},
					DefaultChannelName: "alpha",
					PackageName:        operatorName,
				}))
			})
			It("should return a PackageManifest with two channels and an updated default channel", func() {
				g = Generator{
					OperatorName:     operatorName,
					Version:          "0.0.2",
					ChannelName:      "stable",
					IsDefaultChannel: true,
					getBase:          makeBaseGetter(pkgManDefaultContent),
				}
				pkg, err := g.generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(pkg).To(Equal(&apimanifests.PackageManifest{
					Channels: []apimanifests.PackageChannel{
						{Name: "alpha", CurrentCSVName: genutil.MakeCSVName(operatorName, version)},
						{Name: "stable", CurrentCSVName: genutil.MakeCSVName(operatorName, "0.0.2")},
					},
					DefaultChannelName: "stable",
					PackageName:        operatorName,
				}))
			})
		})

	})

})

func makeBaseGetter(content string) getBaseFunc {
	return func() (*apimanifests.PackageManifest, error) {
		return marshalContent(content)
	}
}

func initTestPackageManifestsHelper() {
	var err error
	pkgManDefault, err = marshalContent(pkgManDefaultContent)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	pkgManSingleChannel, err = marshalContent(pkgManSingleChannelContent)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func readFileHelper(path string) []byte {
	b, err := ioutil.ReadFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return b
}

func marshalContent(content string) (*apimanifests.PackageManifest, error) {
	base := &apimanifests.PackageManifest{}
	if content == "" {
		return base, nil
	}
	err := yaml.Unmarshal([]byte(content), base)
	return base, err
}
