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

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
)

// CSVConfig is a configuration file for CSV composition. Its fields contain
// file path information.
// TODO(estroz): define field for path to write CSV bundle.
// TODO(estroz): make CSVConfig a viper.Config
type CSVConfig struct {
	// The operator manifest file path. Defaults to deploy/operator.yaml.
	OperatorPath string `json:"operator-path,omitempty"`
	// Role and ClusterRole manifest file paths. Defaults to [deploy/role.yaml].
	RolePaths []string `json:"role-paths,omitempty"`
	// A list of CRD and CR manifest file paths. Defaults to [deploy/crds].
	CRDCRPaths []string `json:"crd-cr-paths,omitempty"`
	// OperatorName is the name used to create the CSV and manifest file names.
	// Defaults to the project's name.
	OperatorName string `json:"operator-name,omitempty"`
}

// TODO: discuss case of no config file at default path: write new file or not.
func GetCSVConfig(cfgFile string) (*CSVConfig, error) {
	cfg := &CSVConfig{}
	if _, err := os.Stat(cfgFile); err == nil {
		cfgData, err := ioutil.ReadFile(cfgFile)
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

func (c *CSVConfig) setFields() error {
	if c.OperatorPath == "" {
		info, err := (&scaffold.Operator{}).GetInput()
		if err != nil {
			return err
		}
		c.OperatorPath = info.Path
	}

	if len(c.RolePaths) == 0 {
		info, err := (&scaffold.Role{}).GetInput()
		if err != nil {
			return err
		}
		c.RolePaths = []string{info.Path}
	}

	if len(c.CRDCRPaths) == 0 {
		paths, err := getManifestPathsFromDir(scaffold.CRDsDir)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if os.IsNotExist(err) {
			log.Infof("Default CRDs dir %s does not exist. Omitting field spec.customresourcedefinitions.owned from CSV.", scaffold.CRDsDir)
		} else if len(paths) == 0 {
			log.Infof("Default CRDs dir %s is empty. Omitting field spec.customresourcedefinitions.owned from CSV.", scaffold.CRDsDir)
		} else {
			c.CRDCRPaths = paths
		}
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
			} else if filepath.Ext(path) == yamlExt {
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
		if !info.IsDir() && filepath.Ext(path) == yamlExt {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
