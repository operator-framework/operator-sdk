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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newGenerateOpenAPICmd() *cobra.Command {
	return &cobra.Command{
		Hidden: true,
		Use:    "openapi",
		Short:  "Generates OpenAPI specs for API's",
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
}

const deprecationTemplate = "\033[1;36m%s\033[0m"

//nolint:lll
func openAPIFunc(cmd *cobra.Command, args []string) error {
	fmt.Printf(deprecationTemplate, `[Deprecation notice] The 'operator-sdk generate openapi' command is deprecated!

 - To generate CRDs, use 'operator-sdk generate crds'.
 - To generate Go OpenAPI code, use 'openapi-gen'. For example:

      # Build the latest openapi-gen from source
      which ./bin/openapi-gen > /dev/null || go build -o ./bin/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

      # Run openapi-gen for each of your API group/version packages
      ./bin/openapi-gen --logtostderr=true -o "" -i ./pkg/apis/<group>/<version> -O zz_generated.openapi -p ./pkg/apis/<group>/<version> -h ./hack/boilerplate.go.txt -r "-"

`)

	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}

	if err := genutil.OpenAPIGen(); err != nil {
		log.Fatal(err)
	}

	// Hardcode "v1beta1" here because we never want to change the functionality of this deprecated function.
	if err := genutil.CRDGen("v1beta1"); err != nil {
		log.Fatal(err)
	}

	return nil
}
