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
	"strings"

	"github.com/blang/semver"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/pkg/model/config"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	// File extension for all ClusterServiceVersion manifests written by Generator.
	csvYamlFileExt = ".clusterserviceversion.yaml"
)

var (
	// Internal errors.
	noGetBaseError               = genutil.InternalError("getBase must be set")
	noGetWriterError             = genutil.InternalError("getWriter must be set")
	baseVersionNotAllowedError   = genutil.InternalError("cannot set version when generating a base")
	baseCollectorNotAllowedError = genutil.InternalError("cannot set collector when generating a base")
)

// ClusterServiceVersion configures ClusterServiceVersion manifest generation.
type Generator struct {
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	// OperatorType determines what code API types are written in for getBase.
	OperatorType projutil.OperatorType
	// Version is the CSV current version.
	Version string
	// Collector holds all manifests relevant to the Generator.
	Collector *collector.Manifests

	// Project configuration.
	config *config.Config
	// Func that returns a base CSV.
	getBase getBaseFunc
	// Func that returns the writer the generated CSV's bytes are written to.
	getWriter func() (io.Writer, error)
	// If the CSV is destined for a bundle this will be the path of the updated
	// CSV. Used to bring over data from an existing CSV that is not captured
	// in a base. Not set if a non-file or base writer is returned by getWriter.
	bundledPath string
}

// Type of Generator.getBase.
type getBaseFunc func() (*operatorsv1alpha1.ClusterServiceVersion, error)

// Option is a function that modifies a Generator.
type Option func(*Generator) error

// WithBase sets a Generator's base CSV to a kustomize-style base.
func WithBase(inputDir, apisDir string, ilvl projutil.InteractiveLevel) Option {
	return func(g *Generator) error {
		g.getBase = g.makeKustomizeBaseGetter(inputDir, apisDir, ilvl)
		return nil
	}
}

// WithWriter sets a Generator's writer to w.
func WithWriter(w io.Writer) Option {
	return func(g *Generator) error {
		g.getWriter = func() (io.Writer, error) {
			return w, nil
		}
		return nil
	}
}

// WithBaseWriter sets a Generator's writer to a kustomize-style base file
// under <dir>/bases.
func WithBaseWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := makeCSVFileName(g.OperatorName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, "bases"), fileName)
		}
		// Bases should not be updated with a version or manifests.
		if g.Version != "" {
			return baseVersionNotAllowedError
		}
		if g.Collector != nil {
			return baseCollectorNotAllowedError
		}
		return nil
	}
}

// WithBundleWriter sets a Generator's writer to a bundle CSV file under
// <dir>/manifests.
func WithBundleWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := makeCSVFileName(g.OperatorName)
		g.bundledPath = filepath.Join(dir, bundle.ManifestsDir, fileName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, bundle.ManifestsDir), fileName)
		}
		return nil
	}
}

// Generate configures the generator with cfg and opts then runs it.
func (g *Generator) Generate(cfg *config.Config, opts ...Option) (err error) {
	g.config = cfg
	for _, opt := range opts {
		if err = opt(g); err != nil {
			return err
		}
	}

	if g.getWriter == nil {
		return noGetWriterError
	}

	csv, err := g.generate()
	if err != nil {
		return err
	}

	w, err := g.getWriter()
	if err != nil {
		return err
	}
	return genutil.WriteObject(w, csv)
}

// LegacyOption is a function that modifies a Generator for legacy project layouts.
type LegacyOption Option

// WithBundleBase sets a Generator's base CSV to a legacy-style bundle base.
func WithBundleBase(inputDir, apisDir string, ilvl projutil.InteractiveLevel) LegacyOption {
	return func(g *Generator) error {
		g.getBase = g.makeBundleBaseGetterLegacy(inputDir, apisDir, ilvl)
		return nil
	}
}

// GenerateLegacy configures the generator with opts then runs it. Used for
// generating files for legacy project layouts.
func (g *Generator) GenerateLegacy(opts ...LegacyOption) (err error) {
	for _, opt := range opts {
		if err = opt(g); err != nil {
			return err
		}
	}

	if g.getWriter == nil {
		return noGetWriterError
	}

	csv, err := g.generate()
	if err != nil {
		return err
	}

	w, err := g.getWriter()
	if err != nil {
		return err
	}
	return genutil.WriteObject(w, csv)
}

