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

package add

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/internal/genutil"
	apiflags "github.com/operator-framework/operator-sdk/internal/flags/apiflags"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/helm/watches"
)

var apiFlags apiflags.APIFlags

func newAddAPICmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Adds a new api definition under pkg/apis",
		Long: `operator-sdk add api --kind=<kind> --api-version<group/version> 
creates an API definition for a new custom resource.
This command must be run from the project root directory.

For Go-based operators:

  - Creates the api definition for a new custom resource under pkg/apis.
  - By default, this command runs Kubernetes deepcopy and CRD generators on
  tagged types in all paths under pkg/apis. Go code is generated under
  pkg/apis/<group>/<version>/zz_generated.deepcopy.go. Generation can be disabled with the
  --skip-generation flag for Go-based operators.

For Ansible-based operators:

  - Creates resource folder under /roles.
  - watches.yaml is updated with new resource.
  - deploy/role.yaml will be updated with apiGroup for new API.

For Helm-based operators:
  - Creates resource folder under /helm-charts.
  - watches.yaml is updated with new resource.
  - deploy/role.yaml will be updated to reflact new rules for the incoming API.

CRD's are generated, or updated if they exist for a particular group + version + kind, under
deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI V3 validation YAML
is generated as a 'validation' object.`,
		Example: `  # Create a new API, under an existing project. This command must be run from the project root directory.
# Go Example:
  $ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService

# Ansible Example
  $ operator-sdk add api  \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService

# Helm Example:
  $ operator-sdk add api \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService

  $ operator-sdk add api \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService
  --helm-chart=myrepo/app

  $ operator-sdk add api \
  --helm-chart=myrepo/app

  $ operator-sdk add api \
  --helm-chart=myrepo/app \
  --helm-chart-version=1.2.3

  $ operator-sdk add api \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/

  $ operator-sdk add api \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/ \
  --helm-chart-version=1.2.3

  $ operator-sdk add api \
  --helm-chart=/path/to/local/chart-directories/app/

  $ operator-sdk add api \
  --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`,
		RunE: apiRun,
	}

	// Initialize flagSet struct with command flags
	apiFlags.AddTo(apiCmd.Flags())

	return apiCmd
}

func apiRun(cmd *cobra.Command, args []string) error {

	projutil.MustInProjectRoot()

	operatorType := projutil.GetOperatorType()
	if operatorType == projutil.OperatorTypeUnknown {
		return projutil.ErrUnknownOperatorType{}
	}
	// Verify the incoming flags.
	if err := apiFlags.VerifyCommonFlags(operatorType); err != nil {
		return err
	}

	log.Infof("Generating api version %s for kind %s.", apiFlags.APIVersion, apiFlags.Kind)

	switch operatorType {
	case projutil.OperatorTypeGo:
		if err := doGoAPIScaffold(); err != nil {
			return err
		}
	case projutil.OperatorTypeAnsible:
		if err := doAnsibleAPIScaffold(); err != nil {
			return err
		}
	case projutil.OperatorTypeHelm:
		if err := doHelmAPIScaffold(); err != nil {
			return err
		}
	}
	log.Info("API generation complete.")
	return nil
}

// TODO
// Consolidate scaffold func to be used by both "new" and "add api" commands.
func doAnsibleAPIScaffold() error {
	// Create and validate new resource.
	r, err := scaffold.NewResource(apiFlags.APIVersion, apiFlags.Kind)
	if err != nil {
		return fmt.Errorf("invalid apiVersion and kind: %v", err)
	}
	absProjectPath := projutil.MustGetwd()
	cfg := &input.Config{
		AbsProjectPath: absProjectPath,
	}
	roleFiles := ansible.RolesFiles{Resource: *r}
	roleTemplates := ansible.RolesTemplates{Resource: *r}

	// update watch.yaml for the given resource r.
	if err := ansible.UpdateAnsibleWatchForResource(r, absProjectPath); err != nil {
		return fmt.Errorf("failed to update the Watch manifest for the resource (%v, %v): (%v)",
			r.APIVersion, r.Kind, err)
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&scaffold.CR{Resource: r},
		&ansible.RolesReadme{Resource: *r},
		&ansible.RolesMetaMain{Resource: *r},
		&roleFiles,
		&roleTemplates,
		&ansible.RolesVarsMain{Resource: *r},
		&ansible.RolesDefaultsMain{Resource: *r},
		&ansible.RolesTasksMain{Resource: *r},
		&ansible.RolesHandlersMain{Resource: *r},
	)
	if err != nil {
		return fmt.Errorf("new ansible api scaffold failed: %v", err)
	}
	if err = genutil.GenerateCRDNonGo("", *r, apiFlags.CrdVersion); err != nil {
		return err
	}

	// Remove placeholders from empty directories
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleFiles.Path))
	if err != nil {
		return fmt.Errorf("new ansible api scaffold failed: %v", err)
	}
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleTemplates.Path))
	if err != nil {
		return fmt.Errorf("new ansible api scaffold failed: %v", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(r, absProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): (%v)",
			r.APIVersion, r.Kind, err)
	}
	return nil
}

