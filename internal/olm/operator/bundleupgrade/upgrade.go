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

package bundleupgrade

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

type Upgrade struct {
	BundleImage string

	*registry.IndexImageCatalogCreator
	*registry.OperatorInstaller

	cfg *operator.Configuration
}

func NewUpgrade(cfg *operator.Configuration) Upgrade {
	u := Upgrade{
		OperatorInstaller: registry.NewOperatorInstaller(cfg),
		cfg:               cfg,
	}
	u.IndexImageCatalogCreator = registry.NewIndexImageCatalogCreator(cfg)
	u.CatalogUpdater = u.IndexImageCatalogCreator
	return u
}

func (u *Upgrade) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&u.BundleAddMode, "mode", "", "mode to use for adding new bundle version to index")
	_ = fs.MarkHidden("mode")
}

func (u Upgrade) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := u.setup(ctx); err != nil {
		return nil, err
	}
	return u.UpgradeOperator(ctx)
}

func (u *Upgrade) setup(ctx context.Context) error {
	labels, bundle, err := operator.LoadBundle(ctx, u.BundleImage)
	if err != nil {
		return err
	}
	csv := bundle.CSV

	u.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	u.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(u.OperatorInstaller.PackageName)
	u.OperatorInstaller.StartingCSV = csv.Name
	u.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)
	u.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]

	// Since an existing CatalogSource will have an annotation containing the existing index image,
	// defer defaulting the bundle add mode to after the existing CatalogSource is retrieved.
	u.IndexImageCatalogCreator.PackageName = u.OperatorInstaller.PackageName
	u.IndexImageCatalogCreator.BundleImage = u.BundleImage
	u.IndexImageCatalogCreator.IndexImage = index.DefaultIndexImage

	return nil
}
