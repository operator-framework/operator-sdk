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

package cmd

import (
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/helm"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewMigrateCmd returns a command that will add source code to an existing non-go operator
func NewMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Adds source code to an operator",
		Long:  `operator-sdk migrate adds a main.go source file and any associated source files for an operator that is not of the "go" type.`,
		Run:   migrateRun,
	}
}

// migrateRun determines the current operator type and runs the corresponding
// migrate function.
func migrateRun(cmd *cobra.Command, args []string) {
	projutil.MustInProjectRoot()

	_ = projutil.CheckAndGetProjectGoPkg()

	opType := projutil.GetOperatorType()
	switch opType {
	case projutil.OperatorTypeAnsible:
		migrateAnsible()
	case projutil.OperatorTypeHelm:
		migrateHelm()
	default:
		log.Fatalf("Operator of type %s cannot be migrated.", opType)
	}
}

// migrateAnsible runs the migration process for an ansible-based operator
func migrateAnsible() {
	wd := projutil.MustGetwd()

	cfg := &input.Config{
		AbsProjectPath: wd,
		ProjectName:    filepath.Base(wd),
	}

	dockerfile := ansible.DockerfileHybrid{
		Watches: true,
		Roles:   true,
	}
	_, err := os.Stat("playbook.yaml")
	switch {
	case err == nil:
		dockerfile.Playbook = true
	case os.IsNotExist(err):
		log.Info("No playbook was found, so not including it in the new Dockerfile")
	default:
		log.Fatalf("Error trying to stat playbook.yaml: (%v)", err)
	}

	renameDockerfile()

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&ansible.Main{},
		&ansible.GopkgToml{},
		&dockerfile,
		&ansible.Entrypoint{},
		&ansible.UserSetup{},
	)
	if err != nil {
		log.Fatalf("Migrate scaffold failed: (%v)", err)
	}
}

// migrateHelm runs the migration process for a helm-based operator
func migrateHelm() {
	wd := projutil.MustGetwd()

	cfg := &input.Config{
		AbsProjectPath: wd,
		ProjectName:    filepath.Base(wd),
	}

	renameDockerfile()

	s := &scaffold.Scaffold{}
	err := s.Execute(cfg,
		&helm.Main{},
		&helm.GopkgToml{},
		&helm.DockerfileHybrid{
			Watches:    true,
			HelmCharts: true,
		},
		&helm.Entrypoint{},
		&helm.UserSetup{},
	)
	if err != nil {
		log.Fatalf("Migrate scaffold failed: (%v)", err)
	}
}

func renameDockerfile() {
	dockerfilePath := filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile)
	newDockerfilePath := dockerfilePath + ".sdkold"
	err := os.Rename(dockerfilePath, newDockerfilePath)
	if err != nil {
		log.Fatalf("Failed to rename Dockerfile: (%v)", err)
	}
	log.Infof("Renamed Dockerfile to %s and replaced with newer version. Compare the new Dockerfile to your old one and manually migrate any customizations", newDockerfilePath)
}
