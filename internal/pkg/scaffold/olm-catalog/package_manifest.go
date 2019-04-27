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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"

	"github.com/ghodss/yaml"
	olmregistry "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const PackageManifestPrefix = ".package.yaml"

type PackageManifest struct {
	input.Input

	// CSVVersion is the version of the CSV being updated.
	CSVVersion string
	// Channel is CSVVersion's package manifest channel. If a new package
	// manifest is generated, this channel will be the manifest default.
	Channel string
}

var _ input.File = &PackageManifest{}

func (s *PackageManifest) GetInput() (input.Input, error) {
	if s.Path == "" {
		lowerProjName := strings.ToLower(s.ProjectName)
		// Path is what the operator-registry expects:
		// {manifests -> olm-catalog}/{operator_name}/{operator_name}.package.yaml
		s.Path = filepath.Join(OLMCatalogDir, lowerProjName,
			lowerProjName+PackageManifestPrefix)
	}
	return s.Input, nil
}

var _ scaffold.CustomRenderer = &PackageManifest{}

func (s *PackageManifest) SetFS(_ afero.Fs) {}

func (s *PackageManifest) CustomRender() ([]byte, error) {
	i, err := s.GetInput()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.AbsProjectPath, i.Path)

	pm := s.newPackageManifest()
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "package manifest %s", path)
	} else if err == nil {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "read package manifest %s", path)
		}
		if len(b) > 0 {
			if err = yaml.Unmarshal(b, pm); err != nil {
				return nil, errors.Wrapf(err, "unmarshal package manifest %s", path)
			}
		}
	}

	if s.Channel != "" {
		channelMap := map[string]string{}
		for _, c := range pm.Channels {
			if _, ok := channelMap[c.Name]; ok {
				return nil, fmt.Errorf(`duplicate package manifest channel name "%s"; channel names must be unique`, c.Name)
			}
			channelMap[c.Name] = c.CurrentCSVName
		}
		channelMap[s.Channel] = getCSVName(s.ProjectName, s.CSVVersion)
		channels := []olmregistry.PackageChannel{}
		for n, cn := range channelMap {
			channels = append(channels, olmregistry.PackageChannel{
				Name:           n,
				CurrentCSVName: cn,
			})
		}
		pm.Channels = channels
	}

	sort.Slice(pm.Channels, func(i int, j int) bool {
		return pm.Channels[i].Name < pm.Channels[j].Name
	})

	return yaml.Marshal(pm)
}

func (s *PackageManifest) newPackageManifest() *olmregistry.PackageManifest {
	// Take the current version to be the "alpha" channel, as an operator
	// should be designated anything greater than "alpha" by a human.
	channel := "alpha"
	if s.Channel != "" {
		channel = s.Channel
	}
	pm := &olmregistry.PackageManifest{
		PackageName: s.ProjectName,
		Channels: []olmregistry.PackageChannel{
			{Name: channel, CurrentCSVName: getCSVName(s.ProjectName, s.CSVVersion)},
		},
		DefaultChannelName: channel,
	}
	return pm
}
