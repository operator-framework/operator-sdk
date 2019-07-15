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

package genutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// OpenshiftGen generates all the dirs and manifest that are openshift specifc under deploy/openshift.
func OpenshiftGen() error {
	projutil.MustInProjectRoot()
	absProjectPath := projutil.MustGetwd()
	openshiftAbsPath := fmt.Sprintf("%s/deploy/openshift", absProjectPath)
	repo := projutil.GetGoPkg()
	deployDir := fmt.Sprintf("%s/deploy/", absProjectPath)

	// check if dir openshift already exists
	if _, err := os.Stat(openshiftAbsPath); err == nil || os.IsExist(err) {
		log.Info("The deploy/openshift directory already exists, remove it first to continue")
		return err
	}

	// create directories
	if err := makeDirs([]string{openshiftAbsPath,
		fmt.Sprintf("%s/crds", openshiftAbsPath),
		fmt.Sprintf("%s/metrics", openshiftAbsPath),
		fmt.Sprintf("%s/rbac", openshiftAbsPath)}); err != nil {
		return fmt.Errorf("failed to create directory: %s", err)
	}

	// copy over crds dir files to openshift/crds/ dir
	fs := afero.NewOsFs()
	if err := copyDirs(fs, fmt.Sprintf("%scrds", deployDir),
		fmt.Sprintf("%s/crds", openshiftAbsPath)); err != nil {
		return fmt.Errorf("failed to create openshift/crds directory: %s", err)
	}
	if err := copyFiles(fmt.Sprintf("%srole.yaml", deployDir),
		deployDir, fmt.Sprintf("%s/rbac/", openshiftAbsPath), fs); err != nil {
		return err
	}
	if err := copyFiles(fmt.Sprintf("%srole_binding.yaml", deployDir),
		deployDir, fmt.Sprintf("%s/rbac/", openshiftAbsPath), fs); err != nil {
		return err
	}
	if err := copyFiles(fmt.Sprintf("%sservice_account.yaml", deployDir),
		deployDir, fmt.Sprintf("%s/rbac/", openshiftAbsPath), fs); err != nil {
		return err
	}

	// create service.yaml, service-monitor.yaml and operator.yaml
	s := &scaffold.Scaffold{}
	cfg := &input.Config{
		Repo:           repo,
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	if err := s.Execute(cfg,
		&scaffold.Service{},
		&scaffold.ServiceMonitor{},
		&scaffold.OpenshiftOperator{},
	); err != nil {
		return err
	}

	return nil
}

func makeDirs(dirs []string) error {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyDirs(fs afero.Fs, src string, dst string) error {
	return afero.Walk(fs, src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fullDst := strings.Replace(path, src, dst, 1)
			if info.IsDir() {
				if err = fs.MkdirAll(fullDst, info.Mode()); err != nil {
					return err
				}
			} else {
				if err = copyFiles(path, src, dst, fs); err != nil {
					return err
				}
			}
			return nil
		})
}

func copyFiles(path, src, dst string, fs afero.Fs) error {
	f, err := fs.Open(path)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	fullDst := strings.Replace(path, src, dst, 1)
	if err != nil {
		return err
	}
	if err = afero.WriteReader(fs, fullDst, f); err != nil {
		return err
	}
	if err = fs.Chmod(fullDst, info.Mode()); err != nil {
		return err
	}
	return nil
}
