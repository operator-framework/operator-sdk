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
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/flags/apiflags"
	"github.com/operator-framework/operator-sdk/internal/genutil"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
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
		return fmt.Errorf("the `add api` command is not supported for Go operators")
	case projutil.OperatorTypeHelm:
		return fmt.Errorf("the `add api` command is not supported for Helm operators")
	case projutil.OperatorTypeAnsible:
		if err := doAnsibleAPIScaffold(); err != nil {
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
