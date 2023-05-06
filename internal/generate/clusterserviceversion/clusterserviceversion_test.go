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

package clusterserviceversion

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	operatorversion "github.com/operator-framework/api/pkg/lib/version"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

var (
	testDataDir           = filepath.Join("..", "testdata")
	csvDir                = filepath.Join(testDataDir, "clusterserviceversions")
	csvBasesDir           = filepath.Join(csvDir, "bases")
	csvNewLayoutBundleDir = filepath.Join(csvDir, "output")

	goTestDataDir              = filepath.Join(testDataDir, "go")
	goStaticDir                = filepath.Join(goTestDataDir, "static")
	goBasicOperatorPath        = filepath.Join(goStaticDir, "basic.operator.yaml")
	goMultiVersionOperatorPath = filepath.Join(goStaticDir, "basic.multiversion.operator.yaml")
)

var (
	col *collector.Manifests
)

var (
	// Base CSVs
	baseCSV, baseCSVUIMeta       *v1alpha1.ClusterServiceVersion
	baseCSVStr, baseCSVUIMetaStr string

	// Updated CSVs
	newCSV, newCSVUIMeta *v1alpha1.ClusterServiceVersion
	newCSVUIMetaStr      string

	// Base Multiversion CSV
	baseMultiVersionCSV *v1alpha1.ClusterServiceVersion

	// Multiversion CSVs
	multiVersionCSVStr string
)

