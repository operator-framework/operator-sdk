// Copyright 2019 The Operator-SDK Authors
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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	registryutil "github.com/operator-framework/operator-sdk/internal/util/operator-registry"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const PackageManifestFileExt = ".package.yaml"

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
}

const ManifestsDir = "manifests"

func NewPackageManifest(cfg gen.Config, csvVersion, channel string, isDefault bool) gen.Generator {
	g := pkgGenerator{
		Config:           cfg,
		csvVersion:       csvVersion,
		channel:          channel,
		channelIsDefault: isDefault,
	}
	if g.Inputs == nil {
		g.Inputs = map[string]string{}
	}
	if manifests, ok := g.Inputs[ManifestsDir]; !ok || manifests == "" {
		g.Inputs[ManifestsDir] = filepath.Join(OLMCatalogDir, g.OperatorName)
	}
	if g.OutputDir == "" {
		g.OutputDir = filepath.Join(OLMCatalogDir, g.OperatorName)
	}
	return g
}

func getPkgFileName(operatorName string) string {
	return strings.ToLower(operatorName) + PackageManifestFileExt
}

func (g pkgGenerator) Generate() error {
	fileMap, err := g.generate()
	if err != nil {
		return err
	}
	if len(fileMap) == 0 {
		return errors.New("error generating package manifest: no generated file found")
	}
	if err = os.MkdirAll(g.OutputDir, fileutil.DefaultDirFileMode); err != nil {
		return errors.Wrapf(err, "error mkdir %s", g.OutputDir)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(g.OutputDir, fileName)
		if err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return err
		}
	}
	return nil
}

// generate either reads an existing package manifest or creates a new
// manifest and modifies it based on values set in s.
func (g pkgGenerator) generate() (map[string][]byte, error) {
	fileName := getPkgFileName(g.OperatorName)
	path := filepath.Join(g.Inputs[ManifestsDir], fileName)
	pkg := registry.PackageManifest{}
	if _, err := os.Stat(path); err == nil {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read package manifest %s", path)
		}
		if err = yaml.Unmarshal(b, &pkg); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal package manifest %s", path)
		}
		if err := registryutil.ValidatePackageManifest(&pkg); err != nil {
			return nil, errors.Wrapf(err, "error validating package manifest %s", pkg.PackageName)
		}
	} else if os.IsNotExist(err) {
		pkg = newPackageManifest(g.OperatorName, g.channel, g.csvVersion)
	} else {
		return nil, errors.Wrapf(err, "error reading package manifest %s", path)
	}

	g.setChannels(&pkg)
	sort.Slice(pkg.Channels, func(i int, j int) bool {
		return pkg.Channels[i].Name < pkg.Channels[j].Name
	})

	b, err := yaml.Marshal(pkg)
	if err != nil {
		return nil, err
	}
	fileMap := map[string][]byte{
		fileName: b,
	}
	return fileMap, nil
}

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

	if pkg.DefaultChannelName == "" {
		log.Warn("Package manifest default channel is empty and should be set to an existing channel.")
	} else {
		defaultExists := false
		for _, c := range pkg.Channels {
			if pkg.DefaultChannelName == c.Name {
				defaultExists = true
			}
		}
		if !defaultExists {
			log.Warnf("Package manifest default channel %s does not exist in channels.", pkg.DefaultChannelName)
		}
	}
}
