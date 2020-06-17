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

package new

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/flags/apiflags"
	"github.com/operator-framework/operator-sdk/internal/genutil"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

func NewCmd() *cobra.Command { //nolint:golint
	/*
		The nolint here is used to hide the warning
		"func name will be used as new.NewCmd by other packages,
		and that stutters; consider calling this Cmd"
		which is a false positive.
	*/
	newCmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Creates a new operator application",
		Long: `The operator-sdk new command creates a new operator application and
generates a default directory layout based on the input <project-name>.

<project-name> is the project name of the new operator. (e.g app-operator)
`,
		Example: `  # Create a new project directory
  $ mkdir $HOME/projects/example.com/
  $ cd $HOME/projects/example.com/

  # Ansible project
  $ operator-sdk new app-operator --type=ansible \
    --api-version=app.example.com/v1alpha1 \
    --kind=AppService

  # Helm project
  $ operator-sdk new app-operator --type=helm \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService

  $ operator-sdk new app-operator --type=helm \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService \
  --helm-chart=myrepo/app

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=myrepo/app

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=myrepo/app \
  --helm-chart-version=1.2.3

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/ \
  --helm-chart-version=1.2.3

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=/path/to/local/chart-directories/app/

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`,
		RunE: newFunc,
	}

	newCmd.Flags().StringVar(&operatorType, "type", "",
		"Type of operator to initialize (choices: \"ansible\" or \"helm\")")
	if err := newCmd.MarkFlagRequired("type"); err != nil {
		log.Fatalf("Failed to mark `type` flag for `new` subcommand as required")
	}

	// todo(camilamacedo86): remove before 1.0.0
	newCmd.Flags().BoolVar(&gitInit, "git-init", false,
		"Initialize the project directory as a git repository (default false)")
	err := newCmd.Flags().MarkDeprecated("git-init",
		"instead run `git init` once your project is created to use git")
	if err != nil {
		log.Fatal(err)
	}

	newCmd.Flags().BoolVar(&generatePlaybook, "generate-playbook", false,
		"Generate a playbook skeleton. (Only used for --type ansible)")

	// Initialize flagSet struct with common flags
	apiFlags.AddTo(newCmd.Flags())

	return newCmd
}

var (
	apiFlags         apiflags.APIFlags
	operatorType     string
	projectName      string
	gitInit          bool
	generatePlaybook bool
)

func newFunc(cmd *cobra.Command, args []string) error {
	if err := parse(cmd, args); err != nil {
		return err
	}
	mustBeNewProject()
	if err := verifyFlags(); err != nil {
		return err
	}

	log.Infof("Creating new %s operator '%s'.", strings.Title(operatorType), projectName)

	switch operatorType {
	case projutil.OperatorTypeAnsible:
		if err := doAnsibleScaffold(); err != nil {
			log.Fatal(err)
		}
	case projutil.OperatorTypeHelm:
		// create the project dir
		err := os.MkdirAll(projectName, 0755)
		if err != nil {
			log.Fatal(err)
		}
		// go inside of the project dir
		err = os.Chdir(filepath.Join(projutil.MustGetwd(), projectName))
		if err != nil {
			log.Fatal(err)
		}

		cfg := input.Config{
			AbsProjectPath: filepath.Join(projutil.MustGetwd()),
			ProjectName:    projectName,
		}

		createOpts := helm.CreateChartOptions{
			ResourceAPIVersion: apiFlags.APIVersion,
			ResourceKind:       apiFlags.Kind,
			Chart:              apiFlags.HelmChartRef,
			Version:            apiFlags.HelmChartVersion,
			Repo:               apiFlags.HelmChartRepo,
			CRDVersion:         apiFlags.CrdVersion,
		}

		if err := helm.Init(cfg, createOpts); err != nil {
			log.Fatal(err)
		}
	}
	//todo: remove before 1.0.0
	if gitInit {
		if err := initGit(); err != nil {
			log.Fatal(err)
		}
	}

	log.Info("Project creation complete.")
	return nil
}

func parse(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("command %s requires exactly one argument", cmd.CommandPath())
	}
	projectName = args[0]
	if len(projectName) == 0 {
		return fmt.Errorf("project name must not be empty")
	}
	return nil
}

// mustBeNewProject checks if the given project exists under the current diretory.
// it exits with error when the project exists.
func mustBeNewProject() {
	fp := filepath.Join(projutil.MustGetwd(), projectName)
	stat, err := os.Stat(fp)
	if err != nil && os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Fatalf("Failed to determine if project (%v) exists", projectName)
	}
	if stat.IsDir() {
		log.Fatalf("Project (%v) in (%v) path already exists. Please use a different project name or delete "+
			"the existing one", projectName, fp)
	}
}

