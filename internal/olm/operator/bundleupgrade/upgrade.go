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
	"errors"
	"fmt"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/alpha/action"
	declarativeconfig "github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	bundleInstall "github.com/operator-framework/operator-sdk/internal/olm/operator/bundle"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry"
)

const (
	channelSchema = "olm.channel"
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
	// --mode is hidden so only users who know what they're doing can alter add mode.
	fs.StringVar((*string)(&u.BundleAddMode), "mode", "", "mode to use for adding new bundle version to index")
	_ = fs.MarkHidden("mode")

	u.IndexImageCatalogCreator.BindFlags(fs)
}

func (u Upgrade) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	if err := u.setup(ctx); err != nil {
		return nil, err
	}
	return u.UpgradeOperator(ctx)
}

func (u *Upgrade) setup(ctx context.Context) error {
	// Bundle add mode is defaulted based on in-cluster metadata in u.UpgradeOperator(),
	// so validate only if it was set by a user.
	if u.BundleAddMode != "" {
		if err := u.BundleAddMode.Validate(); err != nil {
			return err
		}
	}

	labels, bundle, err := operator.LoadBundle(ctx, u.BundleImage, u.SkipTLS)
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
	u.IndexImageCatalogCreator.IndexImage = registry.DefaultIndexImage

	// determine if the image is SQLite or FBC
	catalogLabels, err := registryutil.GetImageLabels(ctx, nil, u.IndexImageCatalogCreator.IndexImage, false)
	if err != nil {
		return fmt.Errorf("get index image labels: %v", err)
	}

	if _, hasDBLabel := catalogLabels[containertools.DbLocationLabel]; hasDBLabel {
		log.Infof("Converting SQLite Image to a File-Based Catalog")
	}

	// write out FBC data used in ephemeral pod
	directoryName := filepath.Join("/tmp", strings.Split(csv.Name, ".")[0]+"-index")
	fileName := filepath.Join(directoryName, "testUpgradedFBC")
	bundleChannel := strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]

	if u.IndexImageCatalogCreator.ChannelEntrypoint != "" {
		bundleChannel = u.IndexImageCatalogCreator.ChannelEntrypoint
	}

	// FBC variables
	f := &bundleInstall.FBCContext{
		BundleImage:    u.BundleImage,
		Package:        labels[registrybundle.PackageLabel],
		DefaultChannel: bundleChannel,
		ChannelSchema:  channelSchema,
		ChannelName:    bundleChannel,
		Refs:           []string{u.BundleImage, u.IndexImageCatalogCreator.FBCImage},
		ChannelEntry: declarativeconfig.ChannelEntry{
			Name:     csv.Name,
			Replaces: u.IndexImageCatalogCreator.UpgradeEdge,
		},
	}

	// generate a file-based catalog representation of the bundle image
	declcfg, err := upgradeFBC(f, ctx)
	if err != nil {
		log.Errorf("error creating the upgraded FBC: %v", err)
		return err
	}

	// convert declarative config to string
	content, err := bundleInstall.StringifyDecConfig(declcfg)

	if err != nil {
		log.Errorf("error converting declarative config to string: %v", err)
		return err
	}

	// validate the declarative config
	if err = bundleInstall.ValidateFBC(declcfg); err != nil {
		log.Errorf("error validating the generated FBC: %v", err)
		return err
	}

	fmt.Println("FBC Content")
	fmt.Println(content)

	if content == "" {
		return errors.New("File-Based Catalog contents cannot be empty")
	}

	log.Infof("Generated a valid Upgraded File-Based Catalog")

	u.IndexImageCatalogCreator.FBCcontent = content
	u.IndexImageCatalogCreator.FBCdir = directoryName
	u.IndexImageCatalogCreator.FBCfile = fileName

	return nil
}

func upgradeFBC(f *bundleInstall.FBCContext, ctx context.Context) (*declarativeconfig.DeclarativeConfig, error) {
	var (
		declcfg *declarativeconfig.DeclarativeConfig
		err     error
	)

	if len(f.Refs) < 2 {
		return nil, errors.New("error: incorrect number of references: a bundle image and an index image must be passed in")
	}

	// Rendering the bundle image and index image into declarative config format
	render := action.Render{
		Refs: f.Refs,
	}

	declcfg, err = render.Run(ctx)
	if err != nil {
		log.Errorf("error in rendering the bundle and index image: %v", err)
		return nil, err
	}

	// Ensuring a valid bundle size
	if len(declcfg.Bundles) < 1 {
		log.Errorf("error in rendering the correct number of bundles: %v", err)
		return nil, err
	}

	// Ensuring a valid package size
	if len(declcfg.Packages) < 1 {
		log.Errorf("error in rendering the correct number of packages: %v", err)
		return nil, err
	}

	// Ensuring a valid channel size
	if len(declcfg.Channels) < 1 {
		log.Errorf("error in rendering the correct number of channels: %v", err)
		return nil, err
	}

	// Finding the correct channel to insert into
	for i, _ := range declcfg.Channels {
		if declcfg.Channels[i].Name == f.ChannelName && declcfg.Channels[i].Package == f.Package {
			declcfg.Channels[i].Entries = append(declcfg.Channels[i].Entries, f.ChannelEntry)
		}
	}

	return declcfg, nil
}
