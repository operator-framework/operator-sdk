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

package olmcatalog

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
)

const (
	packageManifestFileExt = ".package.yaml"
)

type pkgGenerator struct {
	gen.Config
	// csvVersion is the version of the CSV being updated.
	csvVersion string
	// channel is csvVersion's package manifest channel. If a new package
	// manifest is generated, this channel will be the manifest default.
	channel string
	// If channelIsDefault is true, channel will be the package manifests'
	// default channel.
	channelIsDefault bool
	// PackageManifest file name
	fileName string
}

func NewPackageManifest(cfg gen.Config, csvVersion, channel string, isDefault bool) gen.Generator {
	g := pkgGenerator{
		Config:           cfg,
		csvVersion:       csvVersion,
		channel:          channel,
		channelIsDefault: isDefault,
		fileName:         getPkgFileName(cfg.OperatorName),
	}

	// Pkg manifest generator has no defined inputs
	g.Inputs = map[string]string{}

	// The olm-catalog directory location depends on where the output directory is set.
	if g.OutputDir == "" {
		g.OutputDir = scaffold.DeployDir
	}

	return g
}

func isFileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// TODO: return and handle this error
		log.Fatalf("Failed to stat %s: %v", path, err)
	}
	return true
}

// getPkgFileName will return the name of the PackageManifestFile
func getPkgFileName(operatorName string) string {
	return strings.ToLower(operatorName) + packageManifestFileExt
}

func (g pkgGenerator) Generate() error {
	fileMap, err := g.generate()
	if err != nil {
		return err
	}
	if len(fileMap) == 0 {
		return errors.New("error generating package manifest: no generated file found")
	}
	pkgManifestOutputDir := filepath.Join(g.OutputDir, OLMCatalogChildDir, g.OperatorName)
	if err = os.MkdirAll(pkgManifestOutputDir, fileutil.DefaultDirFileMode); err != nil {
		return fmt.Errorf("error mkdir %s: %v", pkgManifestOutputDir, err)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(pkgManifestOutputDir, fileName)
		log.Debugf("Package manifest generator writing %s", path)
		if err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return err
		}
	}
	return nil
}

// generate either reads an existing package manifest or creates a new
// manifest and modifies it based on values set in s.
func (g pkgGenerator) generate() (map[string][]byte, error) {
	pkg, err := g.buildPackageManifest()
	if err != nil {
		return nil, err
	}

	g.setChannels(&pkg)
	sortChannelsByName(&pkg)

	if err := validatePackageManifest(&pkg); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	b, err := yaml.Marshal(pkg)
	if err != nil {
		return nil, err
	}

	fileMap := map[string][]byte{
		g.fileName: b,
	}
	return fileMap, nil
}

// buildPackageManifest will create a registry.PackageManifest from scratch, or reads
// an existing one if found at the expected path.
func (g pkgGenerator) buildPackageManifest() (registry.PackageManifest, error) {
	pkg := registry.PackageManifest{}
	olmCatalogDir := filepath.Join(g.OutputDir, OLMCatalogChildDir)
	existingPkgManifest := filepath.Join(olmCatalogDir, g.OperatorName, g.fileName)
	if isFileExist(existingPkgManifest) {
		b, err := ioutil.ReadFile(existingPkgManifest)
		if err != nil {
			return pkg, fmt.Errorf("failed to read package manifest %s: %v", existingPkgManifest, err)
		}
		if err = yaml.Unmarshal(b, &pkg); err != nil {
			return pkg, fmt.Errorf("failed to unmarshal package manifest %s: %v", existingPkgManifest, err)
		}
	} else {
		pkg = newPackageManifest(g.OperatorName, g.channel, g.csvVersion)
	}
	return pkg, nil
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
		return errors.New("generated package manifest is empty")
	}
	results := validation.PackageManifestValidator.Validate(pkg)
	for _, r := range results {
		if r.HasError() {
			var errorMsgs strings.Builder
			for _, e := range r.Errors {
				errorMsgs.WriteString(fmt.Sprintf("%s\n", e.Error()))
			}
			return fmt.Errorf("error validating package manifest: %s", errorMsgs.String())
		}
		for _, w := range r.Warnings {
			log.Warnf("Package manifest validation warning: type [%s] %s", w.Type, w.Detail)
		}
	}
	return nil
}

// newPackageManifest will return the registry.PackageManifest populated
func newPackageManifest(operatorName, channelName, version string) registry.PackageManifest {
	// Take the current CSV version to be the "alpha" channel, as an operator
	// should only be designated anything more stable than "alpha" by a human.
	channel := "alpha"
	if channelName != "" {
		channel = channelName
	}
	lowerOperatorName := strings.ToLower(operatorName)
	pkg := registry.PackageManifest{
		PackageName: lowerOperatorName,
		Channels: []registry.PackageChannel{
			{Name: channel, CurrentCSVName: getCSVName(lowerOperatorName, version)},
		},
		DefaultChannelName: channel,
	}
	return pkg
}

// setChannels checks for duplicate channels in pkg and sets the default
// channel if possible.
func (g pkgGenerator) setChannels(pkg *registry.PackageManifest) {
	if g.channel != "" {
		channelIdx := -1
		for i, channel := range pkg.Channels {
			if channel.Name == g.channel {
				channelIdx = i
				break
			}
		}
		lowerOperatorName := strings.ToLower(g.OperatorName)
		if channelIdx == -1 {
			pkg.Channels = append(pkg.Channels, registry.PackageChannel{
				Name:           g.channel,
				CurrentCSVName: getCSVName(lowerOperatorName, g.csvVersion),
			})
		} else {
			pkg.Channels[channelIdx].CurrentCSVName = getCSVName(lowerOperatorName, g.csvVersion)
		}
		// Use g.channel as the default channel if caller has specified it as the
		// default.
		if g.channelIsDefault {
			pkg.DefaultChannelName = g.channel
		}
	}
}
