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

package packagemanifests

import (
	"context"
	"errors"
	"fmt"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry"
)

type Install struct {
	PackageManifestsDirectory string
	Version                   string

	*registry.ConfigMapCatalogCreator
	*registry.OperatorInstaller

	cfg *operator.Configuration
}

func NewInstall(cfg *operator.Configuration) Install {
	i := Install{
		ConfigMapCatalogCreator: registry.NewConfigMapCatalogCreator(cfg),
		OperatorInstaller:       registry.NewOperatorInstaller(cfg),
		cfg:                     cfg,
	}
	i.OperatorInstaller.CatalogCreator = i.ConfigMapCatalogCreator
	return i
}

func (i *Install) BindFlags(fs *pflag.FlagSet) {
	fs.Var(&i.InstallMode, "install-mode", "install mode")
	fs.StringVar(&i.Version, "version", "", "Packaged version of the operator to deploy")
}

func (i Install) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := i.setup(); err != nil {
		return nil, err
	}
	return i.InstallOperator(ctx)
}

func (i *Install) setup() error {
	pkg, bundles, err := loadPackageManifests(i.PackageManifestsDirectory)
	if err != nil {
		return fmt.Errorf("load package manifests: %v", err)
	}
	bundle, err := getPackageForVersion(bundles, i.Version)
	if err != nil {
		return err
	}

	if err := i.InstallMode.CheckCompatibility(bundle.CSV, i.cfg.Namespace); err != nil {
		return err
	}

	i.OperatorInstaller.PackageName = pkg.PackageName
	i.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = bundle.CSV.GetName()
	i.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(bundle.CSV.Spec.InstallModes)

	if i.OperatorInstaller.SupportedInstallModes.Len() == 0 {
		return fmt.Errorf("operator %q is not installable: no supported install modes", bundle.CSV.GetName())
	}

	i.OperatorInstaller.Channel, err = getChannelForCSVName(pkg, i.OperatorInstaller.StartingCSV)
	if err != nil {
		return err
	}

	i.ConfigMapCatalogCreator.Package = pkg
	i.ConfigMapCatalogCreator.Bundles = bundles

	return nil
}

func loadPackageManifests(rootDir string) (*apimanifests.PackageManifest, []*apimanifests.Bundle, error) {
	// Operator bundles and metadata.
	pkg, bundles, err := apimanifests.GetManifestsDir(rootDir)
	if err != nil {
		return nil, nil, err
	}
	if len(bundles) == 0 {
		return nil, nil, errors.New("no packages found")
	}
	if pkg == nil || pkg.PackageName == "" {
		return nil, nil, errors.New("no package manifest found")
	}
	return pkg, bundles, nil
}

func getPackageForVersion(bundles []*apimanifests.Bundle, version string) (*apimanifests.Bundle, error) {
	versions := []string{}
	for _, bundle := range bundles {
		verStr := bundle.CSV.Spec.Version.String()
		if verStr == version {
			return bundle, nil
		}
		versions = append(versions, verStr)
	}
	return nil, fmt.Errorf("no package found for version %s; valid versions: %+q", version, versions)
}

func getChannelForCSVName(pkg *apimanifests.PackageManifest, csvName string) (string, error) {
	for _, c := range pkg.Channels {
		if c.CurrentCSVName == csvName {
			return c.Name, nil
		}
	}
	return "", fmt.Errorf("no channel in package manifest %s exists for CSV %s", pkg.PackageName, csvName)
}
