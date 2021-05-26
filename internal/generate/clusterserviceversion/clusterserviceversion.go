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

	"github.com/blang/semver/v4"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	// File extension for all ClusterServiceVersion manifests written by Generator.
	csvYamlFileExt = ".clusterserviceversion.yaml"
)

var (
	// Internal errors.
	noGetWriterError = genutil.InternalError("getWriter must be set")
)

// ClusterServiceVersion configures ClusterServiceVersion manifest generation.
type Generator struct {
	// OperatorName is the operator's name, ex. app-operator.
	OperatorName string
	// Version is the CSV current version.
	Version string
	// FromVersion is the version of a previous CSV to upgrade from.
	FromVersion string
	// Collector holds all manifests relevant to the Generator.
	Collector *collector.Manifests
	// Annotations are applied to the resulting CSV.
	Annotations map[string]string
	// ExtraServiceAccounts are ServiceAccount names to consider when matching
	// {Cluster}Roles to include in a CSV via their Bindings.
	ExtraServiceAccounts []string

	// Func that returns the writer the generated CSV's bytes are written to.
	getWriter func() (io.Writer, error)
}

// Option is a function that modifies a Generator.
type Option func(*Generator) error

// WithWriter sets a Generator's writer to w.
func WithWriter(w io.Writer) Option {
	return func(g *Generator) error {
		g.getWriter = func() (io.Writer, error) {
			return w, nil
		}
		return nil
	}
}

// WithBundleWriter sets a Generator's writer to a bundle CSV file under
// <dir>/manifests.
func WithBundleWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := makeCSVFileName(g.OperatorName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, bundle.ManifestsDir), fileName)
		}
		return nil
	}
}

// WithPackageWriter sets a Generator's writer to a package CSV file under
// <dir>/<version>.
func WithPackageWriter(dir string) Option {
	return func(g *Generator) error {
		fileName := makeCSVFileName(g.OperatorName)
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(filepath.Join(dir, g.Version), fileName)
		}
		return nil
	}
}

// Generate configures the generator with col and opts then runs it.
func (g *Generator) Generate(opts ...Option) (err error) {
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

	// Add extra annotations to csv
	g.setAnnotations(csv)

	w, err := g.getWriter()
	if err != nil {
		return err
	}
	return genutil.WriteObject(w, csv)
}

// setSDKAnnotations adds SDK metric labels to the base if they do not exist.
func (g Generator) setAnnotations(csv *v1alpha1.ClusterServiceVersion) {
	annotations := csv.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	for k, v := range g.Annotations {
		annotations[k] = v
	}
	csv.SetAnnotations(annotations)
}

// generate runs a configured Generator.
func (g *Generator) generate() (base *operatorsv1alpha1.ClusterServiceVersion, err error) {
	if g.Collector == nil {
		return nil, fmt.Errorf("cannot generate CSV without a manifests collection")
	}

	// Search for a CSV in the collector with a name matching the package name.
	csvNamePrefix := g.OperatorName + "."
	for _, csv := range g.Collector.ClusterServiceVersions {
		if base == nil && strings.HasPrefix(csv.GetName(), csvNamePrefix) {
			base = csv.DeepCopy()
		}
	}

	// Use a default base if none was supplied.
	if base == nil {
		base = bases.New(g.OperatorName)
	}
	if g.Version != "" {
		// Use the existing version/name unless g.Version is set.
		base.SetName(genutil.MakeCSVName(g.OperatorName, g.Version))
		if base.Spec.Version.Version, err = semver.Parse(g.Version); err != nil {
			return nil, err
		}
	}
	if g.FromVersion != "" {
		base.Spec.Replaces = genutil.MakeCSVName(g.OperatorName, g.FromVersion)
	}

	if err := ApplyTo(g.Collector, base, g.ExtraServiceAccounts); err != nil {
		return nil, err
	}

	return base, nil
}

// makeCSVFileName returns a CSV file name containing name.
func makeCSVFileName(name string) string {
	return strings.ToLower(name) + csvYamlFileExt
}

// requiresInteraction checks if the combination of ilvl and basePath existence
// requires the generator prompt a user interactively.
func requiresInteraction(basePath string, ilvl projutil.InteractiveLevel) bool {
	return (ilvl == projutil.InteractiveSoftOff && genutil.IsNotExist(basePath)) || ilvl == projutil.InteractiveOnAll
}
