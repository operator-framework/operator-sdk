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
	"path"
	"path/filepath"
	"strings"

	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"
	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

For example:

	$ mkdir $HOME/projects/example.com/
	$ cd $HOME/projects/example.com/
	$ operator-sdk new app-operator
generates a skeletal app-operator application in $HOME/projects/example.com/app-operator.
`,
		RunE: newFunc,
	}

	newCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1) - used with \"ansible\" or \"helm\" types")
	newCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService) - used with \"ansible\" or \"helm\" types")
	newCmd.Flags().StringVar(&operatorType, "type", "go", "Type of operator to initialize (choices: \"go\", \"ansible\" or \"helm\")")
	newCmd.Flags().StringVar(&repo, "repo", "", "Project repository path for Go operators. Used as the project's Go import path. This must be set if outside of $GOPATH/src (e.g. github.com/example-inc/my-operator)")
	newCmd.Flags().BoolVar(&gitInit, "git-init", false, "Initialize the project directory as a git repository (default false)")
	newCmd.Flags().StringVar(&headerFile, "header-file", "", "Path to file containing headers for generated Go files. Copied to hack/boilerplate.go.txt")
	newCmd.Flags().BoolVar(&makeVendor, "vendor", false, "Use a vendor directory for dependencies")
	newCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Do not validate the resulting project's structure and dependencies. (Only used for --type go)")
	newCmd.Flags().BoolVar(&generatePlaybook, "generate-playbook", false, "Generate a playbook skeleton. (Only used for --type ansible)")

	newCmd.Flags().StringVar(&helmChartRef, "helm-chart", "", "Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path)")
	newCmd.Flags().StringVar(&helmChartVersion, "helm-chart-version", "", "Specific version of the helm chart (default is latest version)")
	newCmd.Flags().StringVar(&helmChartRepo, "helm-chart-repo", "", "Chart repository URL for the requested helm chart")

	return newCmd
}

var (
	apiVersion       string
	kind             string
	operatorType     string
	projectName      string
	headerFile       string
	repo             string
	gitInit          bool
	makeVendor       bool
	skipValidation   bool
	generatePlaybook bool

	helmChartRef     string
	helmChartVersion string
	helmChartRepo    string
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
	case projutil.OperatorTypeGo:
		if repo == "" {
			repo = path.Join(projutil.GetGoPkg(), projectName)
		}
		if err := doGoScaffold(); err != nil {
			return err
		}
		if err := getDeps(); err != nil {
			return err
		}
		if !skipValidation {
			if err := validateProject(); err != nil {
				return err
			}
		}

	case projutil.OperatorTypeAnsible:
		if err := doAnsibleScaffold(); err != nil {
			return err
		}
	case projutil.OperatorTypeHelm:
		if err := doHelmScaffold(); err != nil {
			return err
		}
	}

	if gitInit {
		if err := initGit(); err != nil {
			return err
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
		log.Fatalf("Project (%v) in (%v) path already exists. Please use a different project name or delete the existing one", projectName, fp)
	}
}

func doGoScaffold() error {
	cfg := &input.Config{
		Repo:           repo,
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), projectName),
		ProjectName:    projectName,
	}
	s := &scaffold.Scaffold{}

	if headerFile != "" {
		err := s.Execute(cfg, &scaffold.Boilerplate{BoilerplateSrcPath: headerFile})
		if err != nil {
			return fmt.Errorf("boilerplate scaffold failed: (%v)", err)
		}
		s.BoilerplatePath = headerFile
	}

	if err := projutil.CheckGoModules(); err != nil {
		return err
	}

	err := s.Execute(cfg,
		&scaffold.GoMod{},
		&scaffold.Tools{},
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

func doAnsibleScaffold() error {
	cfg := &input.Config{
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), projectName),
		ProjectName:    projectName,
	}

	resource, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		return fmt.Errorf("invalid apiVersion and kind: (%v)", err)
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
		&ansible.MoleculeTestLocalPlaybook{Resource: *resource},
		&ansible.RolesDefaultsMain{Resource: *resource},
		&ansible.RolesTasksMain{Resource: *resource},
		&ansible.MoleculeDefaultMolecule{},
		&ansible.BuildTestFrameworkDockerfile{},
		&ansible.MoleculeTestClusterMolecule{},
		&ansible.MoleculeDefaultPrepare{},
		&ansible.MoleculeDefaultPlaybook{
			GeneratePlaybook: generatePlaybook,
			Resource:         *resource,
		},
		&ansible.BuildTestFrameworkAnsibleTestScript{},
		&ansible.MoleculeDefaultAsserts{},
		&ansible.MoleculeTestClusterPlaybook{Resource: *resource},
		&ansible.RolesHandlersMain{Resource: *resource},
		&ansible.Watches{
			GeneratePlaybook: generatePlaybook,
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

	if err = generateCRDNonGo(projectName, *resource); err != nil {
		return err
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
	if generatePlaybook {
		log.Infof("Generating %s playbook.", strings.Title(operatorType))

		err := s.Execute(cfg,
			&ansible.Playbook{Resource: *resource},
		)
		if err != nil {
			return fmt.Errorf("new ansible playbook scaffold failed: (%v)", err)
		}
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, cfg.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): (%v)", resource.APIVersion, resource.Kind, err)
	}
	return nil
}

func doHelmScaffold() error {
	cfg := &input.Config{
		AbsProjectPath: filepath.Join(projutil.MustGetwd(), projectName),
		ProjectName:    projectName,
	}

	createOpts := helm.CreateChartOptions{
		ResourceAPIVersion: apiVersion,
		ResourceKind:       kind,
		Chart:              helmChartRef,
		Version:            helmChartVersion,
		Repo:               helmChartRepo,
	}

	resource, chart, err := helm.CreateChart(cfg.AbsProjectPath, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create helm chart: %w", err)
	}

	valuesPath := filepath.Join("<project_dir>", helm.HelmChartsDir, chart.Name(), "values.yaml")

	rawValues, err := yaml.Marshal(chart.Values)
	if err != nil {
		return fmt.Errorf("failed to get raw chart values: %w", err)
	}
	crSpec := fmt.Sprintf("# Default values copied from %s\n\n%s", valuesPath, rawValues)

	roleScaffold := helm.DefaultRoleScaffold
	if k8sCfg, err := config.GetConfig(); err != nil {
		log.Warnf("Using default RBAC rules: failed to get Kubernetes config: %s", err)
	} else if dc, err := discovery.NewDiscoveryClientForConfig(k8sCfg); err != nil {
		log.Warnf("Using default RBAC rules: failed to create Kubernetes discovery client: %s", err)
	} else {
		roleScaffold = helm.GenerateRoleScaffold(dc, chart)
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&helm.Dockerfile{},
		&helm.WatchesYAML{
			Resource:  resource,
			ChartName: chart.Name(),
		},
		&scaffold.ServiceAccount{},
		&roleScaffold,
		&scaffold.RoleBinding{IsClusterScoped: roleScaffold.IsClusterScoped},
		&helm.Operator{},
		&scaffold.CR{
			Resource: resource,
			Spec:     crSpec,
		},
	)
	if err != nil {
		return fmt.Errorf("new helm scaffold failed: %w", err)
	}

	if err = generateCRDNonGo(projectName, *resource); err != nil {
		return err
	}

	if err := scaffold.UpdateRoleForResource(resource, cfg.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for resource (%v, %v): %w", resource.APIVersion, resource.Kind, err)
	}
	return nil
}

func generateCRDNonGo(projectName string, resource scaffold.Resource) error {
	crdsDir := filepath.Join(projectName, scaffold.CRDsDir)
	gcfg := gen.Config{
		Inputs:    map[string]string{gencrd.CRDsDirKey: crdsDir},
		OutputDir: crdsDir,
	}
	crd := gencrd.NewCRDNonGo(gcfg, resource)
	if err := crd.Generate(); err != nil {
		return fmt.Errorf("error generating CRD for %s: %w", resource, err)
	}
	log.Info("Generated CustomResourceDefinition manifests.")
	return nil
}

func verifyFlags() error {
	if operatorType != projutil.OperatorTypeGo && operatorType != projutil.OperatorTypeAnsible && operatorType != projutil.OperatorTypeHelm {
		return errors.Wrap(projutil.ErrUnknownOperatorType{Type: operatorType}, "value of --type can only be `go`, `ansible`, or `helm`")
	}
	if operatorType != projutil.OperatorTypeAnsible && generatePlaybook {
		return fmt.Errorf("value of --generate-playbook can only be used with --type `ansible`")
	}

	if len(helmChartRef) != 0 {
		if operatorType != projutil.OperatorTypeHelm {
			return fmt.Errorf("value of --helm-chart can only be used with --type=helm")
		}
	} else if len(helmChartRepo) != 0 {
		return fmt.Errorf("value of --helm-chart-repo can only be used with --type=helm and --helm-chart")
	} else if len(helmChartVersion) != 0 {
		return fmt.Errorf("value of --helm-chart-version can only be used with --type=helm and --helm-chart")
	}

	if operatorType == projutil.OperatorTypeGo {
		if len(apiVersion) != 0 || len(kind) != 0 {
			return fmt.Errorf("operators of type Go do not use --api-version or --kind")
		}

		if err := projutil.CheckRepo(repo); err != nil {
			return err
		}
	}

	// --api-version and --kind are required with --type=ansible and --type=helm, with one exception.
	//
	// If --type=helm and --helm-chart is set, --api-version and --kind are optional. If left unset,
	// sane defaults are used when the specified helm chart is created.
	if operatorType == projutil.OperatorTypeAnsible || operatorType == projutil.OperatorTypeHelm && len(helmChartRef) == 0 {
		if len(apiVersion) == 0 {
			return fmt.Errorf("value of --api-version must not have empty value")
		}
		if len(kind) == 0 {
			return fmt.Errorf("value of --kind must not have empty value")
		}
		kindFirstLetter := string(kind[0])
		if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
			return fmt.Errorf("value of --kind must start with an uppercase letter")
		}
		if strings.Count(apiVersion, "/") != 1 {
			return fmt.Errorf("value of --api-version has wrong format (%v); format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", apiVersion)
		}
	}

	return nil
}

func execProjCmd(cmd string, args ...string) error {
	dc := exec.Command(cmd, args...)
	dc.Dir = filepath.Join(projutil.MustGetwd(), projectName)
	return projutil.ExecCmd(dc)
}

func getDeps() error {

	// Only when a user requests a vendor directory be created should
	// "go mod vendor" be run during project initialization.
	if !makeVendor {
		return nil
	}

	log.Info("Running go mod vendor")
	opts := projutil.GoCmdOptions{
		Args: []string{"-v"},
		Dir:  filepath.Join(projutil.MustGetwd(), projectName),
	}
	if err := projutil.GoCmd("mod vendor", opts); err != nil {
		return err
	}
	log.Info("Done getting dependencies")
	return nil
}

func initGit() error {
	log.Info("Running git init")
	if err := execProjCmd("git", "init"); err != nil {
		return errors.Wrapf(err, "failed to run git init")
	}
	log.Info("Run git init done")
	return nil
}

func validateProject() error {
	log.Info("Validating project")
	// Run "go build ./..." to make sure all packages can be built
	// correctly. From "go help build":
	//
	//	When compiling multiple packages or a single non-main package,
	//	build compiles the packages but discards the resulting object,
	//	serving only as a check that the packages can be built.
	opts := projutil.GoCmdOptions{
		PackagePath: "./...",
		Dir:         filepath.Join(projutil.MustGetwd(), projectName),
	}
	if err := projutil.GoBuild(opts); err != nil {
		return err
	}
	log.Info("Project validation successful.")
	return nil
}
