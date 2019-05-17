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

package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type NewCmd struct {
	APIVersion   string
	Kind         string
	OperatorType string
	ProjectName  string
	DepManager   string
	HeaderFile   string
	SkipGit      bool

	Ansible NewAnsibleCmd
	Helm    NewHelmCmd

	// WriteConfig is true if no config file was used during project init.
	WriteConfig bool
}

type NewAnsibleCmd struct {
	GeneratePlaybook bool
}

type NewHelmCmd struct {
	ChartRef     string
	ChartVersion string
	ChartRepo    string
}

func (c *NewCmd) Run() error {

	if err := c.verifyFlags(); err != nil {
		return err
	}

	log.Infof("Creating new %s operator '%s'.", strings.Title(c.OperatorType), c.ProjectName)

	switch c.OperatorType {
	case projutil.OperatorTypeGo:
		if err := c.doGoScaffold(); err != nil {
			return err
		}
		if err := getDeps(c.DepManager, c.ProjectName); err != nil {
			return err
		}
	case projutil.OperatorTypeAnsible:
		if err := c.doAnsibleScaffold(); err != nil {
			return err
		}
	case projutil.OperatorTypeHelm:
		if err := c.doHelmScaffold(); err != nil {
			return err
		}
	}

	// Write config if no config was specified.
	if c.WriteConfig {
		path := filepath.Join(c.ProjectName, config.DefaultFileName)
		if err := config.WriteConfigAs(path); err != nil {
			return err
		}
	}

	if !c.SkipGit {
		if err := initGit(c.ProjectName); err != nil {
			return err
		}
	}

	log.Info("Project creation complete.")
	return nil
}

// mustBeNewProject checks if the given project exists under the current diretory.
// it exits with error when the project exists.
func mustBeNewProject(projectName string) {
	fp := filepath.Join(projutil.MustGetwd(), projectName)
	stat, err := os.Stat(fp)
	if err != nil && os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Fatalf("Failed to determine if project (%v) exists", projectName)
	}
	if stat.IsDir() {
		log.Fatalf("Project (%v) in (%v) path already exists. Please use a different project name or delete the existing one", projectName, fp)
	}
}

func (c *NewCmd) doGoScaffold() error {
	s := &scaffold.Scaffold{
		Repo:           viper.GetString(config.RepoOpt),
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), c.ProjectName),
		ProjectName:    c.ProjectName,
	}

	if c.HeaderFile != "" {
		err := s.Execute(&scaffold.Boilerplate{BoilerplateSrcPath: c.HeaderFile})
		if err != nil {
			return fmt.Errorf("boilerplate scaffold failed: (%v)", err)
		}
		s.BoilerplatePath = c.HeaderFile
	}

	var err error
	switch m := projutil.DepManagerType(c.DepManager); m {
	case projutil.DepManagerDep:
		err = s.Execute(&scaffold.GopkgToml{})
	case projutil.DepManagerGoMod:
		if goModOn, merr := projutil.GoModOn(); merr != nil {
			return merr
		} else if !goModOn {
			log.Fatalf(`Dependency manager "%s" has been selected but go modules are not active. `+
				`Activate modules then run "operator-sdk new %s".`, m, c.ProjectName)
		}
		err = s.Execute(&scaffold.GoMod{}, &scaffold.Tools{})
	default:
		err = projutil.ErrNoDepManager
	}
	if err != nil {
		return fmt.Errorf("dependency manager file scaffold failed: (%v)", err)
	}

	err = s.Execute(
		&scaffold.Cmd{},
		&scaffold.Dockerfile{},
		&scaffold.Entrypoint{},
		&scaffold.UserSetup{},
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.Operator{},
		&scaffold.Apis{},
		&scaffold.Controller{},
		&scaffold.Version{},
		&scaffold.Gitignore{},
	)
	if err != nil {
		return fmt.Errorf("new Go scaffold failed: (%v)", err)
	}
	return nil
}

