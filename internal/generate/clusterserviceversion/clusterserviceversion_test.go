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
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	"github.com/blang/semver"
	operatorversion "github.com/operator-framework/api/pkg/lib/version"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

var (
	testDataDir           = filepath.Join("..", "testdata")
	csvDir                = filepath.Join(testDataDir, "clusterserviceversions")
	csvBasesDir           = filepath.Join(csvDir, "bases")
	csvNewLayoutBundleDir = filepath.Join(csvDir, "newlayout", "manifests")

	// TODO: create a new testdata dir (top level?) that has both a "config"
	// dir and a "deploy" dir that contains `kustomize build config/default`
	// output to simulate actual manifest collection behavior. Using "config"
	// directly is not standard behavior.
	goTestDataDir     = filepath.Join(testDataDir, "non-standard-layout")
	goAPIsDir         = filepath.Join(goTestDataDir, "api")
	goManifestRootDir = filepath.Join(goTestDataDir, "config")
	goCRDsDir         = filepath.Join(goManifestRootDir, "crds")
)

var (
	col *collector.Manifests
	cfg *config.Config
)

var (
	baseCSV, baseCSVUIMeta, newCSV          *v1alpha1.ClusterServiceVersion
	baseCSVStr, baseCSVUIMetaStr, newCSVStr string
)

var _ = BeforeSuite(func() {
	col = &collector.Manifests{}
	Expect(col.UpdateFromDirs(goManifestRootDir, goCRDsDir)).ToNot(HaveOccurred())

	cfg = readConfigHelper(goTestDataDir)

	initTestCSVsHelper()
})

