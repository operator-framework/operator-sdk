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

package packagemanifest

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/operator-registry/pkg/registry"
	log "github.com/sirupsen/logrus"

	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest/bases"
)

// Deprecated: The package manifest generator will no longer create new package
// manifests, only update existing ones. This generator will be removed in v0.19.0.

const (
	packageManifestFileExt = ".package.yaml"
)

type Generator struct {
	//
	OperatorName string
	// version is the version of the CSV being updated.
	Version string
	// channel is operator's package manifest channel. If a new package
	// manifest is generated, this channel will be the manifest default.
	ChannelName string
	// If channelIsDefault is true, channel will be the package manifests'
	// default channel.
	IsDefaultChannel bool

	//
	getBase func() (*registry.PackageManifest, error)
	//
	getWriter func() (io.Writer, error)
}

type Option func(*Generator) error

func WithGetBase(inputDir string) Option {
	return func(g *Generator) error {
		g.getBase = g.makeBaseGetter(inputDir)
		return nil
	}
}

func WithFileWriter(dir string) Option {
	return func(g *Generator) (err error) {
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(dir, getPackageManifestFile(g.OperatorName))
		}
		return nil
	}
}

func WithWriter(w io.Writer) Option {
	return func(g *Generator) error {
		g.getWriter = func() (io.Writer, error) {
			return w, nil
		}
		return nil
	}
}

func (g *Generator) Generate(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return err
		}
	}

	return g.generate()
}

func (g *Generator) generate() error {
	if g.getWriter == nil {
		return genutil.InternalError("getWriter must be set")
	}
	if g.getBase == nil {
		return genutil.InternalError("getBase must be set")
	}

	base, err := g.getBase()
	if err != nil {
		return fmt.Errorf("error getting PackageManifest base: %v", err)
	}

	csvName := genutil.GetCSVName(g.OperatorName, g.Version)
	if g.ChannelName != "" {
		setChannels(base, g.ChannelName, csvName)
		sortChannelsByName(base)
		if g.IsDefaultChannel || len(base.Channels) == 0 {
			base.DefaultChannelName = g.ChannelName
		}
	} else if len(base.Channels) == 0 {
		setChannels(base, "alpha", csvName)
		base.DefaultChannelName = "alpha"
	}

	if err := validatePackageManifest(base); err != nil {
		return err
	}

	w, err := g.getWriter()
	if err != nil {
		return err
	}
	return genutil.WriteYAML(w, base)
}

func (g Generator) makeBaseGetter(inputDir string) func() (*registry.PackageManifest, error) {
	basePath := filepath.Join(inputDir, getPackageManifestFile(g.OperatorName))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}

	return func() (*registry.PackageManifest, error) {
		b := bases.PackageManifest{
			PackageName: g.OperatorName,
			BasePath:    basePath,
		}
		return b.GetBase()
	}
}

// getPackageManifestFile will return the file name of a PackageManifest.
func getPackageManifestFile(operatorName string) string {
	return strings.ToLower(operatorName) + packageManifestFileExt
}

// sortChannelsByName sorts pkg.Channels by each element's name.
// NOTE: sorting makes the channel order always consistent when appending new channels
func sortChannelsByName(pkg *registry.PackageManifest) {
	sort.Slice(pkg.Channels, func(i int, j int) bool {
		return pkg.Channels[i].Name < pkg.Channels[j].Name
	})
}

// validatePackageManifest will validate pkg using the api validation library.
// More info: https://github.com/operator-framework/api
func validatePackageManifest(pkg *registry.PackageManifest) error {
	if pkg == nil {
		return errors.New("empty PackageManifest")
	}
	results := validation.PackageManifestValidator.Validate(pkg)
	for _, r := range results {
		if r.HasError() {
			for _, e := range r.Errors {
				log.Errorf("PackageManifest validation: [%s] %s", e.Type, e.Detail)
			}
			return errors.New("got PackageManifest validation errors")
		}
		for _, w := range r.Warnings {
			log.Warnf("PackageManifest validation: [%s] %s", w.Type, w.Detail)
		}
	}
	return nil
}

// setChannels checks for duplicate channels in pkg and sets the default
// channel if possible.
func setChannels(pkg *registry.PackageManifest, channelName, csvName string) {
	channelIdx := -1
	for i, channel := range pkg.Channels {
		if channel.Name == channelName {
			pkg.Channels[i].CurrentCSVName = csvName
			channelIdx = i
			break
		}
	}
	if channelIdx == -1 {
		pkg.Channels = append(pkg.Channels, registry.PackageChannel{
			Name:           channelName,
			CurrentCSVName: csvName,
		})
	}
}