var _ = Describe("Testing CRDs with single version", func() {
	BeforeEach(func() {
		col = &collector.Manifests{}
		collectManifestsFromFileHelper(col, goBasicOperatorPath)
		initTestCSVsHelper()
	})

	var _ = Describe("Generating a ClusterServiceVersion", func() {
		format.TruncatedDiff = true
		format.UseStringerRepresentation = true

		var (
			g            Generator
			buf          *bytes.Buffer
			operatorName = "memcached-operator"
			zeroZeroOne  = "0.0.1"
			zeroZeroTwo  = "0.0.2"
		)

		BeforeEach(func() {
			buf = &bytes.Buffer{}
		})

		Describe("for a Go project", func() {

			Context("with correct Options", func() {

				var (
					tmp string
					err error
				)

				BeforeEach(func() {
					tmp, err = os.MkdirTemp(".", "")
					Expect(err).ToNot(HaveOccurred())
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*baseCSVUIMeta}
				})

				AfterEach(func() {
					if tmp != "" {
						os.RemoveAll(tmp)
					}
					col.ClusterServiceVersions = nil
				})

				It("should write a ClusterServiceVersion manifest to an io.Writer", func() {
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
					opts := []Option{
						WithWriter(buf),
					}
					Expect(g.Generate(opts...)).ToNot(HaveOccurred())
					Expect(buf.String()).To(MatchYAML(newCSVUIMetaStr))
				})
				It("should write a ClusterServiceVersion manifest to a bundle file", func() {
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
					opts := []Option{
						WithBundleWriter(tmp),
					}
					Expect(g.Generate(opts...)).ToNot(HaveOccurred())
					outputFile := filepath.Join(tmp, bundle.ManifestsDir, makeCSVFileName(operatorName))
					Expect(outputFile).To(BeAnExistingFile())
					Expect(readFileHelper(outputFile)).To(MatchYAML(newCSVUIMetaStr))
				})
				It("should write a ClusterServiceVersion manifest to a package file", func() {
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
					opts := []Option{
						WithPackageWriter(tmp),
					}
					Expect(g.Generate(opts...)).ToNot(HaveOccurred())
					outputFile := filepath.Join(tmp, g.Version, makeCSVFileName(operatorName))
					Expect(outputFile).To(BeAnExistingFile())
					Expect(readFileHelper(outputFile)).To(MatchYAML(newCSVUIMetaStr))
				})
			})

			Context("with incorrect Options", func() {

				BeforeEach(func() {
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
				})

				It("should return an error without any Options", func() {
					opts := []Option{}
					Expect(g.Generate(opts...)).To(MatchError(noGetWriterError))
				})
			})

			Context("to create a new ClusterServiceVersion", func() {
				It("should return a default object when no base is supplied", func() {
					col.ClusterServiceVersions = nil
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*(bases.New(operatorName))}
					csvExp, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(csvExp))
				})

				It("should return a default object", func() {
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*baseCSV}
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
					newCSV.Annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(newCSV))
				})
				It("should return an object with '.spec.replaces' and '.metadata.annotations['olm.skipRange']'", func() {
					baseCSVUIMetaIn := baseCSVUIMeta.DeepCopy()
					baseCSVUIMetaIn.GetAnnotations()["olm.skipRange"] = "<0.0.2"
					baseCSVUIMetaIn.Spec.Replaces = "memcached-operator.v0.0.2"
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*baseCSVUIMetaIn}
					g = Generator{
						OperatorName: operatorName,
						Version:      "0.0.3",
						Collector:    col,
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					csvExp := newCSVUIMeta.DeepCopy()
					csvExp.SetName("memcached-operator.v0.0.3")
					csvExp.GetAnnotations()["olm.skipRange"] = "<0.0.2"
					csvExp.Annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)
					csvExp.Spec.Replaces = "memcached-operator.v0.0.2"
					csvExp.Spec.Version.Patch = 3
					Expect(csv).To(Equal(csvExp))
				})
				It("should return a new object with version set", func() {
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*baseCSVUIMeta}
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector:    col,
					}
					newCSVUIMeta.Annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(newCSVUIMeta))
				})
				It("should return a new object with base version and name preserved", func() {
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*newCSVUIMeta}
					g = Generator{
						OperatorName: operatorName,
						Collector:    col,
					}
					newCSVUIMeta.Annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(newCSVUIMeta))
				})
			})

			Context("to update an existing ClusterServiceVersion", func() {
				It("should return an updated object", func() {
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroOne,
						Collector: &collector.Manifests{
							ClusterServiceVersions: []v1alpha1.ClusterServiceVersion{*newCSVUIMeta},
						},
						Annotations: map[string]string{
							"createdAt": time.Now().UTC().Format(time.RFC3339),
						},
					}
					// Update the input's and expected CSV's Deployment image.
					collectManifestsFromFileHelper(g.Collector, goBasicOperatorPath)
					Expect(g.Collector.Deployments).ToNot(BeEmpty())
					imageTag := "controller:v" + g.Version
					modifyDepImageHelper(&g.Collector.Deployments[0].Spec, imageTag)
					updatedCSV := updateCSV(newCSVUIMeta, modifyCSVDepImageHelper(imageTag))

					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(updatedCSV))

					// verify if conversion webhooks are added
					Expect(csv.Spec.WebhookDefinitions).NotTo(BeEmpty())
					Expect(containsConversionWebhookDefinition(csv.Spec.WebhookDefinitions)).To(BeTrue())
				})
			})

			Context("to upgrade an existing ClusterServiceVersion", func() {
				It("should return an upgraded object", func() {
					col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*newCSVUIMeta}
					g = Generator{
						OperatorName: operatorName,
						Version:      zeroZeroTwo,
						Collector:    col,
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(upgradeCSV(newCSVUIMeta, g.OperatorName, g.Version)))
				})
			})
		})
	})

	var _ = Describe("Generation requires interaction", func() {
		var (
			testExistingPath    = filepath.Join(csvBasesDir, "memcached-operator.clusterserviceversion.yaml")
			testNotExistingPath = filepath.Join(csvBasesDir, "notexist.clusterserviceversion.yaml")
		)

		Context("when base path does not exist", func() {
			It("turning interaction off explicitly, it will returns false", func() {
				Expect(requiresInteraction(testNotExistingPath, projutil.InteractiveHardOff)).To(BeFalse())
			})
			It("turning interaction off implicitly, it will returns true", func() {
				Expect(requiresInteraction(testNotExistingPath, projutil.InteractiveSoftOff)).To(BeTrue())
			})
			It("turning interaction on explicitly, it will returns true", func() {
				Expect(requiresInteraction(testNotExistingPath, projutil.InteractiveOnAll)).To(BeTrue())
			})
		})

		Context("when base path does exist", func() {
			It("turning interaction off explicitly, it will returns false", func() {
				Expect(requiresInteraction(testExistingPath, projutil.InteractiveHardOff)).To(BeFalse())
			})
			It("turning interaction off implicitly, it will returns false", func() {
				Expect(requiresInteraction(testExistingPath, projutil.InteractiveSoftOff)).To(BeFalse())
			})
			It("turning interaction on explicitly, it will returns true", func() {
				Expect(requiresInteraction(testExistingPath, projutil.InteractiveOnAll)).To(BeTrue())
			})
		})
	})

})

var _ = Describe("Testing CRDs with multiple version", func() {
	BeforeEach(func() {
		col = &collector.Manifests{}
		collectManifestsFromFileHelper(col, goMultiVersionOperatorPath)
		initTestMultiVersionCSVHelper()
	})

	var _ = Describe("Generating a clusterserviceVersion", func() {
		format.TruncatedDiff = true
		format.UseStringerRepresentation = true

		var (
			g            Generator
			buf          *bytes.Buffer
			operatorName = "memcached-operator"
			zeroZeroOne  = "0.0.1"
		)

		BeforeEach(func() {
			buf = &bytes.Buffer{}
		})

		Describe("for a go project", func() {
			var (
				err error
			)

			BeforeEach(func() {
				Expect(err).ToNot(HaveOccurred())
				col.ClusterServiceVersions = []v1alpha1.ClusterServiceVersion{*baseMultiVersionCSV}
			})

			AfterEach(func() {
				col.ClusterServiceVersions = nil
			})

			// TODO: to verify conversion webhook specifically, this block has just the
			// tests to see if csv is being written cirrectly to io.Writer.
			It("testing clustersevice version being written to io.Writer", func() {
				g = Generator{
					OperatorName: operatorName,
					Version:      zeroZeroOne,
					Collector:    col,
				}
				opts := []Option{
					WithWriter(buf),
				}
				Expect(g.Generate(opts...)).ToNot(HaveOccurred())
				Expect(buf.String()).To(MatchYAML(multiVersionCSVStr))
			})

		})

	})
})

