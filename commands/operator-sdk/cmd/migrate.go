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
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

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
	default:
		log.Fatalf("operator of type %s cannot be migrated.", opType)
	}
}

// migrateAnsible runs the migration process for an ansible-based operator
func migrateAnsible() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not identify current working directory: (%v)", err)
	}

	cfg := &input.Config{
		AbsProjectPath: projutil.MustGetwd(),
		ProjectName:    filepath.Base(wd),
	}

	dockerfile := ansible.DockerfileHybrid{
		Watches: true,
		Roles:   true,
	}
	_, err = os.Stat("playbook.yaml")
	switch {
	case err == nil:
		dockerfile.Playbook = true
	case !os.IsNotExist(err):
		log.Fatalf("error trying to stat playbook.yaml: (%v)", err)
	}

	dockerfilePath := filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile)
	err = os.Rename(dockerfilePath, dockerfilePath+".sdkold")
	if err != nil {
		log.Fatalf("failed to rename Dockerfile: (%v)", err)
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&ansible.Main{},
		&ansible.GopkgToml{},
		&dockerfile,
		&ansible.Entrypoint{},
		&ansible.UserSetup{},
	)
	if err != nil {
		log.Fatalf("add scaffold failed: (%v)", err)
	}
}
