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

const defaultIndexImage = "quay.io/operator-framework/upstream-opm-builder:latest"

func (i *Install) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&i.IndexImage, "index-image", defaultIndexImage, "index image in which to inject bundle")
	fs.Var(&i.InstallMode, "install-mode", "install mode")
	fs.StringVar(&i.InjectBundleMode, "mode", "", "mode to use for adding bundle to index")
	_ = fs.MarkHidden("mode")
}

func (i Install) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := i.setup(ctx); err != nil {
		return nil, err
	}
	return i.InstallOperator(ctx)
}

func (i *Install) setup(ctx context.Context) error {
	labels, csv, err := loadBundle(ctx, i.BundleImage)
	if err != nil {
		return err
	}

	if err := i.InstallMode.CheckCompatibility(csv, i.cfg.Namespace); err != nil {
		return err
	}

	i.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	i.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = csv.Name
	i.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)
	i.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]
	i.IndexImageCatalogCreator.BundleImage = i.BundleImage
	i.IndexImageCatalogCreator.PackageName = i.OperatorInstaller.PackageName
	i.IndexImageCatalogCreator.InjectBundles = []string{i.BundleImage}
	i.IndexImageCatalogCreator.InjectBundleMode = "replaces"
	if i.IndexImageCatalogCreator.IndexImage == defaultIndexImage {
		i.IndexImageCatalogCreator.InjectBundleMode = "semver"
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
