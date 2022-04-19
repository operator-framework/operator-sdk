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
	"bytes"
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

const (
	schemaChannel = "olm.channel"
	schemaPackage = "olm.package"
)

type Install struct {
	BundleImage string

	*registry.IndexImageCatalogCreator
	*registry.OperatorInstaller

	cfg *operator.Configuration
}

type FBCContext struct {
	Package      string
	ChannelName  string
	Refs         []string
	ChannelEntry declarativeconfig.ChannelEntry
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
	}

	// generate an fbc if an fbc specific label is found on the image or for a default index image.
	if i.IndexImageCatalogCreator.HasFBCLabel {
		// FBC variables
		f := &FBCContext{
			Package:     labels[registrybundle.PackageLabel],
			ChannelName: strings.Split(labels[registrybundle.ChannelsLabel], ",")[0],
			Refs:        []string{i.BundleImage},
			ChannelEntry: declarativeconfig.ChannelEntry{
				Name: csv.Name,
			},
		}
		log.Infof("Creating a File-Based Catalog of the bundle %q", i.BundleImage)

		// generate a file-based catalog representation of the bundle image
		declcfg, err := f.createFBC(ctx)
		if err != nil {
			return fmt.Errorf("error generating file-based catalog with image %q: %v", i.BundleImage, err)
		}

		if i.IndexImageCatalogCreator.IndexImage != registry.DefaultIndexImage { // non-default index image was specified.
			// since an index image is specified, the bundle image will be added to the index image.
			// addBundleToIndexImage will ensure that the bundle is not already present in the index image and error out if it does.
			declcfg, err = addBundleToIndexImage(ctx, i.IndexImageCatalogCreator.IndexImage, declcfg)
			if err != nil {
				return fmt.Errorf("error adding bundle image %q to index image %q: %v", i.BundleImage, i.IndexImageCatalogCreator.IndexImage, err)
			}
		}

		// validate the file based catalog
		if err = validateFBC(declcfg); err != nil {
			return fmt.Errorf("error validating the generated FBC: %v", err)
		}

		// convert declarative config to string
		content, err := stringifyDecConfig(declcfg)
		if err != nil {
			return fmt.Errorf("error converting the declarative config to string: %v", err)
		}

		if content == "" {
			return errors.New("file based catalog contents cannot be empty")
		}

		log.Infof("Generated a valid File-Based Catalog")

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

// addBundleToIndexImage adds the bundle to an existing index image if the bundle is not already present in the index image.
func addBundleToIndexImage(ctx context.Context, indexImage string, bundleDeclConfig *declarativeconfig.DeclarativeConfig) (*declarativeconfig.DeclarativeConfig, error) {
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

	if len(bundleDeclConfig.Bundles) != 1 {
		return nil, fmt.Errorf("there should be exactly one bundle rendered with the bundle image")
	}

	// check if the bundle exists in the index image.
	if len(bundleDeclConfig.Channels) > 0 && len(bundleDeclConfig.Bundles) > 0 {
		for _, channel := range imageDeclConfig.Channels {
			// find the specific channel that the bundle needs to be inserted into.
			if channel.Name == bundleDeclConfig.Channels[0].Name && channel.Package == bundleDeclConfig.Channels[0].Package {
				// check if the CSV name is already present in the channel's entries.
				for _, entry := range channel.Entries {
					if entry.Name == bundleDeclConfig.Bundles[0].Name {
						return nil, fmt.Errorf("bundle %q already present in the index image: %s", bundleDeclConfig.Bundles[0].Name, indexImage)
					}
				}

				break // we only want to search through the specific channel.
			}
		}
	}

	if len(bundleDeclConfig.Bundles) > 0 && len(bundleDeclConfig.Channels) > 0 && len(bundleDeclConfig.Packages) > 0 {
		imageDeclConfig.Bundles = append(imageDeclConfig.Bundles, bundleDeclConfig.Bundles[0])
		imageDeclConfig.Channels = append(imageDeclConfig.Channels, bundleDeclConfig.Channels[0])
		imageDeclConfig.Packages = append(imageDeclConfig.Packages, bundleDeclConfig.Packages[0])
		imageDeclConfig.Others = append(imageDeclConfig.Others, bundleDeclConfig.Others...)

		log.Infof("Inserted the new bundle %q into the index image: %s", bundleDeclConfig.Bundles[0].Name, indexImage)
	}

	return imageDeclConfig, nil
}

// createFBC generates an FBC by creating bundle, package and channel blobs.
func (f *FBCContext) createFBC(ctx context.Context) (*declarativeconfig.DeclarativeConfig, error) {
	// Rendering the bundle image into a declarative config format.
	log.SetOutput(ioutil.Discard)
	render := action.Render{
		Refs: f.Refs,
	}

	// generate bundles by rendering the bundle objects.
	declcfg, err := render.Run(ctx)
	log.SetOutput(os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("error rendering the bundle image: %v", err)
	}

	// Ensuring a valid bundle size.
	if len(declcfg.Bundles) != 1 {
		return nil, fmt.Errorf("bundle image should contain exactly one bundle blob")
	}

	// generate packages.
	pkg := declarativeconfig.Package{
		Schema:         schemaPackage,
		Name:           f.Package,
		DefaultChannel: f.ChannelName,
	}

	declcfg.Packages = []declarativeconfig.Package{pkg}

	// generate channels.
	channel := declarativeconfig.Channel{
		Schema:  schemaChannel,
		Name:    f.ChannelName,
		Package: f.Package,
		Entries: []declarativeconfig.ChannelEntry{f.ChannelEntry},
	}

	declcfg.Channels = []declarativeconfig.Channel{channel}

	return declcfg, nil
}

// stringifyDecConfig writes the generated declarative config to a string.
func stringifyDecConfig(declcfg *declarativeconfig.DeclarativeConfig) (string, error) {
	var buf bytes.Buffer
	err := declarativeconfig.WriteJSON(*declcfg, &buf)
	if err != nil {
		return "", fmt.Errorf("error writing generated declarative config to JSON encoder: %v", err)
	}

	return buf.String(), nil
}

// validateFBC converts the generated declarative config to a model and validates it.
func validateFBC(declcfg *declarativeconfig.DeclarativeConfig) error {
	// validates and converts declarative config to model
	_, err := declarativeconfig.ConvertToModel(*declcfg)
	if err != nil {
		return fmt.Errorf("error converting the declarative config to model: %v", err)

	}

	return nil
}