func (c *NewCmd) doAnsibleScaffold() error {
	resource, err := scaffold.NewResource(c.APIVersion, c.Kind)
	if err != nil {
		return fmt.Errorf("invalid apiVersion and kind: (%v)", err)
	}

	roleFiles := ansible.RolesFiles{Resource: *resource}
	roleTemplates := ansible.RolesTemplates{Resource: *resource}

	s := &scaffold.Scaffold{
		Repo:           viper.GetString(config.RepoOpt),
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), c.ProjectName),
		ProjectName:    c.ProjectName,
	}
	err = s.Execute(
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.CRD{Resource: resource},
		&scaffold.CR{Resource: resource},
		&ansible.BuildDockerfile{GeneratePlaybook: c.Ansible.GeneratePlaybook},
		&ansible.RolesReadme{Resource: *resource},
		&ansible.RolesMetaMain{Resource: *resource},
		&roleFiles,
		&roleTemplates,
		&ansible.RolesVarsMain{Resource: *resource},
		&ansible.MoleculeTestLocalPlaybook{Resource: *resource},
		&ansible.RolesDefaultsMain{Resource: *resource},
		&ansible.RolesTasksMain{Resource: *resource},
		&ansible.MoleculeDefaultMolecule{},
		&ansible.BuildTestFrameworkDockerfile{},
		&ansible.MoleculeTestClusterMolecule{},
		&ansible.MoleculeDefaultPrepare{},
		&ansible.MoleculeDefaultPlaybook{
			GeneratePlaybook: c.Ansible.GeneratePlaybook,
			Resource:         *resource,
		},
		&ansible.BuildTestFrameworkAnsibleTestScript{},
		&ansible.MoleculeDefaultAsserts{},
		&ansible.MoleculeTestClusterPlaybook{Resource: *resource},
		&ansible.RolesHandlersMain{Resource: *resource},
		&ansible.Watches{
			GeneratePlaybook: c.Ansible.GeneratePlaybook,
			Resource:         *resource,
		},
		&ansible.DeployOperator{},
		&ansible.Travis{},
		&ansible.MoleculeTestLocalMolecule{},
		&ansible.MoleculeTestLocalPrepare{Resource: *resource},
	)
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: (%v)", err)
	}

	// Remove placeholders from empty directories
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleFiles.Path))
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: (%v)", err)
	}
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleTemplates.Path))
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: (%v)", err)
	}

	// Decide on playbook.
	if c.Ansible.GeneratePlaybook {
		log.Infof("Generating %s playbook.", strings.Title(c.OperatorType))

		err := s.Execute(
			&ansible.Playbook{Resource: *resource},
		)
		if err != nil {
			return fmt.Errorf("new ansible playbook scaffold failed: (%v)", err)
		}
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, s.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): (%v)", resource.APIVersion, resource.Kind, err)
	}
	return nil
}

func (c *NewCmd) doHelmScaffold() error {
	createOpts := helm.CreateChartOptions{
		ResourceAPIVersion: c.APIVersion,
		ResourceKind:       c.Kind,
		Chart:              c.Helm.ChartRef,
		Version:            c.Helm.ChartVersion,
		Repo:               c.Helm.ChartRepo,
	}

	s := &scaffold.Scaffold{
		Repo:           viper.GetString(config.RepoOpt),
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), c.ProjectName),
		ProjectName:    c.ProjectName,
	}

	resource, chart, err := helm.CreateChart(s.AbsProjectPath, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create helm chart: %s", err)
	}

	valuesPath := filepath.Join("<project_dir>", helm.HelmChartsDir, chart.GetMetadata().GetName(), "values.yaml")
	crSpec := fmt.Sprintf("# Default values copied from %s\n\n%s", valuesPath, chart.GetValues().GetRaw())

	k8sCfg, err := k8sconfig.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %s", err)
	}
	roleScaffold, err := helm.CreateRoleScaffold(k8sCfg, chart)
	if err != nil {
		return fmt.Errorf("failed to generate role scaffold: %s", err)
	}

	err = s.Execute(
		&helm.Dockerfile{},
		&helm.WatchesYAML{
			Resource:  resource,
			ChartName: chart.GetMetadata().GetName(),
		},
		&scaffold.ServiceAccount{},
		roleScaffold,
		&scaffold.RoleBinding{IsClusterScoped: roleScaffold.IsClusterScoped},
		&helm.Operator{},
		&scaffold.CRD{Resource: resource},
		&scaffold.CR{
			Resource: resource,
			Spec:     crSpec,
		},
	)
	if err != nil {
		return fmt.Errorf("new helm scaffold failed: (%v)", err)
	}

	if err := scaffold.UpdateRoleForResource(resource, s.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for resource (%v, %v): (%v)", resource.APIVersion, resource.Kind, err)
	}
	return nil
}