var _ = Describe("Generating a ClusterServiceVersion", func() {
	format.TruncatedDiff = true
	format.UseStringerRepresentation = true

	var (
		g            Generator
		buf          *bytes.Buffer
		operatorName = "memcached-operator"
		operatorType = projutil.OperatorTypeGo
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
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if tmp != "" {
					os.RemoveAll(tmp)
				}
			})

			It("should write a ClusterServiceVersion manifest to an io.Writer", func() {
				g = Generator{
					OperatorName: operatorName,
					OperatorType: operatorType,
					Version:      version,
					Collector:    col,
				}
				opts := []Option{
					WithBase(csvBasesDir, goAPIsDir, projutil.InteractiveHardOff),
					WithWriter(buf),
				}
				Expect(g.Generate(cfg, opts...)).ToNot(HaveOccurred())
				Expect(buf.String()).To(MatchYAML(newCSVStr))
			})
			It("should write a ClusterServiceVersion manifest to a base file", func() {
				g = Generator{
					OperatorName: operatorName,
					OperatorType: operatorType,
				}
				opts := []Option{
					WithBase(csvBasesDir, goAPIsDir, projutil.InteractiveHardOff),
					WithBaseWriter(tmp),
				}
				Expect(g.Generate(cfg, opts...)).ToNot(HaveOccurred())
				outputFile := filepath.Join(tmp, "bases", makeCSVFileName(operatorName))
				Expect(outputFile).To(BeAnExistingFile())
				Expect(string(readFileHelper(outputFile))).To(MatchYAML(baseCSVUIMetaStr))
			})
			It("should write a ClusterServiceVersion manifest to a bundle file", func() {
				g = Generator{
					OperatorName: operatorName,
					OperatorType: operatorType,
					Version:      version,
					Collector:    col,
				}
				opts := []Option{
					WithBase(csvBasesDir, goAPIsDir, projutil.InteractiveHardOff),
					WithBundleWriter(tmp),
				}
				Expect(g.Generate(cfg, opts...)).ToNot(HaveOccurred())
				outputFile := filepath.Join(tmp, bundle.ManifestsDir, makeCSVFileName(operatorName))
				Expect(outputFile).To(BeAnExistingFile())
				Expect(string(readFileHelper(outputFile))).To(MatchYAML(newCSVStr))
			})

			It("should write a ClusterServiceVersion manifest to a legacy base/bundle file", func() {
				g = Generator{
					OperatorName: operatorName,
					OperatorType: operatorType,
					Version:      version,
					Collector:    col,
				}
				opts := []LegacyOption{
					WithBundleBase(csvBasesDir, goAPIsDir, projutil.InteractiveHardOff),
					LegacyOption(WithBundleWriter(tmp)),
				}
				Expect(g.GenerateLegacy(opts...)).ToNot(HaveOccurred())
				outputFile := filepath.Join(tmp, bundle.ManifestsDir, makeCSVFileName(operatorName))
				Expect(outputFile).To(BeAnExistingFile())
				Expect(string(readFileHelper(outputFile))).To(MatchYAML(newCSVStr))
			})
		})

		Context("with incorrect Options", func() {

			BeforeEach(func() {
				g = Generator{
					OperatorName: operatorName,
					OperatorType: operatorType,
					Version:      version,
					Collector:    col,
				}
			})

			It("should return an error without any Options", func() {
				opts := []Option{}
				Expect(g.Generate(cfg, opts...)).To(MatchError(noGetWriterError))
			})
			It("should return an error without a getWriter", func() {
				opts := []Option{
					WithBase(csvBasesDir, goAPIsDir, projutil.InteractiveHardOff),
				}
				Expect(g.Generate(cfg, opts...)).To(MatchError(noGetWriterError))
			})
			It("should return an error without a getBase", func() {
				opts := []Option{
					WithWriter(&bytes.Buffer{}),
				}
				Expect(g.Generate(cfg, opts...)).To(MatchError(noGetBaseError))
			})

			It("should return an error without any LegacyOptions", func() {
				opts := []LegacyOption{}
				Expect(g.GenerateLegacy(opts...)).To(MatchError(noGetWriterError))
			})
			It("should return an error without a getWriter (legacy)", func() {
				opts := []LegacyOption{
					WithBundleBase(csvBasesDir, goAPIsDir, projutil.InteractiveHardOff),
				}
				Expect(g.GenerateLegacy(opts...)).To(MatchError(noGetWriterError))
			})
			It("should return an error without a getBase (legacy)", func() {
				opts := []LegacyOption{
					LegacyOption(WithWriter(&bytes.Buffer{})),
				}
				Expect(g.GenerateLegacy(opts...)).To(MatchError(noGetBaseError))
			})
		})

		Context("to create a new", func() {

			Context("bundle base", func() {
				It("should return the default base object", func() {
					g = Generator{
						OperatorName: operatorName,
						OperatorType: operatorType,
						config:       cfg,
						getBase:      makeBaseGetter(baseCSV),
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(baseCSV))
				})
				It("should return a base object with customresourcedefinitions", func() {
					g = Generator{
						OperatorName: operatorName,
						OperatorType: operatorType,
						config:       cfg,
						getBase:      makeBaseGetter(baseCSVUIMeta),
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(baseCSVUIMeta))
				})
			})

			Context("bundle", func() {
				It("should return the expected object", func() {
					g = Generator{
						OperatorName: operatorName,
						OperatorType: operatorType,
						Version:      version,
						Collector:    col,
						config:       cfg,
						getBase:      makeBaseGetter(baseCSVUIMeta),
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(newCSV))
				})
			})

		})

		Context("to update an existing", func() {
			Context("bundle", func() {
				It("should return the expected object", func() {
					g = Generator{
						OperatorName: operatorName,
						OperatorType: operatorType,
						Version:      version,
						Collector:    &collector.Manifests{},
						config:       cfg,
						getBase:      makeBaseGetter(newCSV),
					}
					// Update the input's and expected CSV's Deployment image.
					Expect(g.Collector.UpdateFromDirs(goManifestRootDir, goCRDsDir)).ToNot(HaveOccurred())
					Expect(len(g.Collector.Deployments)).To(BeNumerically(">=", 1))
					imageTag := "controller:v" + g.Version
					modifyDepImageHelper(&g.Collector.Deployments[0].Spec, imageTag)
					updatedCSV := updateCSV(newCSV, modifyCSVDepImageHelper(imageTag))

					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(updatedCSV))
				})
			})

		})

		Context("to upgrade an existing", func() {

			Context("bundle", func() {
				It("should return the expected manifest", func() {
					g = Generator{
						OperatorName: operatorName,
						OperatorType: operatorType,
						Version:      "0.0.2",
						Collector:    col,
						config:       cfg,
						getBase:      makeBaseGetter(newCSV),
						// Bundles need a path, usually set by an Option, to an existing
						// CSV manifest so "replaces" can be set correctly.
						bundledPath: filepath.Join(csvNewLayoutBundleDir, "memcached-operator.clusterserviceversion.yaml"),
					}
					csv, err := g.generate()
					Expect(err).ToNot(HaveOccurred())
					Expect(csv).To(Equal(upgradeCSV(newCSV, g.OperatorName, g.Version)))
				})
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
		By("turning interaction off explicitly")
		It("returns false", func() {
			Expect(requiresInteraction(testNotExistingPath, projutil.InteractiveHardOff)).To(BeFalse())
		})
		By("turning interaction off implicitly")
		It("returns true", func() {
			Expect(requiresInteraction(testNotExistingPath, projutil.InteractiveSoftOff)).To(BeTrue())
		})
		By("turning interaction on explicitly")
		It("returns true", func() {
			Expect(requiresInteraction(testNotExistingPath, projutil.InteractiveOnAll)).To(BeTrue())
		})
	})

	Context("when base path does exist", func() {
		By("turning interaction off explicitly")
		It("returns false", func() {
			Expect(requiresInteraction(testExistingPath, projutil.InteractiveHardOff)).To(BeFalse())
		})
		By("turning interaction off implicitly")
		It("returns false", func() {
			Expect(requiresInteraction(testExistingPath, projutil.InteractiveSoftOff)).To(BeFalse())
		})
		By("turning interaction on explicitly")
		It("returns true", func() {
			Expect(requiresInteraction(testExistingPath, projutil.InteractiveOnAll)).To(BeTrue())
		})
	})
})

