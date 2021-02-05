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

package bundle

import (
	"context"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
)

type Install struct {
	BundleImage string

	*registry.IndexImageCatalogCreator
	*registry.OperatorInstaller

	cfg *operator.Configuration
}

func NewInstall(cfg *operator.Configuration) Install {
	i := Install{
		OperatorInstaller: registry.NewOperatorInstaller(cfg),
		cfg:               cfg,
	}
	i.IndexImageCatalogCreator = registry.NewIndexImageCatalogCreator(cfg)
	i.CatalogCreator = i.IndexImageCatalogCreator
	return i
}

func (i *Install) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&i.IndexImage, "index-image", index.DefaultIndexImage, "index image in which to inject bundle")
	fs.Var(&i.InstallMode, "install-mode", "install mode")
	fs.StringVar(&i.BundleAddMode, "mode", "", "mode to use for adding bundle to index")
	_ = fs.MarkHidden("mode")
}

func (i Install) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := i.setup(ctx); err != nil {
		return nil, err
	}
	return i.InstallOperator(ctx)
}

func (i *Install) setup(ctx context.Context) error {
	labels, bundle, err := operator.LoadBundle(ctx, i.BundleImage)
	if err != nil {
		return err
	}
	csv := bundle.CSV

	if err := i.InstallMode.CheckCompatibility(csv, i.cfg.Namespace); err != nil {
		return err
	}

	i.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	i.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = csv.Name
	i.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)
	i.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]

	i.IndexImageCatalogCreator.PackageName = i.OperatorInstaller.PackageName
	i.IndexImageCatalogCreator.BundleImage = i.BundleImage
	if i.IndexImageCatalogCreator.BundleAddMode == "" {
		i.IndexImageCatalogCreator.BundleAddMode = index.ReplacesBundleAddMode
		if i.IndexImageCatalogCreator.IndexImage == index.DefaultIndexImage {
			i.IndexImageCatalogCreator.BundleAddMode = index.SemverBundleAddMode
		}
	}

	return nil
}