func (c *NewCmd) verifyFlags() error {
	if c.OperatorType != projutil.OperatorTypeGo && c.OperatorType != projutil.OperatorTypeAnsible && c.OperatorType != projutil.OperatorTypeHelm {
		return errors.Wrap(projutil.ErrUnknownOperatorType{Type: c.OperatorType}, "value of --type can only be `go`, `ansible`, or `helm`")
	}
	if c.OperatorType != projutil.OperatorTypeAnsible && c.Ansible.GeneratePlaybook {
		return fmt.Errorf("value of --generate-playbook can only be used with --type `ansible`")
	}

	if len(c.Helm.ChartRef) != 0 {
		if c.OperatorType != projutil.OperatorTypeHelm {
			return fmt.Errorf("value of --helm-chart can only be used with --type=helm")
		}
	} else if len(c.Helm.ChartRepo) != 0 {
		return fmt.Errorf("value of --helm-chart-repo can only be used with --type=helm and --helm-chart")
	} else if len(c.Helm.ChartVersion) != 0 {
		return fmt.Errorf("value of --helm-chart-version can only be used with --type=helm and --helm-chart")
	}

	if c.OperatorType == projutil.OperatorTypeGo && (len(c.APIVersion) != 0 || len(c.Kind) != 0) {
		return fmt.Errorf("operators of type Go do not use --api-version or --kind")
	}

	// --api-version and --kind are required with --type=ansible and --type=helm, with one exception.
	//
	// If --type=helm and --helm-chart is set, --api-version and --kind are optional. If left unset,
	// sane defaults are used when the specified helm chart is created.
	if c.OperatorType == projutil.OperatorTypeAnsible || c.OperatorType == projutil.OperatorTypeHelm && len(c.Helm.ChartRef) == 0 {
		if len(c.APIVersion) == 0 {
			return fmt.Errorf("value of --api-version must not have empty value")
		}
		if len(c.Kind) == 0 {
			return fmt.Errorf("value of --kind must not have empty value")
		}
		kindFirstLetter := string(c.Kind[0])
		if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
			return fmt.Errorf("value of --kind must start with an uppercase letter")
		}
		if strings.Count(c.APIVersion, "/") != 1 {
			return fmt.Errorf("value of --api-version has wrong format (%v); format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", c.APIVersion)
		}
	}

	return nil
}

func execCmdInDir(dir, cmd string, args ...string) error {
	dc := exec.Command(cmd, args...)
	dc.Dir = filepath.Join(projutil.MustGetwd(), dir)
	return projutil.ExecCmd(dc)
}

func getDeps(dm, projectName string) error {
	switch m := projutil.DepManagerType(dm); m {
	case projutil.DepManagerDep:
		log.Info("Running dep ensure ...")
		if err := execCmdInDir(projectName, "dep", "ensure", "-v"); err != nil {
			return err
		}
	case projutil.DepManagerGoMod:
		log.Info("Running go mod ...")
		if err := execCmdInDir(projectName, "go", "mod", "vendor", "-v"); err != nil {
			return err
		}
	default:
		return projutil.ErrInvalidDepManager(dm)
	}
	log.Info("Done getting dependencies")
	return nil
}

func initGit(projectName string) error {
	log.Info("Run git init ...")
	if err := execCmdInDir(projectName, "git", "init"); err != nil {
		return err
	}
	if err := execCmdInDir(projectName, "git", "add", "--all"); err != nil {
		return err
	}
	if err := execCmdInDir(projectName, "git", "commit", "-q", "-m", "INITIAL COMMIT"); err != nil {
		return err
	}
	log.Info("Run git init done")
	return nil
}
