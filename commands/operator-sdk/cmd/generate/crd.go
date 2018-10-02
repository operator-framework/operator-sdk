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
	"os"
	"path/filepath"
	"strings"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/generator"

	"github.com/spf13/cobra"
)

var (
	apiVersion string
	kind       string
)

const (
	goDir        = "GOPATH"
	deployCrdDir = "deploy"
)

func NewGenerateCrdCmd() *cobra.Command {
	crdCmd := &cobra.Command{
		Use:   "crd",
		Short: "Generates a custom resource definition (CRD) and the custom resource (CR) files",
		Long: `The operator-sdk generate command will create a custom resource definition (CRD) and the custom resource (CR) files for the specified api-version and kind.

Generated CRD filename: <project-name>/deploy/<group>_<version>_<kind>_crd.yaml
Generated CR  filename: <project-name>/deploy/<group>_<version>_<kind>_cr.yaml

	<project-name>/deploy path must already exist
	--api-version and --kind are required flags to generate the new operator application.
`,
		Run: crdFunc,
	}
	crdCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	crdCmd.MarkFlagRequired("api-version")
	crdCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService)")
	crdCmd.MarkFlagRequired("kind")
	return crdCmd
}

func crdFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("crd command doesn't accept any arguments."))
	}
	verifyCrdFlags()
	verifyCrdDeployPath()

	fmt.Fprintln(os.Stdout, "Generating custom resource definition (CRD) file")

	// generate CR/CRD file
	wd, err := os.Getwd()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, err)
	}
	if err := generator.RenderDeployCrdFiles(filepath.Join(wd, deployCrdDir), apiVersion, kind); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to generate CRD and CR files: (%v)", err))
	}
}

func verifyCrdFlags() {
	if len(apiVersion) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--api-version must not have empty value"))
	}
	if len(kind) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--kind must not have empty value"))
	}
	kindFirstLetter := string(kind[0])
	if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--kind must start with an uppercase letter"))
	}
	if strings.Count(apiVersion, "/") != 1 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("api-version has wrong format (%v); format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", apiVersion))
	}
}

// verifyCrdDeployPath checks if the path <project-name>/deploy sub-directory is exists, and that is rooted under $GOPATH
func verifyCrdDeployPath() {
	wd, err := os.Getwd()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to determine the full path of the current directory: %v", err))
	}
	// check if the deploy sub-directory exist
	_, err = os.Stat(filepath.Join(wd, deployCrdDir))
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("the path (./%v) does not exist. run this command in your project directory", deployCrdDir))
	}
}
