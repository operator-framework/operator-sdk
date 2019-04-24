// Copyright 2018 The Operator-SDK Authors
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

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"

	"github.com/ghodss/yaml"
	"github.com/spf13/afero"
)

const CSVConfigYamlFile = "csv-config.yaml"

// CSVConfig is a scaffold wrapper for CSVConfig such that CSVConfig's can
// be marshalled and unmarshalled correctly.
type CSVConfig struct {
	input.Input
	File *CSVConfigFile

	// TODO: remove once fs is moved to input.Input.
	fs afero.Fs
}

var _ input.File = &CSVConfig{}

func (c *CSVConfig) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join(OLMCatalogDir, CSVConfigYamlFile)
	}
	if c.File == nil {
		c.File = &CSVConfigFile{}
	}
	return c.Input, nil
}

var _ scaffold.CustomRenderer = &CSVConfig{}

func (c *CSVConfig) SetFS(fs afero.Fs) { c.fs = fs }

func (c *CSVConfig) CustomRender() ([]byte, error) {
	if err := c.File.setFields(); err != nil {
		return nil, err
	}
	return yaml.Marshal(c.File)
}

// CSVConfigFile is a configuration file for CSV composition. Its fields contain
// file path information.
type CSVConfigFile struct {
	// The operator manifest file path. Defaults to deploy/operator.yaml.
	OperatorPath string `json:"operator-path,omitempty"`
	// The RBAC role manifest file path. Defaults to deploy/role.yaml.
	RolePath string `json:"role-path,omitempty"`
	// A list of CRD and CR manifest file paths. Defaults to deploy/crds.
	CRDCRPaths []string `json:"crd-cr-paths,omitempty"`
}

func GetCSVConfigFile(path string) (*CSVConfigFile, error) {
	cfg := &CSVConfigFile{}
	if _, err := os.Stat(path); err == nil {
		cfgData, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := cfg.setFields(); err != nil {
		return nil, err
	}
	return cfg, nil
}

const yamlExt = ".yaml"

func isYAMLFile(path string) bool {
	return filepath.Ext(path) == yamlExt
}

func (c *CSVConfigFile) setFields() error {
	if c.OperatorPath == "" {
		info, err := (&scaffold.Operator{}).GetInput()
		if err != nil {
			return err
		}
		c.OperatorPath = info.Path
	}

	if c.RolePath == "" {
		info, err := (&scaffold.Role{}).GetInput()
		if err != nil {
			return err
		}
		c.RolePath = info.Path
	}

	if len(c.CRDCRPaths) == 0 {
		paths, err := getManifestPathsFromDir(scaffold.CRDsDir)
		if err != nil {
			return err
		}
		c.CRDCRPaths = paths
	} else {
		// Allow user to specify a list of dirs to search. Avoid duplicate files.
		paths, seen := make([]string, 0), make(map[string]struct{})
		for _, path := range c.CRDCRPaths {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				tmpPaths, err := getManifestPathsFromDir(path)
				if err != nil {
					return err
				}
				for _, p := range tmpPaths {
					if _, ok := seen[p]; !ok {
						paths = append(paths, p)
						seen[p] = struct{}{}
					}
				}
			} else if isYAMLFile(path) {
				if _, ok := seen[path]; !ok {
					paths = append(paths, path)
					seen[path] = struct{}{}
				}
			}
		}
		c.CRDCRPaths = paths
	}

	return nil
}

func getManifestPathsFromDir(dir string) (paths []string, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return fmt.Errorf("file info for %s was nil", path)
		}
		if !info.IsDir() && isYAMLFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
