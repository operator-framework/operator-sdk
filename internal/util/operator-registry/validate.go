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
	"fmt"
	"regexp"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/appregistry"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/pkg/errors"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// ValidatePackageManifest ensures each datum in pkg is valid relative to other
// related data in pkg.
func ValidatePackageManifest(pkg *registry.PackageManifest) error {
	if pkg.PackageName == "" {
		return errors.New("package name cannot be empty")
	}
	if len(pkg.Channels) == 0 {
		return errors.New("channels cannot be empty")
	}
	if pkg.DefaultChannelName == "" {
		return errors.New("default channel cannot be empty")
	}

	seen := map[string]struct{}{}
	for i, c := range pkg.Channels {
		if c.Name == "" {
			return fmt.Errorf("channel %d name cannot be empty", i)
		}
		if c.CurrentCSVName == "" {
			return fmt.Errorf("channel %q currentCSV cannot be empty", c.Name)
		}
		if _, ok := seen[c.Name]; ok {
			return fmt.Errorf("duplicate package manifest channel name %q; channel names must be unique", c.Name)
		}
		seen[c.Name] = struct{}{}
	}
	if _, ok := seen[pkg.DefaultChannelName]; !ok {
		return fmt.Errorf("default channel %q does not exist in channels", pkg.DefaultChannelName)
	}

	return nil
}

// dns1123LabelRegexp defines the character set allowed in a DNS 1123 label.
var dns1123LabelRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")

// FormatOperatorNameDNS1123 ensures name is DNS1123 label-compliant by
// replacing all non-compliant UTF-8 characters with "-".
func FormatOperatorNameDNS1123(name string) string {
	if len(validation.IsDNS1123Label(name)) != 0 {
		return dns1123LabelRegexp.ReplaceAllString(name, "-")
	}
	return name
}

// validateBundle ensures all objects in bundle have the correct data.
// TODO(estroz): remove once operator-verify library is complete.
func validateBundle(bundle *registry.Bundle) (err error) {
	bcsv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return err
	}
	csv, err := BundleCSVToCSV(bcsv)
	if err != nil {
		return err
	}
	crds, err := bundle.CustomResourceDefinitions()
	if err != nil {
		return err
	}
	crdMap := map[string]struct{}{}
	for _, crd := range crds {
		for _, k := range getCRDKeys(crd) {
			crdMap[k.String()] = struct{}{}
		}
	}
	// If at least one CSV has an owned CRD it must be present.
	if len(csv.Spec.CustomResourceDefinitions.Owned) > 0 && len(crds) == 0 {
		return errors.Errorf("bundled CSV has an owned CRD but no CRD's are present in bundle dir")
	}
	// Ensure all CRD's referenced in each CSV exist in BundleDir.
	for _, o := range csv.Spec.CustomResourceDefinitions.Owned {
		key := getCRDDescKey(o)
		if _, hasCRD := crdMap[key.String()]; !hasCRD {
			return errors.Errorf("bundle dir does not contain owned CRD %q from CSV %q", key, csv.GetName())
		}
	}
	if !hasSupportedInstallMode(csv) {
		return errors.Errorf("at least one installMode must be marked \"supported\" in CSV %q", csv.GetName())
	}
	return nil
}

// hasSupportedInstallMode returns true if a csv supports at least one
// installMode.
func hasSupportedInstallMode(csv *olmapiv1alpha1.ClusterServiceVersion) bool {
	for _, mode := range csv.Spec.InstallModes {
		if mode.Supported {
			return true
		}
	}
	return false
}

// getCRDKeys returns a key uniquely identifying crd.
func getCRDDescKey(crd olmapiv1alpha1.CRDDescription) appregistry.CRDKey {
	return appregistry.CRDKey{
		Kind:    crd.Kind,
		Name:    crd.Name,
		Version: crd.Version,
	}
}

// getCRDKeys returns a set of keys uniquely identifying crd per version.
// getCRDKeys assumes at least one of spec.version, spec.versions is non-empty.
func getCRDKeys(crd *apiextv1beta1.CustomResourceDefinition) (keys []appregistry.CRDKey) {
	if crd.Spec.Version != "" && len(crd.Spec.Versions) == 0 {
		return []appregistry.CRDKey{{
			Kind:    crd.Spec.Names.Kind,
			Name:    crd.GetName(),
			Version: crd.Spec.Version,
		},
		}
	}
	for _, v := range crd.Spec.Versions {
		keys = append(keys, appregistry.CRDKey{
			Kind:    crd.Spec.Names.Kind,
			Name:    crd.GetName(),
			Version: v.Name,
		})
	}
	return keys
}
