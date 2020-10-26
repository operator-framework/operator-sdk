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
	errNoGetBase   = genutil.InternalError("getBase must be set")
	errNoGetWriter = genutil.InternalError("getWriter must be set")
)

type Generator struct {
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

	// Func that returns a base PackageManifest.
	getBase getBaseFunc
	// Func that returns the writer the generated PackageManifest's bytes are written to.
	getWriter func() (io.Writer, error)
}

// Type of Generator.getBase.
type getBaseFunc func() (*apimanifests.PackageManifest, error)

// Option is a function that modifies a Generator.
type Option func(*Generator) error

// WithBase sets a Generator's base PackageManifest to one that either exists or a default.
func WithBase(inputDir string) Option {
	return func(g *Generator) error {
		g.getBase = g.makeBaseGetter(inputDir)
		return nil
	}
}

// WithWriter sets a Generator's writer to w.
func WithWriter(w io.Writer) Option {
	return func(g *Generator) error {
		g.getWriter = func() (io.Writer, error) {
			return w, nil
		}
		return nil
	}
}

// WithFileWriter sets a Generator's writer to a PackageManifest file under <dir>.
func WithFileWriter(dir string) Option {
	return func(g *Generator) (err error) {
		g.getWriter = func() (io.Writer, error) {
			return genutil.Open(dir, makePkgManFileName(g.OperatorName))
		}
		return nil
	}
}

// Generate configures the Generator with opts then runs it.
func (g *Generator) Generate(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return err
		}
	}

	if g.getWriter == nil {
		return errNoGetWriter
	}

	pkg, err := g.generate()
	if err != nil {
		return err
	}

	w, err := g.getWriter()
	if err != nil {
		return err
	}
	return genutil.WriteYAML(w, pkg)
}

// generate runs a configured Generator.
func (g *Generator) generate() (*apimanifests.PackageManifest, error) {
	if g.getBase == nil {
		return nil, errNoGetBase
	}
	if g.Version == "" {
		return nil, errNoVersion
	}

	base, err := g.getBase()
	if err != nil {
		return nil, fmt.Errorf("error getting PackageManifest base: %v", err)
	}

	csvName := genutil.MakeCSVName(g.OperatorName, g.Version)
	if g.ChannelName != "" {
		setChannels(base, g.ChannelName, csvName)
		sortChannelsByName(base)
		if g.IsDefaultChannel || len(base.Channels) == 1 {
			base.DefaultChannelName = g.ChannelName
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

// makeBaseGetter returns a function that gets a base from inputDir.
func (g Generator) makeBaseGetter(inputDir string) func() (*apimanifests.PackageManifest, error) {
	basePath := filepath.Join(inputDir, makePkgManFileName(g.OperatorName))
	if genutil.IsNotExist(basePath) {
		basePath = ""
	}

	return func() (*apimanifests.PackageManifest, error) {
		b := bases.PackageManifest{
			PackageName: g.OperatorName,
			BasePath:    basePath,
		}
		return b.GetBase()
	}
}

// makePkgManFileName will return the file name of a PackageManifest.
func makePkgManFileName(operatorName string) string {
	return operatorName + packageManifestFileExt
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
