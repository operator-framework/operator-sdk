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

package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MigrateCmd struct {
	DepManager string
	HeaderFile string
}

// MigrateRun determines the current operator type and runs the corresponding
// migrate function.
func (c *MigrateCmd) Run() error {
	opType := projutil.GetOperatorType()
	switch opType {
	case projutil.OperatorTypeAnsible:
		return c.migrateAnsible()
	case projutil.OperatorTypeHelm:
		return c.migrateHelm()
	}
	return fmt.Errorf("operator of type %s cannot be migrated", opType)
}

// migrateAnsible runs the migration process for an ansible-based operator
func (c *MigrateCmd) migrateAnsible() error {
	wd := projutil.MustGetwd()

	dockerfile := ansible.DockerfileHybrid{
		Watches: true,
		Roles:   true,
	}
	_, err := os.Stat(ansible.PlaybookYamlFile)
	switch {
	case err == nil:
		dockerfile.Playbook = true
	case os.IsNotExist(err):
		log.Info("No playbook was found, so not including it in the new Dockerfile")
	default:
		return fmt.Errorf("error trying to stat %s: (%v)", ansible.PlaybookYamlFile, err)
	}
	if err := renameDockerfile(); err != nil {
		return err
	}

	s := &scaffold.Scaffold{
		Repo:           viper.GetString(config.RepoOpt),
		AbsProjectPath: wd,
		ProjectName:    filepath.Base(wd),
	}

	if c.HeaderFile != "" {
		err = s.Execute(&scaffold.Boilerplate{BoilerplateSrcPath: c.HeaderFile})
		if err != nil {
			return fmt.Errorf("boilerplate scaffold failed: (%v)", err)
		}
		s.BoilerplatePath = c.HeaderFile
	}

	if err := scaffoldAnsibleDepManager(c.DepManager, s); err != nil {
		return errors.Wrap(err, "migrate Ansible dependency manager file scaffold failed")
	}

	err = s.Execute(
		&ansible.Main{},
		&dockerfile,
		&ansible.Entrypoint{},
		&ansible.UserSetup{},
		&ansible.K8sStatus{},
		&ansible.AoLogs{},
	)
	if err != nil {
		return fmt.Errorf("migrate ansible scaffold failed: (%v)", err)
	}
	return nil
}

// migrateHelm runs the migration process for a helm-based operator
func (c *MigrateCmd) migrateHelm() error {
	wd := projutil.MustGetwd()

	if err := renameDockerfile(); err != nil {
		return err
	}

	s := &scaffold.Scaffold{
		Repo:           viper.GetString(config.RepoOpt),
		AbsProjectPath: wd,
		ProjectName:    filepath.Base(wd),
	}

	if c.HeaderFile != "" {
		err := s.Execute(&scaffold.Boilerplate{BoilerplateSrcPath: c.HeaderFile})
		if err != nil {
			return fmt.Errorf("boilerplate scaffold failed: (%v)", err)
		}
		s.BoilerplatePath = c.HeaderFile
	}

	if err := scaffoldHelmDepManager(c.DepManager, s); err != nil {
		return errors.Wrap(err, "migrate Helm dependency manager file scaffold failed")
	}

	err := s.Execute(
		&helm.Main{},
		&helm.DockerfileHybrid{
			Watches:    true,
			HelmCharts: true,
		},
		&helm.Entrypoint{},
		&helm.UserSetup{},
	)
	if err != nil {
		return fmt.Errorf("migrate helm scaffold failed: (%v)", err)
	}
	return nil
}

func renameDockerfile() error {
	dockerfilePath := filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile)
	newDockerfilePath := dockerfilePath + ".sdkold"
	err := os.Rename(dockerfilePath, newDockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to rename Dockerfile: (%v)", err)
	}
	log.Infof("Renamed Dockerfile to %s and replaced with newer version. Compare the new Dockerfile to your old one and manually migrate any customizations", newDockerfilePath)
	return nil
}

func scaffoldHelmDepManager(dm string, s *scaffold.Scaffold) error {
	var files []input.File
	switch m := projutil.DepManagerType(dm); m {
	case projutil.DepManagerDep:
		files = append(files, &helm.GopkgToml{})
	case projutil.DepManagerGoMod:
		if err := goModCheck(); err != nil {
			return err
		}
		files = append(files, &helm.GoMod{}, &scaffold.Tools{})
	default:
		return projutil.ErrInvalidDepManager(dm)
	}
	return s.Execute(files...)
}

func scaffoldAnsibleDepManager(dm string, s *scaffold.Scaffold) error {
	var files []input.File
	switch m := projutil.DepManagerType(dm); m {
	case projutil.DepManagerDep:
		files = append(files, &ansible.GopkgToml{})
	case projutil.DepManagerGoMod:
		if err := goModCheck(); err != nil {
			return err
		}
		files = append(files, &ansible.GoMod{}, &scaffold.Tools{})
	default:
		return projutil.ErrInvalidDepManager(dm)
	}
	return s.Execute(files...)
}

func goModCheck() error {
	goModOn, err := projutil.GoModOn()
	if err == nil && !goModOn {
		log.Fatal(`Dependency manager "modules" has been selected but go modules are not active. ` +
			`Activate modules then run "operator-sdk migrate".`)
	}
	return err
}
