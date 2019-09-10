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

package catalog

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	registryutil "github.com/operator-framework/operator-sdk/internal/util/operator-registry"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const PackageManifestFileExt = ".package.yaml"

type PackageManifest struct {
	input.Input

	// CSVVersion is the version of the CSV being updated.
	CSVVersion string
	// Channel is CSVVersion's package manifest channel. If a new package
	// manifest is generated, this channel will be the manifest default.
	Channel string
	// If ChannelIsDefault is true, Channel will be the package manifests'
	// default channel.
	ChannelIsDefault bool
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
}

var _ input.File = &PackageManifest{}

// GetInput gets s' Input.
func (s *PackageManifest) GetInput() (input.Input, error) {
	if s.Path == "" {
		lowerOperatorName := strings.ToLower(s.OperatorName)
		// Path is what the operator-registry expects:
		// {manifests -> olm-catalog}/{operator_name}/{operator_name}.package.yaml
		s.Path = filepath.Join(OLMCatalogDir, lowerOperatorName,
			lowerOperatorName+PackageManifestFileExt)
	}
	return s.Input, nil
}

var _ scaffold.CustomRenderer = &PackageManifest{}

// SetFS is a no-op to implement CustomRenderer.
func (s *PackageManifest) SetFS(_ afero.Fs) {}

// CustomRender either reads an existing package manifest or creates a new
// manifest and modifies it based on values set in s.
func (s *PackageManifest) CustomRender() ([]byte, error) {
	i, err := s.GetInput()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.AbsProjectPath, i.Path)

	pm := &registry.PackageManifest{}
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		pm = s.newPackageManifest()
	} else if err == nil {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read package manifest %s", path)
		}
		if len(b) > 0 {
			if err = yaml.Unmarshal(b, pm); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal package manifest %s", path)
			}
		} else {
			// File exists but is empty.
			pm = s.newPackageManifest()
		}
	} else {
		return nil, errors.Wrapf(err, "package manifest %s", path)
	}

	if err := registryutil.ValidatePackageManifest(pm); err != nil {
		return nil, errors.Wrapf(err, "failed to validate package manifest %s", pm.PackageName)
	}

	if err = s.setChannels(pm); err != nil {
		return nil, err
	}

	sort.Slice(pm.Channels, func(i int, j int) bool {
		return pm.Channels[i].Name < pm.Channels[j].Name
	})

	return yaml.Marshal(pm)
}

func (s *PackageManifest) newPackageManifest() *registry.PackageManifest {
	// Take the current CSV version to be the "alpha" channel, as an operator
	// should only be designated anything more stable than "alpha" by a human.
	channel := "alpha"
	if s.Channel != "" {
		channel = s.Channel
	}
	lowerOperatorName := strings.ToLower(s.OperatorName)
	pm := &registry.PackageManifest{
		PackageName: lowerOperatorName,
		Channels: []registry.PackageChannel{
			{Name: channel, CurrentCSVName: getCSVName(lowerOperatorName, s.CSVVersion)},
		},
		DefaultChannelName: channel,
	}
	return pm
}

// setChannels checks for duplicate channels in pm and sets the default
// channel if possible.
func (s *PackageManifest) setChannels(pm *registry.PackageManifest) error {
	if s.Channel != "" {
		channelIdx := -1
		for i, channel := range pm.Channels {
			if channel.Name == s.Channel {
				channelIdx = i
				break
			}
		}
		lowerOperatorName := strings.ToLower(s.OperatorName)
		if channelIdx == -1 {
			pm.Channels = append(pm.Channels, registry.PackageChannel{
				Name:           s.Channel,
				CurrentCSVName: getCSVName(lowerOperatorName, s.CSVVersion),
			})
		} else {
			pm.Channels[channelIdx].CurrentCSVName = getCSVName(lowerOperatorName, s.CSVVersion)
		}
		// Use s.Channel as the default channel if caller has specified it as the
		// default.
		if s.ChannelIsDefault {
			pm.DefaultChannelName = s.Channel
		}
	}

	if pm.DefaultChannelName == "" {
		log.Warn("Package manifest default channel is empty and should be set to an existing channel.")
	}
	defaultExists := false
	for _, c := range pm.Channels {
		if pm.DefaultChannelName == c.Name {
			defaultExists = true
		}
	}
	if !defaultExists {
		log.Warnf("Package manifest default channel %s does not exist in channels.", pm.DefaultChannelName)
	}

	return nil
}
