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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
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
	u.CatalogCreator = u.IndexImageCatalogCreator
	u.CatalogUpdater = u.IndexImageCatalogCreator
	return u
}

const defaultIndexImage = "quay.io/operator-framework/upstream-opm-builder:latest"

func (u *Upgrade) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&u.InjectBundleMode, "mode", "", "mode to use for adding new bundle version to index")
	_ = fs.MarkHidden("mode")
}

func (u Upgrade) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := u.setup(ctx); err != nil {
		return nil, err
	}
	return u.UpgradeOperator(ctx)
}

func (u *Upgrade) setup(ctx context.Context) error {
	labels, csv, err := loadBundle(ctx, u.BundleImage)
	if err != nil {
		return err
	}

	// TODO: Remove adding annotations here.
	u.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	u.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(u.OperatorInstaller.PackageName)
	u.OperatorInstaller.StartingCSV = csv.Name
	u.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)
	u.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]
	u.IndexImageCatalogCreator.IndexImage = defaultIndexImage
	u.IndexImageCatalogCreator.BundleImage = u.BundleImage
	u.IndexImageCatalogCreator.PackageName = u.OperatorInstaller.PackageName
	u.IndexImageCatalogCreator.InjectBundles = []string{u.BundleImage}
	u.IndexImageCatalogCreator.InjectBundleMode = "replaces"
	if u.IndexImageCatalogCreator.IndexImage == defaultIndexImage {
		u.IndexImageCatalogCreator.InjectBundleMode = "semver"
	}

	return nil
}

func loadBundle(ctx context.Context, bundleImage string) (registryutil.Labels, *v1alpha1.ClusterServiceVersion, error) {
	bundlePath, err := registryutil.ExtractBundleImage(ctx, nil, bundleImage, false)
	if err != nil {
		return nil, nil, fmt.Errorf("pull bundle image: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(bundlePath)
	}()

	labels, _, err := registryutil.FindBundleMetadata(bundlePath)
	if err != nil {
		return nil, nil, fmt.Errorf("load bundle metadata: %v", err)
	}

	relManifestsDir, ok := labels.GetManifestsDir()
	if !ok {
		return nil, nil, fmt.Errorf("manifests directory not defined in bundle metadata")
	}
	manifestsDir := filepath.Join(bundlePath, relManifestsDir)
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("load bundle: %v", err)
	}

	return labels, bundle.CSV, nil
}
