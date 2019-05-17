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

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	yaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const DefaultFileName = ".osdk-config.yaml"

type SDKConfig struct {
	ConfigFile  string `json:"-"`
	ProjectName string `json:"-"`

	Repo          string `json:"repo"`
	DeployDir     string `json:"deploy-dir,omitempty"`
	CRDsDir       string `json:"crds-dir,omitempty"`
	APIsDir       string `json:"apis-dir,omitempty"`
	OLMCatalogDir string `json:"olm-catalog-dir"`
	Verbose       bool   `json:"verbose,omitempty"`
}

func (s *SDKConfig) Init() error {
	if err := SetDefaults(); err != nil {
		return nil
	}

	if s.Repo != "" {
		viper.Set(RepoOpt, s.Repo)
	} else {
		repo, err := GetGoPathRepo()
		if err != nil {
			return err
		}
		viper.Set(RepoOpt, repo)
	}
	if s.DeployDir != "" {
		viper.Set(DeployDirOpt, s.DeployDir)
	}
	if s.CRDsDir != "" {
		viper.Set(CRDsDirOpt, s.CRDsDir)
	}
	if s.OLMCatalogDir != "" {
		viper.Set(OLMCatalogDirOpt, s.OLMCatalogDir)
	}
	if s.APIsDir != "" {
		viper.Set(APIsDirOpt, s.APIsDir)
	}
	viper.Set(VerboseOpt, s.Verbose)
	return nil
}

func SetDefaults() error {
	// Use strings to avoid import cycles.
	viper.SetDefault(DeployDirOpt, "deploy")
	viper.SetDefault(CRDsDirOpt, filepath.Join("deploy", "crds"))
	viper.SetDefault(OLMCatalogDirOpt, filepath.Join("deploy", "olm-catalog"))
	viper.SetDefault(APIsDirOpt, filepath.Join("pkg", "apis"))
	return nil
}

var ErrNoRepo = errors.New("repo not set in config and project not in $GOPATH/src")

func CheckRepo() error {
	if viper.GetString(RepoOpt) != "" {
		return nil
	}
	inGoPathSrc, err := projutil.WdInGoPathSrc()
	if err == nil && !inGoPathSrc {
		return ErrNoRepo
	}
	return err
}

func GetGoPathRepo() (string, error) {
	if err := CheckRepo(); err != nil {
		return "", err
	}
	goPath := os.Getenv(projutil.GoPathEnv)
	if goPath == "" {
		hd, err := projutil.GetHomeDir()
		if err != nil {
			return "", err
		}
		goPath = filepath.Join(hd, "go")
	}
	return getGoPkgFromGoPath(goPath)
}

func getGoPkgFromGoPath(gopath string) (string, error) {
	goSrc := filepath.Join(gopath, "src")
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	currPkg := strings.Replace(wd, goSrc, "", 1)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(currPkg, string(filepath.Separator)), nil
}

func WriteConfigAs(path string) error {
	// A bug in SafeWriteConfig() prevents a new config from being written.
	// Once PR https://github.com/spf13/viper/pull/450 is merged, we can directly
	// use that function.
	b, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		return errors.Wrap(err, "marshal config")
	}
	if err := ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
		return errors.Wrapf(err, "write config")
	}
	log.Infof("Created config file %s", path)
	return nil
}
