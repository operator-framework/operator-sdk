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

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/generator"
	"github.com/spf13/cobra"
)

var (
	apiVersion string
	kind       string
)

func NewGenerateCrdCmd() *cobra.Command {
	crdCmd := &cobra.Command{
		Use:   "crd",
		Short: "Generates a custom resource definition",
		Long: `generates a custom resource definition (CRD) file for the .
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

	fmt.Fprintln(os.Stdout, "Generating custom resource definition (CRD) file")

	// Generate CRD file
	if err := generator.RenderDeployCrdFile(apiVersion, kind); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to generate CRD file: (%v)", err))
	}
}

func verifyCrdFlags() {
	if len(apiVersion) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--api-version must not have empty value"))
	}
	if len(kind) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("--kind must not have empty value"))
	}
}