func collectManifestsFromFileHelper(col *collector.Manifests, path string) {
	f, err := os.Open(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, col.UpdateFromReader(f)).ToNot(HaveOccurred())
	ExpectWithOffset(1, f.Close()).Should(Succeed())
}

func initTestCSVsHelper() {
	var err error
	path := filepath.Join(csvBasesDir, "memcached-operator.clusterserviceversion.yaml")
	baseCSV, baseCSVStr, err = getCSVFromFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	path = filepath.Join(csvBasesDir, "with-ui-metadata.clusterserviceversion.yaml")
	baseCSVUIMeta, baseCSVUIMetaStr, err = getCSVFromFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	path = filepath.Join(csvNewLayoutBundleDir, "memcached-operator.clusterserviceversion.yaml")
	newCSV, _, err = getCSVFromFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	path = filepath.Join(csvNewLayoutBundleDir, "with-ui-metadata.clusterserviceversion.yaml")
	newCSVUIMeta, newCSVUIMetaStr, err = getCSVFromFile(path)
	newCSVUIMetaStr = strings.ReplaceAll(newCSVUIMetaStr, "2022-11-08T16:44:38Z", time.Now().UTC().Format(time.RFC3339))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func initTestMultiVersionCSVHelper() {
	var err error
	path := filepath.Join(csvBasesDir, "memcached-operator-multiVersion.yaml")
	baseMultiVersionCSV, _, err = getCSVFromFile(path)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	path = filepath.Join(csvNewLayoutBundleDir, "memcached-operator-multiVersion.yaml")
	_, multiVersionCSVStr, err = getCSVFromFile(path)
	multiVersionCSVStr = strings.ReplaceAll(multiVersionCSVStr, "2022-11-08T16:44:38Z", time.Now().UTC().Format(time.RFC3339))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func readFileHelper(path string) string {
	b, err := os.ReadFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return string(b)
}

func modifyCSVDepImageHelper(tag string) func(csv *v1alpha1.ClusterServiceVersion) {
	return func(csv *v1alpha1.ClusterServiceVersion) {
		depSpecs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
		ExpectWithOffset(2, depSpecs).ToNot(BeEmpty())
		modifyDepImageHelper(&depSpecs[0].Spec, tag)
	}
}

func modifyDepImageHelper(depSpec *appsv1.DeploymentSpec, tag string) {
	containers := depSpec.Template.Spec.Containers
	ExpectWithOffset(1, containers).ToNot(BeEmpty())
	containers[0].Image = tag
}

func getCSVFromFile(path string) (*v1alpha1.ClusterServiceVersion, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	csv := &v1alpha1.ClusterServiceVersion{}
	if err = yaml.Unmarshal(b, csv); err == nil {
		// Any updates applied to a CSV object will create non-nil slice type fields,
		// which cause comparison issues if their counterpart was only unmarshaled.
		if csv.Spec.InstallStrategy.StrategySpec.Permissions == nil {
			csv.Spec.InstallStrategy.StrategySpec.Permissions = []v1alpha1.StrategyDeploymentPermissions{}
		}
		if csv.Spec.InstallStrategy.StrategySpec.ClusterPermissions == nil {
			csv.Spec.InstallStrategy.StrategySpec.ClusterPermissions = []v1alpha1.StrategyDeploymentPermissions{}
		}
		if csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs == nil {
			csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs = []v1alpha1.StrategyDeploymentSpec{}
		}
		if csv.Spec.WebhookDefinitions == nil {
			csv.Spec.WebhookDefinitions = []v1alpha1.WebhookDescription{}
		}
	}
	return csv, string(b), err
}

func updateCSV(csv *v1alpha1.ClusterServiceVersion,
	opts ...func(*v1alpha1.ClusterServiceVersion)) *v1alpha1.ClusterServiceVersion {

	updated := csv.DeepCopy()
	updated.Annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)
	for _, opt := range opts {
		opt(updated)
	}
	return updated
}

func upgradeCSV(csv *v1alpha1.ClusterServiceVersion, name, version string) *v1alpha1.ClusterServiceVersion {
	upgraded := csv.DeepCopy()

	// Update CSV name and upgrade version.
	upgraded.SetName(genutil.MakeCSVName(name, version))
	upgraded.Spec.Version = operatorversion.OperatorVersion{Version: semver.MustParse(version)}
	upgraded.Annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)

	return upgraded
}

func containsConversionWebhookDefinition(whdef []v1alpha1.WebhookDescription) bool {
	for _, def := range whdef {
		if def.Type == v1alpha1.ConversionWebhook {
			return true
		}
	}
	return false
}
