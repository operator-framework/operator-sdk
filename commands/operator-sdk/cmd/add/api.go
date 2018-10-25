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
	"log"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/generate"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/spf13/cobra"
)

var (
	apiVersion string
	kind       string
)

func NewApiCmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Adds a new api definition under pkg/apis",
		Long: `operator-sdk add api --kind=<kind> --api-version=<group/version> creates the
api definition for a new custom resource under pkg/apis. This command must be run from the project root directory.
If the api already exists at pkg/apis/<group>/<version> then the command will not overwrite and return an error.

Example:
	$ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService
	$ tree pkg/apis
	pkg/apis/
	├── addtoscheme_app_appservice.go
	├── apis.go
	└── app
		└── v1alpha1
			├── doc.go
			├── register.go
			├── types.go

`,
		Run: apiRun,
	}

	apiCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes APIVersion that has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	apiCmd.MarkFlagRequired("api-version")
	apiCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes resource Kind name. (e.g AppService)")
	apiCmd.MarkFlagRequired("kind")

	return apiCmd
}

func apiRun(cmd *cobra.Command, args []string) {
	// Create and validate new resource
	projutil.MustInProjectRoot()
	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		log.Fatal(err)
	}

	absProjectPath := projutil.MustGetwd()

	cfg := &input.Config{
		Repo:           projutil.CheckAndGetCurrPkg(),
		AbsProjectPath: absProjectPath,
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&scaffold.Types{Resource: r},
		&scaffold.AddToScheme{Resource: r},
		&scaffold.Register{Resource: r},
		&scaffold.Doc{Resource: r},
		&scaffold.Cr{Resource: r},
		&scaffold.Crd{Resource: r},
	)
	if err != nil {
		log.Fatalf("add scaffold failed: (%v)", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(r, absProjectPath); err != nil {
		log.Fatalf("failed to update the RBAC manifest for the resource (%v, %v): %v", r.APIVersion, r.Kind, err)
	}

	// Run k8s codegen for deepcopy
	generate.K8sCodegen()
}
