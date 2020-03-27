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
	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	crdVersion string
)

func newGenerateCRDsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crds",
		Short: "Generates CRDs for API's",
		Long: `generate crds generates CRDs or updates them if they exist,
under deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI
V3 validation YAML is generated as a 'validation' object.

Example:

	$ operator-sdk generate crds
	$ tree deploy/crds
	├── deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml
	├── deploy/crds/app.example.com_appservices_crd.yaml
`,
		RunE: crdsFunc,
	}

	cmd.Flags().StringVar(&crdVersion, "crd-version", gencrd.DefaultCRDVersion, "CRD version to generate")
	return cmd
}

func crdsFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}

	// Skip usage printing on error, since this command will never fail from
	// improper CLI usage.
	if err := genutil.CRDGen(crdVersion); err != nil {
		log.Fatal(err)
	}
	return nil
}
