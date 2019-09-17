// Copyright 2019 The Operator-SDK Authors
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

package registry

import (
	"encoding/json"

	"github.com/blang/semver"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// manifestsLoad loads a manifests directory from disk.
type manifestsLoad struct {
	dir     string
	pkg     registry.PackageManifest
	bundles map[string]*registry.Bundle
}

// Ensure manifestsLoad implements registry.Load.
var _ registry.Load = &manifestsLoad{}

// populate uses operator-registry's sqlite.NewSQLLoaderForDirectory to load
// l.dir's manifests. Note that this method does not call any functions that
// use SQL drivers.
func (l *manifestsLoad) populate() error {
	dl := sqlite.NewSQLLoaderForDirectory(l, l.dir)
	if err := dl.Populate(); err != nil {
		return errors.Wrapf(err, "error getting bundles from manifests dir %q", l.dir)
	}
	return nil
}

// AddOperatorBundle adds a bundle to l.
func (l *manifestsLoad) AddOperatorBundle(bundle *registry.Bundle) error {
	csvRaw, err := bundle.ClusterServiceVersion()
	if err != nil {
		return errors.Wrap(err, "error getting bundle CSV")
	}
	csvSpec := olmapiv1alpha1.ClusterServiceVersionSpec{}
	if err := json.Unmarshal(csvRaw.Spec, &csvSpec); err != nil {
		return errors.Wrap(err, "error unmarshaling CSV spec")
	}
	l.bundles[csvSpec.Version.String()] = bundle
	return nil
}

// AddOperatorBundle adds the package manifest to l.
func (l *manifestsLoad) AddPackageChannels(pkg registry.PackageManifest) error {
	l.pkg = pkg
	return nil
}

// ManifestsStore knows how to query for an operator's package manifest and
// related bundles.
type ManifestsStore interface {
	// GetPackageManifest returns the ManifestsStore's registry.PackageManifest.
	// The returned object is assumed to be valid.
	GetPackageManifest() registry.PackageManifest
	// GetBundles returns the ManifestsStore's set of registry.Bundle. These
	// bundles are unique by CSV version, since only one operator type should
	// exist in one manifests dir.
	// The returned objects are assumed to be valid.
	GetBundles() []*registry.Bundle
	// GetBundleForVersion returns the ManifestsStore's registry.Bundle for a
	// given version string. An error should be returned if the passed version
	// does not exist in the store.
	// The returned object is assumed to be valid.
	GetBundleForVersion(string) (*registry.Bundle, error)
}

// manifests implements ManifestsStore
type manifests struct {
	pkg     registry.PackageManifest
	bundles map[string]*registry.Bundle
}

// ManifestsStoreForDir populates a ManifestsStore from the metadata in dir.
// Each bundle and the package manifest are statically validated, and will
// return an error if any are not valid.
func ManifestsStoreForDir(dir string) (ManifestsStore, error) {
	l := &manifestsLoad{
		dir:     dir,
		bundles: map[string]*registry.Bundle{},
	}
	if err := l.populate(); err != nil {
		return nil, err
	}
	if err := ValidatePackageManifest(&l.pkg); err != nil {
		return nil, errors.Wrap(err, "error validating package manifest")
	}
	for _, bundle := range l.bundles {
		if err := validateBundle(bundle); err != nil {
			return nil, errors.Wrap(err, "error validating bundle")
		}
	}
	return &manifests{
		pkg:     l.pkg,
		bundles: l.bundles,
	}, nil
}

func (l manifests) GetPackageManifest() registry.PackageManifest {
	return l.pkg
}

func (l manifests) GetBundles() (bundles []*registry.Bundle) {
	for _, bundle := range l.bundles {
		bundles = append(bundles, bundle)
	}
	return bundles
}

func (l manifests) GetBundleForVersion(version string) (*registry.Bundle, error) {
	if _, err := semver.Parse(version); err != nil {
		return nil, errors.Wrapf(err, "error getting bundle for version %q", version)
	}
	bundle, ok := l.bundles[version]
	if !ok {
		return nil, errors.Errorf("bundle for version %q does not exist", version)
	}
	return bundle, nil
}

// MustBundleCSVToCSV converts a registry.ClusterServiceVersion bcsv to a
// v1alpha1.ClusterServiceVersion. The returned type will not have a status.
// MustBundleCSVToCSV will exit if bcsv's Spec is incorrectly formatted,
// since operator-registry should have not been able to parse the CSV
// if it were not.
func MustBundleCSVToCSV(bcsv *registry.ClusterServiceVersion) *olmapiv1alpha1.ClusterServiceVersion {
	spec := olmapiv1alpha1.ClusterServiceVersionSpec{}
	if err := json.Unmarshal(bcsv.Spec, &spec); err != nil {
		log.Fatalf("Error converting bundle CSV %q type: %v", bcsv.GetName(), err)
	}
	return &olmapiv1alpha1.ClusterServiceVersion{
		TypeMeta:   bcsv.TypeMeta,
		ObjectMeta: bcsv.ObjectMeta,
		Spec:       spec,
	}
}
