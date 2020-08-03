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

package generate

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation"
	log "github.com/sirupsen/logrus"

	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest/bases"
)

const (
	// File extension for all PackageManifests written by Generator.
	packageManifestFileExt = ".package.yaml"
)

var (
	// User-facing errors.
	errNoVersion = errors.New("version must be set")

	// Internal errors.
	errNoOpts   = genutil.InternalError("generator options must be set")
	errNoOpName = genutil.InternalError("operator name must be set")
	//errNoBase      = genutil.InternalError("base directory must be set")
	errNoOutputWriter = genutil.InternalError("output writer must be set")
)

type PkgOptions struct {
	// OperatorName is the operator's name, ex. app-operator.
	OperatorName string
	// Version is the version of the operator being updated.
	Version string
	// ChannelName is operator's PackageManifest channel. If a new PackageManifest is generated
	// or ChannelName is the only channel in the generated PackageManifest,
	// this channel will be set to the PackageManifest's default.
	ChannelName string
	// IsDefaultChannel determines whether ChannelName should be the default channel in the
	// generated PackageManifest. If true, ChannelName will be the PackageManifest's default channel.
	// Setting this field is only necessary when more than one channel exists.
	IsDefaultChannel bool

	//BaseDir directory to look for a base package manifest
	BaseDir      string
	OutputWriter io.Writer
}

// Generate configures the Generator with opts then runs it.
func (g Generator) GeneratePackageManifest(opts *PkgOptions) error {
	if opts == nil {
		return errNoOpts
	} else if opts.OperatorName == "" {
		return errNoOpName
	} else if opts.OutputWriter == nil {
		return errNoOutputWriter
	} else if opts.Version == "" {
		return errNoVersion
	}

	pkg, err := g.generatePackageManifest(opts)
	if err != nil {
		return err
	}

	return genutil.WriteYAML(opts.OutputWriter, pkg)
}

// generatePackageManifest takes the input and generates the populated package manifest object
func (g *Generator) generatePackageManifest(opts *PkgOptions) (*apimanifests.PackageManifest, error) {
	b := bases.PackageManifest{
		PackageName: opts.OperatorName,
	}
	if opts.BaseDir != "" {
		b.BasePath = filepath.Join(opts.BaseDir, makePkgManFileName(opts.OperatorName))
	}
	base, err := b.GetBase()
	if err != nil {
		return nil, fmt.Errorf("error getting PackageManifest base: %v", err)
	}

	csvName := genutil.MakeCSVName(opts.OperatorName, opts.Version)
	if opts.ChannelName != "" {
		setChannels(base, opts.ChannelName, csvName)
		sortChannelsByName(base)
		if opts.IsDefaultChannel || len(base.Channels) == 1 {
			base.DefaultChannelName = opts.ChannelName
		}
	} else if len(base.Channels) == 0 {
		setChannels(base, "alpha", csvName)
		base.DefaultChannelName = "alpha"
	}

	if err = validatePackageManifest(base); err != nil {
		return nil, err
	}

	return base, nil
}

// makePkgManFileName will return the file name of a PackageManifest.
func makePkgManFileName(operatorName string) string {
	return strings.ToLower(operatorName) + packageManifestFileExt
}

// sortChannelsByName sorts pkg.Channels by each element's name.
func sortChannelsByName(pkg *apimanifests.PackageManifest) {
	sort.Slice(pkg.Channels, func(i int, j int) bool {
		return pkg.Channels[i].Name < pkg.Channels[j].Name
	})
}

// validatePackageManifest will validate pkg and log warnings and errors.
// If a validation error is encountered, an error is returned.
func validatePackageManifest(pkg *apimanifests.PackageManifest) error {
	if pkg == nil {
		return errors.New("empty PackageManifest")
	}

	hasErrors := false
	results := validation.PackageManifestValidator.Validate(pkg)
	for _, r := range results {
		for _, e := range r.Errors {
			log.Errorf("PackageManifest validation: [%s] %s", e.Type, e.Detail)
		}
		for _, w := range r.Warnings {
			log.Warnf("PackageManifest validation: [%s] %s", w.Type, w.Detail)
		}
		if r.HasError() {
			hasErrors = true
		}
	}

	if hasErrors {
		return errors.New("invalid generated PackageManifest")
	}

	return nil
}

// setChannels checks for duplicate channels in pkg and sets the default channel if possible.
func setChannels(pkg *apimanifests.PackageManifest, channelName, csvName string) {
	channelIdx := -1
	for i, channel := range pkg.Channels {
		if channel.Name == channelName {
			pkg.Channels[i].CurrentCSVName = csvName
			channelIdx = i
			break
		}
	}
	if channelIdx == -1 {
		pkg.Channels = append(pkg.Channels, apimanifests.PackageChannel{
			Name:           channelName,
			CurrentCSVName: csvName,
		})
	}
}
