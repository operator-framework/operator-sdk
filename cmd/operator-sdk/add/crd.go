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

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newAddCRDCmd - add crd command
func newAddCRDCmd() *cobra.Command {
	c := &scaffold.AddCRDCmd{}

	crdCmd := &cobra.Command{
		Use:   "crd",
		Short: "Adds a Custom Resource Definition (CRD) and the Custom Resource (CR) files",
		Long: `The operator-sdk add crd command will create a Custom Resource Definition (CRD) and the Custom Resource (CR) files for the specified api-version and kind.

Generated CRD filename: deploy/crds/<group>_<version>_<kind>_crd.yaml
Generated CR  filename: deploy/crds/<group>_<version>_<kind>_cr.yaml

	--api-version and --kind are required flags to generate the new operator application.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}
			projutil.MustInProjectRoot()
			return c.Run()
		},
	}

	crdCmd.Flags().StringVar(&c.APIVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	if err := crdCmd.MarkFlagRequired("api-version"); err != nil {
		log.Fatalf("Failed to mark `api-version` flag for `add crd` subcommand as required")
	}
	crdCmd.Flags().StringVar(&c.Kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService)")
	if err := crdCmd.MarkFlagRequired("kind"); err != nil {
		log.Fatalf("Failed to mark `kind` flag for `add crd` subcommand as required")
	}

	return crdCmd
}
