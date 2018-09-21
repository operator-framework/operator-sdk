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
	"bytes"
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/spf13/cobra"
)

func NewControllerCmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   "controller",
		Short: "Adds a new controller pkg",
		Long: `operator-sdk add controller --kind=<kind> --api-version=<group/version> creates a new
controller pkg under pkg/controller/<kind> that, by default, reconciles on a custom resource for the specified apiversion and kind.
The controller will expect to use the custom resource type that should already be defined under pkg/apis/<group>/<version> 
via the "operator-sdk add api --kind=<kind> --api-version=<group/version>" command.
This command must be run from the project root directory.
If the controller pkg for that Kind already exists at pkg/controller/<kind> then the command will not overwrite and return an error.

Example:
	$ operator-sdk add controller --api-version=app.example.com/v1alpha1 --kind=AppService
	$ tree pkg/controller
	pkg/controller/
	├── add_appservice.go
	├── appservice
	│   └── appservice_controller.go
	└── controller.go

`,
		Run: controllerRun,
	}

	apiCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes APIVersion that has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	apiCmd.MarkFlagRequired("api-version")
	apiCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes resource Kind name. (e.g AppService)")
	apiCmd.MarkFlagRequired("kind")

	return apiCmd
}

func controllerRun(cmd *cobra.Command, args []string) {
	projectPath := cmdutil.MustInProjectRoot()
	fullProjectPath := mustGetwd()

	// Create and validate new resource
	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		log.Fatal(err)
	}

	// Must be controller for a new kind: pkg/controller/<kind>/<kind>_controller.go shouldn't exist
	kindControllerFileName := r.LowerKind + "_controller.go"
	pkgControllerDir := filepath.Join(fullProjectPath, "pkg", "controller", r.LowerKind)
	mustNotExist(filepath.Join(pkgControllerDir, kindControllerFileName))

	// Scaffold pkg/controller/add_<kind>.go
	filePath := filepath.Join(fullProjectPath, "pkg", "controller", "add_"+r.LowerKind+".go")
	codeGen := scaffold.NewAddControllerCodegen(&scaffold.AddControllerInput{ProjectPath: projectPath, Resource: r})
	buf := &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// Scaffold pkg/controller/<kind> directory
	if err := os.MkdirAll(pkgControllerDir, cmdutil.DefaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", pkgControllerDir, err)
	}

	// Scaffold pkg/controller/<kind>/<kind>_controller.go
	filePath = filepath.Join(pkgControllerDir, kindControllerFileName)
	codeGen = scaffold.NewControllerKindCodegen(&scaffold.ControllerKindInput{ProjectPath: projectPath, Resource: r})
	buf = &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

}
