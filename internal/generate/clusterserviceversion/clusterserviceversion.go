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
	OperatorType projutil.OperatorType
	// Version is the CSV current version.
	Version string
	// FromVersion is the version of a previous CSV to upgrade from.
	FromVersion string
	// Collector holds all manifests relevant to the Generator.
	Collector *collector.Manifests
	// Annotations are applied to the resulting CSV.
	Annotations map[string]string

	// Func that returns the writer the generated CSV's bytes are written to.
	getWriter func() (io.Writer, error)
	// If the CSV is destined for a bundle this will be the path of the updated
	// CSV. Used to bring over data from an existing CSV that is not captured
	// in a base. Not set if a non-file or base writer is returned by getWriter.
	bundledPath string
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
		g.bundledPath = filepath.Join(dir, bundle.ManifestsDir, fileName)
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
		if g.FromVersion != "" {
			g.bundledPath = filepath.Join(dir, g.FromVersion, fileName)
		}
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

	// Search for a CSV in the collector with a name matching the package name,
	// but prefer an exact match "<package name>.vX.Y.Z" to preserve existing behavior.
	var oldBase *operatorsv1alpha1.ClusterServiceVersion
	csvNamePrefix := g.OperatorName + "."
	oldBaseCSVName := genutil.MakeCSVName(g.OperatorName, "X.Y.Z")
	for _, csv := range g.Collector.ClusterServiceVersions {
		if csv.GetName() == oldBaseCSVName {
			oldBase = csv.DeepCopy()
		} else if base == nil && strings.HasPrefix(csv.GetName(), csvNamePrefix) {
			base = csv.DeepCopy()
		}
	}

	if base == nil && oldBase == nil {
		return nil, fmt.Errorf("no CSV found with name prefix %q", csvNamePrefix)
	} else if oldBase != nil {
		// Only update versions in the old way to preserve existing behavior.
		base = oldBase
		if err := g.updateVersionsWithReplaces(base); err != nil {
			return nil, err
		}
	} else if g.Version != "" {
		// Use the existing version/name unless g.Version is set.
		base.SetName(genutil.MakeCSVName(g.OperatorName, g.Version))
		if base.Spec.Version.Version, err = semver.Parse(g.Version); err != nil {
			return nil, err
		}
	}

	if err := ApplyTo(g.Collector, base); err != nil {
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

// updateVersionsWithReplaces updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (g Generator) updateVersionsWithReplaces(csv *operatorsv1alpha1.ClusterServiceVersion) (err error) {

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
	if oldVer != "0.0.0" && newVer != oldVer {
		csv.Spec.Replaces = oldName
	}

	csv.SetName(newName)
	csv.Spec.Version.Version, err = semver.Parse(newVer)
	return err
}