// TODO
// Consolidate scaffold func to be used by both "new" and "add api" commands.
func doHelmAPIScaffold() error {

	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)
	cfg := &input.Config{
		AbsProjectPath: absProjectPath,
		ProjectName:    projectName,
	}

	createOpts := helm.CreateChartOptions{
		ResourceAPIVersion: apiFlags.APIVersion,
		ResourceKind:       apiFlags.Kind,
		Chart:              apiFlags.HelmChartRef,
		Version:            apiFlags.HelmChartVersion,
		Repo:               apiFlags.HelmChartRepo,
	}

	r, chart, err := helm.CreateChart(cfg.AbsProjectPath, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create helm chart: %v", err)
	}

	valuesPath := filepath.Join("<project_dir>", helm.HelmChartsDir, chart.Name(), "values.yaml")

	rawValues, err := yaml.Marshal(chart.Values)
	if err != nil {
		return fmt.Errorf("failed to get raw chart values: %v", err)
	}
	crSpec := fmt.Sprintf("# Default values copied from %s\n\n%s", valuesPath, rawValues)

	// update watch.yaml for the given resource.
	watchesFile := filepath.Join(cfg.AbsProjectPath, watches.WatchesFile)
	if err := watches.UpdateForResource(watchesFile, r, chart.Name()); err != nil {
		return fmt.Errorf("failed to update watches.yaml: %w", err)
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&scaffold.CR{
			Resource: r,
			Spec:     crSpec,
		},
	)
	if err != nil {
		log.Fatalf("API scaffold failed: %v", err)
	}
	if err = genutil.GenerateCRDNonGo("", *r, apiFlags.CrdVersion); err != nil {
		return err
	}

	roleScaffold := helm.DefaultRoleScaffold

	if k8sCfg, err := config.GetConfig(); err != nil {
		log.Warnf("Using default RBAC rules: failed to get Kubernetes config: %s", err)
	} else if dc, err := discovery.NewDiscoveryClientForConfig(k8sCfg); err != nil {
		log.Warnf("Using default RBAC rules: failed to create Kubernetes discovery client: %s", err)
	} else {
		roleScaffold = helm.GenerateRoleScaffold(dc, chart)
	}

	if err = scaffold.MergeRoleForResource(r, absProjectPath, roleScaffold); err != nil {
		return fmt.Errorf("failed to merge rules in the RBAC manifest for resource (%v, %v): %v",
			r.APIVersion, r.Kind, err)
	}

	return nil
}

// TODO
// Consolidate scaffold func to be used by both "new" and "add api" commands.
func doGoAPIScaffold() error {

	// Create and validate new resource.
	r, err := scaffold.NewResource(apiFlags.APIVersion, apiFlags.Kind)
	if err != nil {
		return err
	}

	absProjectPath := projutil.MustGetwd()

	cfg := &input.Config{
		Repo:           projutil.GetGoPkg(),
		AbsProjectPath: absProjectPath,
	}
	s := &scaffold.Scaffold{}

	// Check if any package files for this API group dir exist, and if not
	// scaffold a group.go to prevent erroneous gengo parse errors.
	group := &scaffold.Group{Resource: r}
	if err := scaffoldIfNoPkgFileExists(s, cfg, group); err != nil {
		log.Fatalf("Failed to scaffold group file: %v", err)
	}

	err = s.Execute(cfg,
		&scaffold.Types{Resource: r},
		&scaffold.AddToScheme{Resource: r},
		&scaffold.Register{Resource: r},
		&scaffold.Doc{Resource: r},
		&scaffold.CR{Resource: r},
	)
	if err != nil {
		log.Fatalf("API scaffold failed: %v", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(r, absProjectPath); err != nil {
		log.Fatalf("Failed to update the RBAC manifest for the resource (%v, %v): (%v)",
			r.APIVersion, r.Kind, err)
	}

	if !apiFlags.SkipGeneration {
		// Run k8s codegen for deepcopy
		if err := genutil.K8sCodegen(); err != nil {
			log.Fatal(err)
		}

		// Generate a validation spec for the new CRD.
		if err := genutil.CRDGen(apiFlags.CrdVersion); err != nil {
			log.Fatal(err)
		}
	}

	log.Info("API generation complete.")
	return nil
}

// scaffoldIfNoPkgFileExists executes f using s and cfg if no go files
// in f's directory exist.
func scaffoldIfNoPkgFileExists(s *scaffold.Scaffold, cfg *input.Config, f input.File) error {
	i, err := f.GetInput()
	if err != nil {
		return fmt.Errorf("error getting file %s input: %v", i.Path, err)
	}
	groupDir := filepath.Dir(i.Path)
	gdInfos, err := ioutil.ReadDir(groupDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error reading dir %s: %v", groupDir, err)
	}
	if err == nil {
		for _, info := range gdInfos {
			if !info.IsDir() && filepath.Ext(info.Name()) == ".go" {
				return nil
			}
		}
	}
	// err must be a non-existence error or no go files exist, so execute f.
	return s.Execute(cfg, f)
}