func doAnsibleScaffold() error {
	cfg := &input.Config{
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), projectName),
		ProjectName:    projectName,
	}

	resource, err := scaffold.NewResource(apiFlags.APIVersion, apiFlags.Kind)
	if err != nil {
		return fmt.Errorf("invalid apiVersion and kind: %v", err)
	}

	roleFiles := ansible.RolesFiles{Resource: *resource}
	roleTemplates := ansible.RolesTemplates{Resource: *resource}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.CR{Resource: resource},
		&ansible.BuildDockerfile{GeneratePlaybook: generatePlaybook},
		&ansible.RolesReadme{Resource: *resource},
		&ansible.RolesMetaMain{Resource: *resource},
		&roleFiles,
		&roleTemplates,
		&ansible.RolesVarsMain{Resource: *resource},
		&ansible.MoleculeTestLocalConverge{Resource: *resource},
		&ansible.RolesDefaultsMain{Resource: *resource},
		&ansible.RolesTasksMain{Resource: *resource},
		&ansible.MoleculeDefaultMolecule{},
		&ansible.MoleculeDefaultPrepare{},
		&ansible.MoleculeDefaultConverge{
			GeneratePlaybook: generatePlaybook,
			Resource:         *resource,
		},
		&ansible.MoleculeDefaultVerify{},
		&ansible.RolesHandlersMain{Resource: *resource},
		&ansible.Watches{
			GeneratePlaybook: generatePlaybook,
			Resource:         *resource,
		},
		&ansible.DeployOperator{},
		&ansible.Travis{},
		&ansible.RequirementsYml{},
		&ansible.MoleculeTestLocalMolecule{},
		&ansible.MoleculeTestLocalPrepare{},
		&ansible.MoleculeTestLocalVerify{},
		&ansible.MoleculeClusterMolecule{Resource: *resource},
		&ansible.MoleculeClusterCreate{},
		&ansible.MoleculeClusterPrepare{Resource: *resource},
		&ansible.MoleculeClusterConverge{},
		&ansible.MoleculeClusterVerify{Resource: *resource},
		&ansible.MoleculeClusterDestroy{Resource: *resource},
		&ansible.MoleculeTemplatesOperator{},
	)
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: %v", err)
	}

	if err = genutil.GenerateCRDNonGo(projectName, *resource, apiFlags.CrdVersion); err != nil {
		return err
	}

	// Remove placeholders from empty directories
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleFiles.Path))
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: %v", err)
	}
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleTemplates.Path))
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: %v", err)
	}

	// Decide on playbook.
	if generatePlaybook {
		log.Infof("Generating %s playbook.", strings.Title(operatorType))

		err := s.Execute(cfg,
			&ansible.Playbook{Resource: *resource},
		)
		if err != nil {
			return fmt.Errorf("new ansible playbook scaffold failed: %v", err)
		}
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, cfg.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): %v",
			resource.APIVersion, resource.Kind, err)
	}
	return nil
}

func verifyFlags() error {
	if operatorType != projutil.OperatorTypeAnsible && operatorType != projutil.OperatorTypeHelm {
		return fmt.Errorf("value of --type can only be `ansible`, or `helm`: %v",
			projutil.ErrUnknownOperatorType{Type: operatorType})
	}
	if operatorType != projutil.OperatorTypeAnsible && generatePlaybook {
		return fmt.Errorf("value of --generate-playbook can only be used with --type `ansible`")
	}
	if err := apiFlags.VerifyCommonFlags(operatorType); err != nil {
		return err
	}

	return nil
}

// todo(camilamacedo86): remove before 1.0.0
// Deprecated: the git-init flag was deprecated since has no need to make the command run the git init.
// users are allowed to easily do that when they wish. This func is just used here to run the git-init
func execProjCmd(cmd string, args ...string) error {
	dc := exec.Command(cmd, args...)
	dc.Dir = filepath.Join(projutil.MustGetwd(), projectName)
	return projutil.ExecCmd(dc)
}

// todo(camilamacedo86): remove before 1.0.0
// Deprecated: the git-init flag was deprecated since has no need to make the command run the git init.
// users are allowed to easily do that when they wish.
func initGit() error {
	log.Info("Running git init")
	if err := execProjCmd("git", "init"); err != nil {
		return fmt.Errorf("failed to run git init: %v", err)
	}
	log.Info("Run git init done")
	return nil
}
