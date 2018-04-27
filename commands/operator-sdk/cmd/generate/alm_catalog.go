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

func NewGenerateAlmCatalogCmd() *cobra.Command {
	almCatalogCmd := &cobra.Command{
		Use:   "alm-catalog",
		Short: "Generates ALM Catalog manifests",
		Long: `alm-catalog generator generates the following ALM Catalog manifests needed to create a catalog entry in ALM:
- Cluster Service Version: deploy/alm-catalog/csv.yaml
- Package: deploy/alm-catalog/package.yaml
- Custom Resource Definition: deploy/alm-catalog/crd.yaml

The following flags are required:
--image: The container image name to set in the CSV to deploy the operator
--version: The version of the current CSV

For example:
	$ operator-sdk generate alm-catalog --image=quay.io/example/operator:v0.0.1 --version=0.0.1
`,
		Run: almCatalogFunc,
	}
	almCatalogCmd.Flags().StringVar(&image, "image", "", "The container image name to set in the CSV to deploy the operator e.g: quay.io/example/operator:v0.0.1")
	almCatalogCmd.MarkFlagRequired("image")
	almCatalogCmd.Flags().StringVar(&version, "version", "", "The version of the current CSV e.g: 0.0.1")
	almCatalogCmd.MarkFlagRequired("version")

	return almCatalogCmd
}

func almCatalogFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("alm-catalog command doesn't accept any arguments."))
	}
	verifyFlags()

	fmt.Fprintln(os.Stdout, "Generating ALM catalog manifests")

	c := &generator.Config{}
	fp, err := ioutil.ReadFile(configYaml)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to read config file %v: (%v)", configYaml, err))
	}
	if err = yaml.Unmarshal(fp, c); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to unmarshal config file %v: (%v)", configYaml, err))
	}

	// Generate ALM catalog manifests
	if err = generator.RenderAlmCatalog(c, image, version); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to generate deploy/alm-catalog: (%v)", err))
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
