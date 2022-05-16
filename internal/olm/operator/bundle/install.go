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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/alpha/action"
	declarativeconfig "github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
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

func (i *Install) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&i.IndexImage, "index-image", registry.DefaultIndexImage, "index image in which to inject bundle")
	fs.Var(&i.InstallMode, "install-mode", "install mode")

	// --mode is hidden so only users who know what they're doing can alter add mode.
	fs.StringVar((*string)(&i.BundleAddMode), "mode", "", "mode to use for adding bundle to index")
	_ = fs.MarkHidden("mode")

	i.IndexImageCatalogCreator.BindFlags(fs)
}

func (i Install) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := i.setup(ctx); err != nil {
		return nil, err
	}
	return i.InstallOperator(ctx)
}

func (i *Install) setup(ctx context.Context) error {
	// Validate add mode in case it was set by a user.
	if i.BundleAddMode != "" {
		if err := i.BundleAddMode.Validate(); err != nil {
			return err
		}
	}

	//if user sets --skip-tls then set --use-http to true as --skip-tls is deprecated
	if i.SkipTLS {
		i.UseHTTP = true
	}

	// Load bundle labels and set label-dependent values.
	labels, bundle, err := operator.LoadBundle(ctx, i.BundleImage, i.SkipTLSVerify, i.UseHTTP)
	if err != nil {
		return err
	}
	csv := bundle.CSV

	if err := i.InstallMode.CheckCompatibility(csv, i.cfg.Namespace); err != nil {
		return err
	}

	// get index image labels.
	catalogLabels, err := registryutil.GetImageLabels(ctx, nil, i.IndexImageCatalogCreator.IndexImage, false)
	if err != nil {
		return fmt.Errorf("get index image labels: %v", err)
	}

	// set the field to true if FBC label is on the image or for a default index image.
	if _, hasFBCLabel := catalogLabels[containertools.ConfigsLocationLabel]; hasFBCLabel || i.IndexImageCatalogCreator.IndexImage == registry.DefaultIndexImage {
		i.IndexImageCatalogCreator.HasFBCLabel = true
		if i.IndexImageCatalogCreator.BundleAddMode != "" {
			return fmt.Errorf("specifying the bundle add mode is not supported for File-Based Catalog bundles and index images")
		}
	} else {
		// index image is of the SQLite index format.
		deprecationMsg := fmt.Sprintf("%s is a SQLite index image. SQLite based index images are being deprecated and will be removed in a future release, please migrate your catalogs to the new File-Based Catalog format", i.IndexImageCatalogCreator.IndexImage)
		log.Warn(deprecationMsg)
	}

	if i.IndexImageCatalogCreator.HasFBCLabel {
		// FBC variables
		f := &registry.FBCContext{
			Package: labels[registrybundle.PackageLabel],
			Refs:    []string{i.BundleImage},
			ChannelEntry: declarativeconfig.ChannelEntry{
				Name: csv.Name,
			},
		}

		if _, hasChannelMetadata := labels[registrybundle.ChannelsLabel]; hasChannelMetadata {
			f.ChannelName = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]
		} else {
			f.ChannelName = registry.DefaultChannel
		}

		// generate an fbc if an fbc specific label is found on the image or for a default index image.
		content, err := generateFBCContent(f, ctx, i.BundleImage, i.IndexImageCatalogCreator.IndexImage)
		if err != nil {
			return fmt.Errorf("error generating File-Based Catalog with bundle %q: %v", i.BundleImage, err)
		}

		i.IndexImageCatalogCreator.FBCContent = content
	}

	i.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	i.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = csv.Name
	i.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)
	i.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]

	i.IndexImageCatalogCreator.PackageName = i.OperatorInstaller.PackageName
	i.IndexImageCatalogCreator.BundleImage = i.BundleImage

	return nil
}

