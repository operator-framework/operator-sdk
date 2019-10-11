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

package generate

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/internal/genutil"
	"github.com/spf13/cobra"
)

var (
	skipGeneration bool
	group          []string
)

func newGenerateOpenAPICmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   "openapi",
		Short: "Generates OpenAPI specs for API's",
		Long: `generate openapi generates OpenAPI validation specs in Go from tagged types
in all pkg/apis/<group>/<version> directories. Go code is generated under
pkg/apis/<group>/<version>/zz_generated.openapi.go. CRD's are generated, or
updated if they exist for a particular group + version + kind, under
deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI V3 validation YAML
is generated as a 'validation' object.

Example:
	$ operator-sdk generate openapi
	$ tree pkg/apis
	pkg/apis/
	└── app
		└── v1alpha1
			├── zz_generated.openapi.go
	$ tree deploy/crds
	├── deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml
	├── deploy/crds/app.example.com_appservices_crd.yaml
`,
		RunE: openAPIFunc,
	}

	apiCmd.Flags().BoolVar(&skipGeneration, "skip-generation", false, "Skips creating scafold for specified GKV")
	apiCmd.Flags().StringSliceVar(&group, "group", nil, "Specify group to ignore genrating openapis")
	return apiCmd
}

func openAPIFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}

	if skipGeneration {
		return genutil.OpenAPIGenWithIgnoreFlag(group)
	}
	if len(group) > 0 {
		return fmt.Errorf("can not use --group flag without --skip-generation flag")
	}
	return genutil.OpenAPIGen()
}
