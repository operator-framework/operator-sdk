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
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blang/semver"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	// OLMCatalogDir is the named directory for OLM catalog manifests.
	OLMCatalogDir = "olm-catalog"

	csvYamlFileExt = ".clusterserviceversion.yaml"
)

type Generator struct {
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	//
	OperatorType projutil.OperatorType
	// Version is the CSV current version.
	Version string
	//
	FromVersion string
	//
	Collector *collector.Manifests

	//
	config *config.Config
	//
	getBase getBaseFunc
	//
	getWriter func() (io.Writer, error)
}

type getBaseFunc func() (*operatorsv1alpha1.ClusterServiceVersion, error)

type Option func(*Generator) error

func WithBase(inputDir, apisDir string) Option {
	return func(g *Generator) error {
		g.getBase = g.makeBaseGetterKustomize(inputDir, apisDir)
		return nil
	}
}

func WithWriter(w io.Writer) Option {
	return func(g *Generator) error {
		g.getWriter = func() (io.Writer, error) {
			return w, nil
		}
		return nil
	}
}

func WithBaseWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := getCSVFile(g.OperatorName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, "bases"), fileName)
		}
		return nil
	}
}

func WithBundleWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := getCSVFile(g.OperatorName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, bundle.ManifestsDir), fileName)
		}
		return nil
	}
}

func WithPackageWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := getCSVFile(g.OperatorName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, g.Version), fileName)
		}
		return nil
	}
}

func (g *Generator) Generate(cfg *config.Config, opts ...Option) (err error) {
	g.config = cfg
	for _, opt := range opts {
		if err = opt(g); err != nil {
			return err
		}
	}

	return g.generate()
}

type LegacyOption Option

func WithPackageWriterLegacy(dir string) LegacyOption {
	return func(g *Generator) error {
		fileName := getCSVFileLegacy(g.OperatorName, g.Version)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, g.Version), fileName)
		}
		return nil
	}
}

func WithBundleBase(inputDir, apisDir string) LegacyOption {
	return func(g *Generator) error {
		g.getBase = g.makeBaseGetterBundleLegacy(inputDir, apisDir)
		return nil
	}
}

func WithPackageBase(inputDir, apisDir string) LegacyOption {
	return func(g *Generator) error {
		g.getBase = g.makeBaseGetterPackageLegacy(inputDir, apisDir)
		return nil
	}
}

func (g *Generator) GenerateLegacy(opts ...LegacyOption) (err error) {
	for _, opt := range opts {
		if err = opt(g); err != nil {
			return err
		}
	}

	return g.generate()
}

func (g *Generator) generate() (err error) {
	if g.getBase == nil {
		return genutil.InternalError("getBase must be set")
	}
	if g.getWriter == nil {
		return genutil.InternalError("getWriter must be set")
	}

	base, err := g.getBase()
	if err != nil {
		return fmt.Errorf("error getting ClusterServiceVersion base: %v", err)
	}

	if err = g.updateVersions(base); err != nil {
		return err
	}

	if g.Collector != nil {
		if err := applyTo(g.Collector, base); err != nil {
			return err
		}
	}

	w, err := g.getWriter()
	if err != nil {
		return err
	}
	return genutil.WriteObject(w, base)
}

func getCSVFile(name string) string {
	return strings.ToLower(name) + csvYamlFileExt
}

func getCSVFileLegacy(name, version string) string {
	return fmt.Sprintf("%s.v%s%s", strings.ToLower(name), version, csvYamlFileExt)
}

func (g Generator) makeBaseGetterKustomize(inputDir, apisDir string) getBaseFunc {
	basePath := filepath.Join(inputDir, "bases", getCSVFile(g.OperatorName))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}

	return g.makeBaseGetter(basePath, apisDir)
}

func (g Generator) makeBaseGetter(basePath, apisDir string) getBaseFunc {
	gvks := make([]schema.GroupVersionKind, len(g.config.Resources))
	for i, gvk := range g.config.Resources {
		gvks[i].Group = fmt.Sprintf("%s.%s", gvk.Group, g.config.Domain)
		gvks[i].Version = gvk.Version
		gvks[i].Kind = gvk.Kind
	}

	return func() (*operatorsv1alpha1.ClusterServiceVersion, error) {
		b := bases.ClusterServiceVersion{
			OperatorName: g.OperatorName,
			OperatorType: g.OperatorType,
			BasePath:     basePath,
			APIsDir:      apisDir,
			GVKs:         gvks,
		}
		return b.GetBase()
	}
}

func (g Generator) makeBaseGetterBundleLegacy(inputDir, apisDir string) getBaseFunc {
	basePath := filepath.Join(inputDir, bundle.ManifestsDir, getCSVFile(g.OperatorName))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}
	return g.makeBaseGetterLegacy(basePath, apisDir)
}

func (g Generator) makeBaseGetterPackageLegacy(inputDir, apisDir string) getBaseFunc {
	version := g.FromVersion
	if version == "" {
		version = g.Version
	}
	basePath := filepath.Join(inputDir, version, getCSVFileLegacy(g.OperatorName, version))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}
	return g.makeBaseGetterLegacy(basePath, apisDir)
}

func (g Generator) makeBaseGetterLegacy(basePath, apisDir string) getBaseFunc {
	var gvks []schema.GroupVersionKind
	if g.Collector != nil {
		for _, crd := range g.Collector.CustomResourceDefinitions {
			for _, version := range crd.Spec.Versions {
				gvks = append(gvks, schema.GroupVersionKind{
					Group:   crd.Spec.Group,
					Version: version.Name,
					Kind:    crd.Spec.Names.Kind,
				})
			}
		}
	}

	return func() (*operatorsv1alpha1.ClusterServiceVersion, error) {
		b := bases.ClusterServiceVersion{
			OperatorName: g.OperatorName,
			OperatorType: g.OperatorType,
			BasePath:     basePath,
			APIsDir:      apisDir,
			GVKs:         gvks,
		}
		return b.GetBase()
	}
}

// updateVersions updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (g Generator) updateVersions(csv *operatorsv1alpha1.ClusterServiceVersion) (err error) {

	oldVer, newVer := csv.Spec.Version.String(), g.Version
	newCSVName := genutil.GetCSVName(g.OperatorName, newVer)
	oldCSVName := genutil.GetCSVName(g.OperatorName, oldVer)

	// If the new version is empty, either because a CSV is only being updated or
	// a base was generated, no update is needed.
	if newVer == "0.0.0" || newVer == "" || newVer == oldVer {
		return nil
	}

	if oldVer != "0.0.0" {
		// Replace all references to the old operator name.
		oldRe, err := regexp.Compile(fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(oldCSVName)))
		if err != nil {
			return fmt.Errorf("error compiling CSV name regexp %s: %v", oldRe, err)
		}
		b, err := yaml.Marshal(csv)
		if err != nil {
			return err
		}
		b = oldRe.ReplaceAll(b, []byte(newCSVName))
		*csv = operatorsv1alpha1.ClusterServiceVersion{}
		if err = yaml.Unmarshal(b, csv); err != nil {
			return fmt.Errorf("error unmarshalling CSV %s after replacing old CSV name: %v", csv.GetName(), err)
		}
		csv.Spec.Replaces = oldCSVName
	}

	csv.SetName(newCSVName)
	csv.Spec.Version.Version, err = semver.Parse(newVer)
	return err
}