// generateFBCContent creates a File-Based Catalog using the bundle image and index image from the run bundle command.
func generateFBCContent(f *registry.FBCContext, ctx context.Context, bundleImage, indexImage string) (string, error) {
	log.Infof("Creating a File-Based Catalog of the bundle %q", bundleImage)
	// generate a File-Based Catalog representation of the bundle image
	bundleDeclcfg, err := f.CreateFBC(ctx)
	if err != nil {
		return "", fmt.Errorf("error creating a File-Based Catalog with image %q: %v", bundleImage, err)
	}

	declcfg := &declarativeconfig.DeclarativeConfig{
		Bundles:  []declarativeconfig.Bundle{bundleDeclcfg.Bundle},
		Packages: []declarativeconfig.Package{bundleDeclcfg.Package},
		Channels: []declarativeconfig.Channel{bundleDeclcfg.Channel},
	}

	if indexImage != registry.DefaultIndexImage { // non-default index image was specified.
		// since an index image is specified, the bundle image will be added to the index image.
		// addBundleToIndexImage will ensure that the bundle is not already present in the index image and error out if it does.
		declcfg, err = addBundleToIndexImage(ctx, indexImage, bundleDeclcfg)
		if err != nil {
			return "", fmt.Errorf("error adding bundle image %q to index image %q: %v", bundleImage, indexImage, err)
		}
	}

	// validate the generated File-Based Catalog
	if err = registry.ValidateFBC(declcfg); err != nil {
		return "", fmt.Errorf("error validating the generated FBC: %v", err)
	}

	// convert declarative config to string
	content, err := registry.StringifyDeclConfig(declcfg)
	if err != nil {
		return "", fmt.Errorf("error converting the declarative config to string: %v", err)
	}

	if content == "" {
		return "", errors.New("file based catalog contents cannot be empty")
	}

	log.Infof("Generated a valid File-Based Catalog")

	return content, nil
}

// addBundleToIndexImage adds the bundle to an existing index image if the bundle is not already present in the index image.
func addBundleToIndexImage(ctx context.Context, indexImage string, bundleDeclConfig registry.BundleDeclcfg) (*declarativeconfig.DeclarativeConfig, error) {
	log.Infof("Rendering a File-Based Catalog of the Index Image %q", indexImage)
	log.SetOutput(ioutil.Discard)
	render := action.Render{
		Refs: []string{indexImage},
	}

	imageDeclConfig, err := render.Run(ctx)
	log.SetOutput(os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("error rendering the index image %q: %v", indexImage, err)
	}

	for _, bundle := range imageDeclConfig.Bundles {
		if bundle.Name == bundleDeclConfig.Bundle.Name && bundle.Package == bundleDeclConfig.Bundle.Package {
			return nil, fmt.Errorf("bundle %q already exists in the index image: %s", bundleDeclConfig.Bundle.Name, indexImage)
		}
	}

	for _, channel := range imageDeclConfig.Channels {
		if channel.Name == bundleDeclConfig.Channel.Name && channel.Package == bundleDeclConfig.Bundle.Package {
			return nil, fmt.Errorf("channel %q already exists in the index image: %s", bundleDeclConfig.Channel.Name, indexImage)
		}
	}

	var isPackagePresent bool
	for _, pkg := range imageDeclConfig.Packages {
		if pkg.Name == bundleDeclConfig.Package.Name {
			isPackagePresent = true
			break
		}
	}

	imageDeclConfig.Bundles = append(imageDeclConfig.Bundles, bundleDeclConfig.Bundle)
	imageDeclConfig.Channels = append(imageDeclConfig.Channels, bundleDeclConfig.Channel)

	if !isPackagePresent {
		imageDeclConfig.Packages = append(imageDeclConfig.Packages, bundleDeclConfig.Package)
	}

	log.Infof("Inserted the new bundle %q into the index image: %s", bundleDeclConfig.Bundle.Name, indexImage)

	return imageDeclConfig, nil
}
