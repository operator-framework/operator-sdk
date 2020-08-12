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

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/operator"
	"github.com/operator-framework/operator-sdk/internal/operator/internal"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

type Install struct {
	BundleImage string

	*internal.IndexImageCatalogCreator
	*internal.OperatorInstaller
}

func NewInstall(cfg *operator.Configuration) Install {
	i := Install{
		OperatorInstaller: internal.NewOperatorInstaller(cfg),
	}
	i.IndexImageCatalogCreator = internal.NewIndexImageCatalogCreator(cfg)
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
		return fmt.Errorf("load bundle: %v", err)
	}

	i.OperatorInstaller.PackageName = labels["operators.operatorframework.io.bundle.package.v1"]
	i.OperatorInstaller.CatalogSourceName = fmt.Sprintf("%s-catalog", i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = csv.Name
	i.OperatorInstaller.Channel = strings.Split(labels["operators.operatorframework.io.bundle.channels.v1"], ",")[0]

	i.IndexImageCatalogCreator.InjectBundles = []string{i.BundleImage}
	i.IndexImageCatalogCreator.InjectBundleMode = "replaces"
	if i.IndexImageCatalogCreator.IndexImage == defaultIndexImage {
		i.IndexImageCatalogCreator.InjectBundleMode = "semver"
	}

	return nil
}

func loadBundle(ctx context.Context, bundleImage string) (labels registryutil.Labels, csv *registry.ClusterServiceVersion, err error) {
	bundlePath, err := registryutil.ExtractBundleImage(ctx, nil, bundleImage, false)
	if err != nil {
		return nil, nil, fmt.Errorf("pull bundle image: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(bundlePath)
	}()

	labels, _, err = registryutil.FindBundleMetadata(bundlePath)
	if err != nil {
		return nil, nil, fmt.Errorf("load bundle metadata: %v", err)
	}

	relManifestsDir, ok := labels.GetManifestsDir()
	if !ok {
		relManifestsDir = "manifests"
	}
	manifestsDir := filepath.Join(bundlePath, relManifestsDir)
	csv, err = registry.ReadCSVFromBundleDirectory(manifestsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("read bundle csv: %v", err)
	}

	return labels, csv, nil
}
