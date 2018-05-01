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

package generate

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/generator"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

const (
	configYaml = "./config/config.yaml"
)

var (
	image   string
	version string
)

func NewGenerateOlmCatalogCmd() *cobra.Command {
	olmCatalogCmd := &cobra.Command{
		Use:   "olm-catalog",
		Short: "Generates OLM Catalog manifests",
		Long: `olm-catalog generator generates the following OLM Catalog manifests needed to create a catalog entry in OLM:
- Cluster Service Version: deploy/olm-catalog/csv.yaml
- Package: deploy/olm-catalog/package.yaml
- Custom Resource Definition: deploy/olm-catalog/crd.yaml

The following flags are required:
--image: The container image name to set in the CSV to deploy the operator
--version: The version of the current CSV

For example:
	$ operator-sdk generate olm-catalog --image=quay.io/example/operator:v0.0.1 --version=0.0.1
`,
		Run: olmCatalogFunc,
	}
	olmCatalogCmd.Flags().StringVar(&image, "image", "", "The container image name to set in the CSV to deploy the operator e.g: quay.io/example/operator:v0.0.1")
	olmCatalogCmd.MarkFlagRequired("image")
	olmCatalogCmd.Flags().StringVar(&version, "version", "", "The version of the current CSV e.g: 0.0.1")
	olmCatalogCmd.MarkFlagRequired("version")

	return olmCatalogCmd
}

func olmCatalogFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("olm-catalog command doesn't accept any arguments."))
	}
	verifyFlags()

	fmt.Fprintln(os.Stdout, "Generating OLM catalog manifests")

	c := &generator.Config{}
	fp, err := ioutil.ReadFile(configYaml)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to read config file %v: (%v)", configYaml, err))
	}
	if err = yaml.Unmarshal(fp, c); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to unmarshal config file %v: (%v)", configYaml, err))
	}

	// Generate OLM catalog manifests
	if err = generator.RenderOlmCatalog(c, image, version); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to generate deploy/olm-catalog: (%v)", err))
	}
}

func verifyFlags() {
	if len(image) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--image must not have empty value"))
	}
	if len(version) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--version must not have empty value"))
	}
}