func readConfigHelper(dir string) *config.Config {
	wd, err := os.Getwd()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, os.Chdir(dir)).ToNot(HaveOccurred())
	cfg, err := kbutil.ReadConfig()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, os.Chdir(wd)).ToNot(HaveOccurred())
	return cfg
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
	newCSV, newCSVStr, err = getCSVFromFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func readFileHelper(path string) []byte {
	b, err := ioutil.ReadFile(path)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return b
}

func modifyCSVDepImageHelper(tag string) func(csv *v1alpha1.ClusterServiceVersion) {
	return func(csv *v1alpha1.ClusterServiceVersion) {
		depSpecs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
		ExpectWithOffset(2, len(depSpecs)).To(BeNumerically(">=", 1))
		modifyDepImageHelper(&depSpecs[0].Spec, tag)
	}
}

func modifyDepImageHelper(depSpec *appsv1.DeploymentSpec, tag string) {
	containers := depSpec.Template.Spec.Containers
	ExpectWithOffset(1, len(containers)).To(BeNumerically(">=", 1))
	containers[0].Image = tag
}

func makeBaseGetter(csv *v1alpha1.ClusterServiceVersion) getBaseFunc {
	return func() (*v1alpha1.ClusterServiceVersion, error) {
		return csv.DeepCopy(), nil
	}
}

func getCSVFromFile(path string) (*v1alpha1.ClusterServiceVersion, string, error) {
	b, err := ioutil.ReadFile(path)
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
	for _, opt := range opts {
		opt(updated)
	}
	return updated
}

func upgradeCSV(csv *v1alpha1.ClusterServiceVersion, name, version string) *v1alpha1.ClusterServiceVersion {
	upgraded := csv.DeepCopy()

	// Update CSV name and upgrade version, then add "replaces" for the old CSV name.
	oldName := upgraded.GetName()
	upgraded.SetName(genutil.MakeCSVName(name, version))
	upgraded.Spec.Version = operatorversion.OperatorVersion{Version: semver.MustParse(version)}
	upgraded.Spec.Replaces = oldName

	return upgraded
}