// generate runs a configured Generator.
func (g *Generator) generate() (*operatorsv1alpha1.ClusterServiceVersion, error) {
	if g.getBase == nil {
		return nil, noGetBaseError
	}

	base, err := g.getBase()
	if err != nil {
		return nil, fmt.Errorf("error getting ClusterServiceVersion base: %v", err)
	}

	if err = g.updateVersions(base); err != nil {
		return nil, err
	}

	if g.Collector != nil {
		if err := ApplyTo(g.Collector, base); err != nil {
			return nil, err
		}
	}

	return base, nil
}

// makeCSVFileName returns a CSV file name containing name.
func makeCSVFileName(name string) string {
	return strings.ToLower(name) + csvYamlFileExt
}

// makeKustomizeBaseGetter returns a function that gets a kustomize-style base.
func (g Generator) makeKustomizeBaseGetter(inputDir, apisDir string, ilvl projutil.InteractiveLevel) getBaseFunc {
	basePath := filepath.Join(inputDir, "bases", makeCSVFileName(g.OperatorName))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}

	return g.makeBaseGetter(basePath, apisDir, requiresInteraction(basePath, ilvl))
}

// makeBaseGetter returns a function that gets a base from inputDir.
// apisDir is used by getBaseFunc to populate base fields.
func (g Generator) makeBaseGetter(basePath, apisDir string, interactive bool) getBaseFunc {
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
			Interactive:  interactive,
		}
		return b.GetBase()
	}
}

// makeBundleBaseGetterLegacy returns a function that gets a bundle base
// for legacy project layouts.
func (g Generator) makeBundleBaseGetterLegacy(inputDir, apisDir string, ilvl projutil.InteractiveLevel) getBaseFunc {
	basePath := filepath.Join(inputDir, bundle.ManifestsDir, makeCSVFileName(g.OperatorName))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}
	return g.makeBaseGetterLegacy(basePath, apisDir, requiresInteraction(basePath, ilvl))
}

// makeBaseGetterLegacy returns a function that gets a base from inputDir.
// apisDir is used by getBaseFunc to populate base fields. This method should
// be used when creating LegacyOptions.
func (g Generator) makeBaseGetterLegacy(basePath, apisDir string, interactive bool) getBaseFunc {
	var gvks []schema.GroupVersionKind
	if g.Collector != nil {
		v1crdGVKs := k8sutil.GVKsForV1CustomResourceDefinitions(g.Collector.V1CustomResourceDefinitions...)
		gvks = append(gvks, v1crdGVKs...)
		v1beta1crdGVKs := k8sutil.GVKsForV1beta1CustomResourceDefinitions(g.Collector.V1beta1CustomResourceDefinitions...)
		gvks = append(gvks, v1beta1crdGVKs...)
	}

	return func() (*operatorsv1alpha1.ClusterServiceVersion, error) {
		b := bases.ClusterServiceVersion{
			OperatorName: g.OperatorName,
			OperatorType: g.OperatorType,
			BasePath:     basePath,
			APIsDir:      apisDir,
			GVKs:         gvks,
			Interactive:  interactive,
		}
		return b.GetBase()
	}
}

// requiresInteraction checks if the combination of ilvl and basePath existence
// requires the generator prompt a user interactively.
func requiresInteraction(basePath string, ilvl projutil.InteractiveLevel) bool {
	return (ilvl == projutil.InteractiveSoftOff && genutil.IsNotExist(basePath)) || ilvl == projutil.InteractiveOnAll
}

// updateVersions updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (g Generator) updateVersions(csv *operatorsv1alpha1.ClusterServiceVersion) (err error) {

	oldVer, newVer := csv.Spec.Version.String(), g.Version
	newName := genutil.MakeCSVName(g.OperatorName, newVer)
	oldName := csv.GetName()

	// A bundled CSV may not have a base containing the previous version to use,
	// so use the current bundled CSV for version information.
	if genutil.IsExist(g.bundledPath) {
		existing, err := (bases.ClusterServiceVersion{BasePath: g.bundledPath}).GetBase()
		if err != nil {
			return fmt.Errorf("error reading existing ClusterServiceVersion: %v", err)
		}
		oldVer = existing.Spec.Version.String()
		oldName = existing.GetName()
	}

	// If the new version is empty, either because a CSV is only being updated or
	// a base was generated, no update is needed.
	if newVer == "0.0.0" || newVer == "" {
		return nil
	}

	// Set replaces by default.
	// TODO: consider all possible CSV versioning schemes supported  by OLM.
	if oldVer != "0.0.0" && newVer != oldVer {
		csv.Spec.Replaces = oldName
	}

	csv.SetName(newName)
	csv.Spec.Version.Version, err = semver.Parse(newVer)
	return err
}
