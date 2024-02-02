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

package pkgmantobundle

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Running pkgmanToBundle command", func() {
	var (
		p         pkgManToBundleCmd
		pkgManDir string
		outputDir string = "bundle-output"
	)

	BeforeEach(func() {
		p = pkgManToBundleCmd{}
	})

	Describe("validate", func() {
		It("fail if anything other than one argumanet for packagemanifest directory is provided", func() {
			err := p.validate([]string{})
			Expect(err).To(HaveOccurred())

			err = p.validate([]string{"one", "two"})
			Expect(err).To(HaveOccurred())
		})

		It("succeeds if exactly one argument is provided", func() {
			err := p.validate([]string{"inputdir"})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("migrate packagemanifests to bundle ", func() {
		AfterEach(func() {
			err := os.RemoveAll(outputDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should generate multiple bundles for each version of manifests", func() {
			// Specify input package manifest directory and output directory
			pkgManDir = filepath.Join("testdata", "packagemanifests")

			p.pkgmanifestDir = pkgManDir
			p.outputDir = outputDir
			p.baseImg = "quay.io/example/memcached-operator"

			err := p.run()
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the number of bundles created")
			Expect(getNumberOfDirectories(p.outputDir)).To(BeEquivalentTo(2))

			By("Verifying that each of them are valid bundles and their package name")
			bundles, err := os.ReadDir(p.outputDir)
			Expect(err).NotTo(HaveOccurred())

			for _, bundle := range bundles {
				b, err := apimanifests.GetBundleFromDir(filepath.Join(p.outputDir, bundle.Name()))
				Expect(err).NotTo(HaveOccurred())
				Expect(b).NotTo(BeNil())

				// Verifying that bundle contains required files
				Expect(fileExists(filepath.Join(p.outputDir, bundle.Name(), "bundle.Dockerfile"))).To(BeTrue())
				Expect(fileExists(filepath.Join(p.outputDir, bundle.Name(), defaultSubBundleDir, "metadata", "annotations.yaml"))).To(BeTrue())
				Expect(b.CSV).NotTo(BeNil())
				Expect(b.V1CRDs).NotTo(BeNil())

				// Verify if scorecard config exiss in the bundle
				if bundle.Name() == "bundle-0.0.1" {
					Expect(fileExists(filepath.Join(p.outputDir, bundle.Name(), defaultSubBundleDir, "tests", "scorecard", "config.yaml"))).To(BeTrue())
				}
			}
		})

		It("should build image when build command is provided", func() {
			// Specify input package manifest directory and output directory
			pkgManDir = filepath.Join("testdata", "packagemanifests")

			p.pkgmanifestDir = pkgManDir
			p.outputDir = outputDir
			p.baseImg = "quay.io/example/memcached-operator"
			p.buildCmd = "docker build -f bundle.Dockerfile . -t"

			err := p.run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should error when output directory already exists", func() {
			err := os.Mkdir(outputDir, projutil.DirMode)
			Expect(err).NotTo(HaveOccurred())

			p.outputDir = outputDir
			err = p.run()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("output directory: %s for bundles already exists", p.outputDir)))
		})
	})

	Describe("getSDKStampsAndChannels", func() {
		Describe("getSDKStamps", func() {
			It("should be able to extract SDK stamps from CSV", func() {
				annotations := map[string]string{
					"operators.operatorframework.io/builder":        "operator-sdk-v1.5.0",
					"operators.operatorframework.io/project_layout": "go.kubebuilder.io/v4",
				}

				csv := operatorsv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: annotations,
					},
				}

				bundle := apimanifests.Bundle{
					CSV: &csv,
				}

				stamps, err := getSDKStamps(&bundle)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(stamps)).To(BeEquivalentTo(2))
				Expect(reflect.DeepEqual(annotations, stamps)).To(BeTrue())

			})

			It("should error when bundle is empty", func() {
				stamps, err := getSDKStamps(&apimanifests.Bundle{})
				Expect(err).To(HaveOccurred())
				Expect(stamps).To(BeNil())

			})
		})

		Describe("getChannelsByCSV", func() {
			bundle := apimanifests.Bundle{
				CSV: &operatorsv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "memcached-operator:0.0.1",
					},
				},
			}

			defaultChannel := "gamma"

			It("should get the list of channels for corresponding CSV", func() {
				channels := map[string][]string{
					"memcached-operator:0.0.1": {"alpha", "beta"},
				}

				ch := getChannelsByCSV(&bundle, channels, defaultChannel)
				Expect(ch).To(BeEquivalentTo("alpha,beta"))
			})

			It("if no channel is provided, default to candidate", func() {
				channels := map[string][]string{}
				ch := getChannelsByCSV(&bundle, channels, defaultChannel)
				Expect(ch).To(BeEquivalentTo(defaultChannel))
			})
		})
	})

	Describe("getPackageMetadata", func() {
		var (
			pkg apimanifests.PackageManifest
		)

		BeforeEach(func() {
			pkg = apimanifests.PackageManifest{
				PackageName:        "memcached-operator",
				DefaultChannelName: "alpha",
				Channels: []apimanifests.PackageChannel{
					{
						Name:           "alpha",
						CurrentCSVName: "memcached-operator:v1.0.0",
					},
					{
						Name:           "beta",
						CurrentCSVName: "memcached-operator:v1.0.1",
					},
					{
						Name:           "alpha",
						CurrentCSVName: "memcached-operator:v1.0.1",
					},
				},
			}

		})

		It("should return pkgName, channels and channelsByCSV", func() {
			pkgName, defaultChannel, channelByCSV, err := getPackageMetadata(&pkg)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultChannel).To(BeEquivalentTo("alpha"))
			Expect(pkgName).To(BeEquivalentTo("memcached-operator"))
			Expect(len(channelByCSV["memcached-operator:v1.0.0"])).To(BeEquivalentTo(1))
			Expect(len(channelByCSV["memcached-operator:v1.0.1"])).To(BeEquivalentTo(2))
		})

		It("return error when packagename is not found", func() {
			pkg.PackageName = ""
			_, _, _, err := getPackageMetadata(&pkg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot find packagename from the manifest directory"))
		})

		It("should error when defaultChannel name is empty", func() {
			pkg.DefaultChannelName = ""
			_, _, _, err := getPackageMetadata(&pkg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot find the default channel for package"))
		})
	})
})

func getNumberOfDirectories(inputDir string) int {
	count := 0

	dirs, err := os.ReadDir(inputDir)
	Expect(err).NotTo(HaveOccurred())

	for _, d := range dirs {
		if d.IsDir() {
			count++
		}
	}
	return count
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}
