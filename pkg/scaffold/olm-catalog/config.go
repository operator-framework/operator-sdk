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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/ghodss/yaml"
)

// CsvConfig is a configuration file for CSV composition. Its fields contain
// file path information.
type CsvConfig struct {
	OperatorPath string   `json:"operator-path,omitempty"`
	CrdCrPaths   []string `json:"crd-cr-paths,omitempty"`
	RolePath     string   `json:"role-path,omitempty"`
}

func getCSVConfig(configFile string) (*CsvConfig, error) {
	config := &CsvConfig{}
	if _, err := os.Stat(configFile); err == nil {
		configData, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(configData, config); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := config.setFields(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *CsvConfig) setFields() error {
	if c.OperatorPath == "" {
		info, err := (&scaffold.Operator{}).GetInput()
		if err != nil {
			return err
		}
		c.OperatorPath = info.Path
	}

	if len(c.CrdCrPaths) == 0 {
		paths, err := getManifestPathsFromDir(scaffold.CrdsDir)
		if err != nil {
			return err
		}
		c.CrdCrPaths = paths
	} else {
		// Allow user to specify a list of dirs to search. Avoid duplicate files.
		paths, seen := make([]string, 0), make(map[string]struct{})
		for _, path := range c.CrdCrPaths {
			finfo, err := os.Stat(path)
			if err != nil {
				return err
			}
			if finfo.IsDir() {
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
			} else {
				if _, ok := seen[path]; !ok {
					paths = append(paths, path)
					seen[path] = struct{}{}
				}
			}
		}
		c.CrdCrPaths = paths
	}

	if c.RolePath == "" {
		path, _ := (&scaffold.Role{}).GetInput()
		c.RolePath = path.Path
	}

	return nil
}

func getManifestPathsFromDir(dir string) (paths []string, err error) {
	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, finfo := range finfos {
		if finfo != nil && !finfo.IsDir() {
			paths = append(paths, filepath.Join(dir, finfo.Name()))
		}
	}
	return paths, nil
}
