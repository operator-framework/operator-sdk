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
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	declarativeconfig "github.com/operator-framework/operator-registry/alpha/declcfg"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	fbcutil "github.com/operator-framework/operator-sdk/internal/olm/fbcutil"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry"
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
	fs.StringVar(&i.IndexImage, "index-image", fbcutil.DefaultIndexImage, "index image in which to inject bundle")
	fs.StringVar(&i.InitImage, "decompression-image", fbcutil.DefaultInitImage, "image used in an init container in "+
		"the registry pod to decompress the compressed catalog contents. cat and gzip binaries are expected to exist "+
		"in the PATH")
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

	// check if index image adopts File-Based Catalog or SQLite index image format
	isFBCImage, err := fbcutil.IsFBC(ctx, i.IndexImageCatalogCreator.IndexImage)
	if err != nil {
		return fmt.Errorf("error determining whether index %q is FBC or SQLite based: %v", i.IndexImageCatalogCreator.IndexImage, err)
	}
	i.IndexImageCatalogCreator.HasFBCLabel = isFBCImage

	// set the field to true if FBC label is on the image or for a default index image.
	if i.IndexImageCatalogCreator.HasFBCLabel {
		if i.IndexImageCatalogCreator.BundleAddMode != "" {
			return fmt.Errorf("specifying the bundle add mode is not supported for File-Based Catalog bundles and index images")
		}

		// FBC variables
		f := &fbcutil.FBCContext{
			Package: labels[registrybundle.PackageLabel],
			Refs:    []string{i.BundleImage},
			ChannelEntry: declarativeconfig.ChannelEntry{
				Name: csv.Name,
			},
			SkipTLSVerify: i.SkipTLSVerify,
			UseHTTP:       i.UseHTTP,
		}

		// ignore channels for the bundle and instead use the default
		f.ChannelName = fbcutil.DefaultChannel

		// generate an fbc if an fbc specific label is found on the image or for a default index image.
		content, err := generateFBCContent(ctx, f, i.BundleImage, i.IndexImageCatalogCreator.IndexImage)
		if err != nil {
			return fmt.Errorf("error generating File-Based Catalog with bundle %q: %v", i.BundleImage, err)
		}

		i.IndexImageCatalogCreator.FBCContent = content
		i.OperatorInstaller.Channel = fbcutil.DefaultChannel
	} else {
		// index image is of the SQLite index format.
		deprecationMsg := fmt.Sprintf("%s is a SQLite index image. SQLite based index images are being deprecated and will be removed in a future release, please migrate your catalogs to the new File-Based Catalog format", i.IndexImageCatalogCreator.IndexImage)
		log.Warn(deprecationMsg)

		// set the channel the old way
		i.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]
	}

	i.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	i.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = csv.Name
	i.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)

	i.IndexImageCatalogCreator.PackageName = i.OperatorInstaller.PackageName
	i.IndexImageCatalogCreator.BundleImage = i.BundleImage

	return nil
}

// generateFBCContent creates a File-Based Catalog using the bundle image and index image from the run bundle command.
func generateFBCContent(ctx context.Context, f *fbcutil.FBCContext, bundleImage, indexImage string) (string, error) {
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

	if indexImage != fbcutil.DefaultIndexImage { // non-default index image was specified.
		// since an index image is specified, the bundle image will be added to the index image.
		// generateExtraFBC will ensure that the bundle is not already present in the index image and error out if it does.
		declcfg, err = generateFBC(ctx, indexImage, bundleDeclcfg, f.SkipTLSVerify, f.UseHTTP)
		if err != nil {
			return "", fmt.Errorf("error adding bundle image %q to index image %q: %v", bundleImage, indexImage, err)
		}
	}

	// validate the declarative config and convert it to a string
	var content string
	if content, err = fbcutil.ValidateAndStringify(declcfg); err != nil {
		return "", fmt.Errorf("error validating and converting the declarative config object to a string format: %v", err)
	}

	log.Infof("Generated a valid File-Based Catalog")

	return content, nil
}

// generateFBC verifies that a bundle is not already present on the index and if not, it renders the bundle contents into a
// declarative config type.
func generateFBC(ctx context.Context, indexImage string, bundleDeclConfig fbcutil.BundleDeclcfg, skipTLSVerify bool, useHTTP bool) (*declarativeconfig.DeclarativeConfig, error) {
	log.Infof("Rendering a File-Based Catalog of the Index Image %q to verify if bundle %q is present", indexImage, bundleDeclConfig.Bundle.Name)

	imageDeclConfig, err := fbcutil.RenderRefs(ctx, []string{indexImage}, skipTLSVerify, useHTTP)
	if err != nil {
		return nil, err
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

	extraDeclConfig := &declarativeconfig.DeclarativeConfig{
		Bundles:  append(imageDeclConfig.Bundles, bundleDeclConfig.Bundle),
		Channels: append(imageDeclConfig.Channels, bundleDeclConfig.Channel),
	}

	if !isPackagePresent {
		extraDeclConfig.Packages = append(imageDeclConfig.Packages, bundleDeclConfig.Package)
	}

	log.Infof("Generated the extra FBC for the bundle image %q", bundleDeclConfig.Bundle.Name)

	return extraDeclConfig, nil
}
