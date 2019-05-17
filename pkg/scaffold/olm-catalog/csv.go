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
	"strings"

	"github.com/Masterminds/semver"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	catalog "github.com/operator-framework/operator-sdk/internal/pkg/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

type GenCSVCmd struct {
	OperatorPath string
	RolePath     string
	CRDCRPaths   []string
	CSVVersion   string
	FromVersion  string
	UpdateCRDs   bool
}

func (c *GenCSVCmd) Run() error {

	if err := c.verifyFlags(); err != nil {
		return err
	}
	if err := c.setInGlobal(); err != nil {
		return err
	}

	log.Infof("Generating CSV manifest version %s", c.CSVVersion)

	absProjectPath := projutil.MustGetwd()
	s := &scaffold.Scaffold{
		Repo:           viper.GetString(config.RepoOpt),
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	csv := &catalog.CSV{
		CSVVersion:  c.CSVVersion,
		FromVersion: c.FromVersion,
		IncludeManifestPaths: append(viper.GetStringSlice(catalog.CRDCRPathsOpt),
			viper.GetString(catalog.OperatorPathOpt),
			viper.GetString(catalog.RolePathOpt)),
	}
	if err := s.Execute(csv); err != nil {
		return fmt.Errorf("catalog scaffold failed: (%v)", err)
	}

	// Write CRD's to the new or updated CSV package dir.
	if c.UpdateCRDs {
		input, err := csv.GetInput()
		if err != nil {
			return err
		}
		err = writeCRDsToDir(c.CRDCRPaths, filepath.Dir(input.Path))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *GenCSVCmd) setInGlobal() error {
	if err := c.expandPaths(); err != nil {
		return err
	}
	if c.OperatorPath != "" {
		viper.Set(catalog.OperatorPathOpt, c.OperatorPath)
	}
	if c.RolePath != "" {
		viper.Set(catalog.RolePathOpt, c.RolePath)
	}
	if len(c.CRDCRPaths) != 0 {
		viper.Set(catalog.CRDCRPathsOpt, c.CRDCRPaths)
	}
	return nil
}

// expandPaths finds all manifest files in fields of c that can contain dirs
// and expands them. Some operators only have "required" CRDs, which will
// not be present locally on disk; an info message is generated if there are
// no c.CRDCRPaths on disk.
func (c *GenCSVCmd) expandPaths() (err error) {
	// Use defaults if unset.
	if len(c.CRDCRPaths) == 0 {
		c.CRDCRPaths = viper.GetStringSlice(catalog.CRDCRPathsOpt)
	}
	// Allow user to specify a list of dirs to search. Avoid duplicate files.
	paths := []string{}
	seen := map[string]struct{}{}
	for _, path := range c.CRDCRPaths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Infof(`CRD/CR path "%s" does not exist. Omitting spec.customresourcedefinitions.owned element for this manifest from CSV.`, path)
				continue
			}
			return err
		}
		if info.IsDir() {
			manifests, err := getManifestPathsFromDir(path)
			if err != nil {
				return err
			}
			for _, p := range manifests {
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
	if len(paths) == 0 {
		log.Info("No CRD/CR manifests found. Omitting field spec.customresourcedefinitions.owned from CSV.")
		viper.Set(catalog.CRDCRPathsOpt, []string{})
	}
	c.CRDCRPaths = paths
	return nil
}

func (c *GenCSVCmd) verifyFlags() error {
	if err := verifyCSVVersion(c.CSVVersion); err != nil {
		return err
	}
	if c.FromVersion != "" {
		if err := verifyCSVVersion(c.FromVersion); err != nil {
			return err
		}
	}
	if c.FromVersion != "" && c.CSVVersion == c.FromVersion {
		return fmt.Errorf("from-version (%s) cannot equal csv-version; set only csv-version instead", c.FromVersion)
	}

	if viper.GetString(catalog.RolePathOpt) == "" {
		return fmt.Errorf("role RBAC manifest path must be set")
	}
	if viper.GetString(catalog.OperatorPathOpt) == "" {
		return fmt.Errorf("operator Deployment manifest path must be set")
	}
	return nil
}

func verifyCSVVersion(version string) error {
	v, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("%s is not a valid semantic version: (%v)", version, err)
	}
	// Ensures numerical values composing CSVVersion don't contain leading 0's,
	// ex. 01.01.01
	if v.String() != version {
		return fmt.Errorf("provided CSV version %s contains bad values (parses to %s)", version, v)
	}
	return nil
}

func writeCRDsToDir(crdPaths []string, toDir string) error {
	for _, p := range crdPaths {
		if !strings.HasSuffix(p, "crd.yaml") {
			continue
		}
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		path := filepath.Join(toDir, filepath.Base(p))
		err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode)
		if err != nil {
			return err
		}
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

func isYAMLFile(path string) bool {
	return filepath.Ext(path) == ".yaml"
}
