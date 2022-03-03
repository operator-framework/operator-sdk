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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/alpha/action"
	declarativeconfig "github.com/operator-framework/operator-registry/alpha/declcfg"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry"
)

type Install struct {
	BundleImage string

	*registry.IndexImageCatalogCreator
	*registry.OperatorInstaller

	cfg *operator.Configuration
}

type FBCContext struct {
	BundleImage       string
	Package           string
	DefaultChannel    string
	FBCName           string
	FBCPath           string
	FBCDirContext     string
	ChannelSchema     string
	ChannelName       string
	ChannelEntries    []declarativeconfig.ChannelEntry
	DescriptionReader io.Reader
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

	wd, err := os.Getwd()
	if err != nil {
		log.Error(err)
	}

	// TODO (rashmi/venkat) might have to clean this up to remove/update some stuff.
	f := &FBCContext{
		BundleImage:       i.BundleImage,
		Package:           labels[registrybundle.PackageLabel],
		FBCDirContext:     "testdata",
		FBCPath:           filepath.Join(wd, "testdata"),
		FBCName:           "testFBC",
		DescriptionReader: bytes.NewBufferString("foo"),
		DefaultChannel:    "foo",
		ChannelSchema:     "olm.channel",
		ChannelName:       "foo",
	}

	// generate an FBC
	declcfg, err := f.createFBC()
	if err != nil {
		log.Errorf("error creating a minimal FBC: %v", err)
		return err
	}

	// write declarative config to file
	if err = f.writeDecConfigToFile(declcfg); err != nil {
		log.Errorf("error writing declarative config to file: %v", err)
		return err
	}

	// validate the generated declarative config
	if err = validateFBC(declcfg); err != nil {
		log.Errorf("error validating the generated FBC: %v", err)
		return err
	}

	i.OperatorInstaller.PackageName = labels[registrybundle.PackageLabel]
	i.OperatorInstaller.CatalogSourceName = operator.CatalogNameForPackage(i.OperatorInstaller.PackageName)
	i.OperatorInstaller.StartingCSV = csv.Name
	i.OperatorInstaller.SupportedInstallModes = operator.GetSupportedInstallModes(csv.Spec.InstallModes)
	i.OperatorInstaller.Channel = strings.Split(labels[registrybundle.ChannelsLabel], ",")[0]

	i.IndexImageCatalogCreator.PackageName = i.OperatorInstaller.PackageName
	i.IndexImageCatalogCreator.BundleImage = i.BundleImage
	// TODO (rashmi/venkat) add FBC to IndexImageCatalogCreator
	return nil
}

//createFBC generates an FBC by creating bundle, package and channel blobs.
func (f *FBCContext) createFBC() (*declarativeconfig.DeclarativeConfig, error) {

	var (
		declcfg        *declarativeconfig.DeclarativeConfig
		declcfgpackage *declarativeconfig.Package
		err            error
	)

	render := action.Render{
		Refs: []string{f.BundleImage},
	}

	// generate bundles by rendering the bundle objects.
	declcfg, err = render.Run(context.TODO())
	if err != nil {
		log.Errorf("error in rendering the bundle image: %v", err)
		return nil, err
	}

	if len(declcfg.Bundles) < 0 {
		log.Errorf("error in rendering the correct number of bundles: %v", err)
		return nil, err
	}
	// validate length of bundles and add them to declcfg.Bundles.
	if len(declcfg.Bundles) == 1 {
		declcfg.Bundles = []declarativeconfig.Bundle{*&declcfg.Bundles[0]}
	} else {
		return nil, errors.New("error in expected length of bundles")
	}

	// init packages
	init := action.Init{
		Package:           f.Package,
		DefaultChannel:    f.DefaultChannel,
		DescriptionReader: f.DescriptionReader,
	}

	// generate packages
	declcfgpackage, err = init.Run()
	if err != nil {
		log.Errorf("error in generating packages for the FBC: %v", err)
		return nil, err
	}
	declcfg.Packages = []declarativeconfig.Package{*declcfgpackage}

	// generate channels
	channel := declarativeconfig.Channel{
		Schema:  f.ChannelSchema,
		Name:    f.ChannelName,
		Package: f.Package,
		Entries: f.ChannelEntries,
	}

	declcfg.Channels = []declarativeconfig.Channel{channel}

	return declcfg, nil
}

// writeDecConfigToFile writes the generated declarative config to a file.
func (f *FBCContext) writeDecConfigToFile(declcfg *declarativeconfig.DeclarativeConfig) error {
	var buf bytes.Buffer
	err := declarativeconfig.WriteJSON(*declcfg, &buf)
	if err != nil {
		log.Errorf("error writing to JSON encoder: %v", err)
		return err
	}
	if err := os.MkdirAll(f.FBCDirContext, 0755); err != nil {
		log.Errorf("error creating a directory for FBC: %v", err)
		return err
	}
	fbcFilePath := filepath.Join(f.FBCPath, f.FBCName)
	file, err := os.OpenFile(fbcFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("error opening FBC file: %v", err)
		return err
	}

	defer file.Close()

	if _, err := file.WriteString(string(buf.Bytes()) + "\n"); err != nil {
		log.Errorf("error writing to FBC file: %v", err)
		return err
	}

	return nil
}

// validateFBC converts the generated declarative config to a model and validates it.
func validateFBC(declcfg *declarativeconfig.DeclarativeConfig) error {
	// convert declarative config to model
	FBCmodel, err := declarativeconfig.ConvertToModel(*declcfg)
	if err != nil {
		log.Errorf("error converting the declarative config to mode: %v", err)
		return err
	}

	if err = FBCmodel.Validate(); err != nil {
		log.Errorf("error validating the generated FBC: %v", err)
		return err
	}

	return nil
}
