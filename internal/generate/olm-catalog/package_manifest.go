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

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
)

const (
	packageManifestFileExt = ".package.yaml"
)

type PkgGenerator struct {
	OperatorName string
	OutputDir    string
	// csvVersion is the version of the CSV being updated.
	CSVVersion string
	// channel is csvVersion's package manifest channel. If a new package
	// manifest is generated, this channel will be the manifest default.
	Channel string
	// If channelIsDefault is true, channel will be the package manifests'
	// default channel.
	ChannelIsDefault bool
	// PackageManifest file name
	fileName string
}

// getPkgFileName will return the name of the PackageManifestFile
func getPkgFileName(operatorName string) string {
	return strings.ToLower(operatorName) + packageManifestFileExt
}

func (g *PkgGenerator) setDefaults() {
	// The olm-catalog directory location depends on where the output directory is set.
	if g.OutputDir == "" {
		g.OutputDir = scaffold.DeployDir
	}
	g.fileName = getPkgFileName(g.OperatorName)
}

func (g PkgGenerator) Generate() error {
	g.setDefaults()
	fileMap, err := g.generate()
	if err != nil {
		return err
	}
	if len(fileMap) == 0 {
		return errors.New("error generating package manifest: no generated file found")
	}
	pkgManifestOutputDir := filepath.Join(g.OutputDir, OLMCatalogChildDir, g.OperatorName)
	if err = os.MkdirAll(pkgManifestOutputDir, 0755); err != nil {
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
func (g PkgGenerator) generate() (map[string][]byte, error) {
	pkg, err := g.buildPackageManifest()
	if err != nil {
		return nil, err
	}

	g.setChannels(&pkg)
	sortChannelsByName(&pkg)

	if err := validatePackageManifest(&pkg); err != nil {
		return nil, err
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

// buildPackageManifest will create a apimanifests.PackageManifest from scratch, or reads
// an existing one if found at the expected path.
func (g PkgGenerator) buildPackageManifest() (apimanifests.PackageManifest, error) {
	pkgManifestOutputDir := filepath.Join(g.OutputDir, OLMCatalogChildDir, g.OperatorName)
	path := filepath.Join(pkgManifestOutputDir, g.fileName)
	pkg := apimanifests.PackageManifest{}
	if isExist(path) {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return pkg, fmt.Errorf("failed to read package manifest %s: %v", path, err)
		}
		if err = yaml.Unmarshal(b, &pkg); err != nil {
			return pkg, fmt.Errorf("failed to unmarshal package manifest %s: %v", path, err)
		}
	} else {
		pkg = newPackageManifest(g.OperatorName, g.Channel, g.CSVVersion)
	}
	return pkg, nil
}

// sortChannelsByName sorts pkg.Channels by each element's name.
// NOTE: sorting makes the channel order always consistent when appending new channels
func sortChannelsByName(pkg *apimanifests.PackageManifest) {
	sort.Slice(pkg.Channels, func(i int, j int) bool {
		return pkg.Channels[i].Name < pkg.Channels[j].Name
	})
}

// validatePackageManifest will validate pkg using the api validation library.
// More info: https://github.com/operator-framework/api
func validatePackageManifest(pkg *apimanifests.PackageManifest) error {
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

// newPackageManifest will return the apimanifests.PackageManifest populated.
func newPackageManifest(operatorName, channelName, version string) apimanifests.PackageManifest {
	// Take the current CSV version to be the "alpha" channel, as an operator
	// should only be designated anything more stable than "alpha" by a human.
	channel := "alpha"
	if channelName != "" {
		channel = channelName
	}
	lowerOperatorName := strings.ToLower(operatorName)
	pkg := apimanifests.PackageManifest{
		PackageName: lowerOperatorName,
		Channels: []apimanifests.PackageChannel{
			{Name: channel, CurrentCSVName: getCSVName(lowerOperatorName, version)},
		},
		DefaultChannelName: channel,
	}
	return pkg
}

// setChannels checks for duplicate channels in pkg and sets the default
// channel if possible.
func (g PkgGenerator) setChannels(pkg *apimanifests.PackageManifest) {
	if g.Channel != "" {
		channelIdx := -1
		for i, channel := range pkg.Channels {
			if channel.Name == g.Channel {
				channelIdx = i
				break
			}
		}
		lowerOperatorName := strings.ToLower(g.OperatorName)
		if channelIdx == -1 {
			pkg.Channels = append(pkg.Channels, apimanifests.PackageChannel{
				Name:           g.Channel,
				CurrentCSVName: getCSVName(lowerOperatorName, g.CSVVersion),
			})
		} else {
			pkg.Channels[channelIdx].CurrentCSVName = getCSVName(lowerOperatorName, g.CSVVersion)
		}
		// Use g.Channel as the default channel if caller has specified it as the
		// default.
		if g.ChannelIsDefault {
			pkg.DefaultChannelName = g.Channel
		}
	}
}
