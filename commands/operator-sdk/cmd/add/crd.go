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
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewAddCrdCmd - add crd command
func NewAddCrdCmd() *cobra.Command {
	crdCmd := &cobra.Command{
		Use:   "crd",
		Short: "Adds a custom resource definition (CRD) and the custom resource (CR) files",
		Long: `The operator-sdk add crd command will create a custom resource definition (CRD) and the custom resource (CR) files for the specified api-version and kind.

Generated CRD filename: <project-name>/deploy/crds/<group>_<version>_<kind>_crd.yaml
Generated CR  filename: <project-name>/deploy/crds/<group>_<version>_<kind>_cr.yaml

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
	cfg := &input.Config{
		AbsProjectPath: projutil.MustGetwd(),
	}
	if len(args) != 0 {
		log.Fatal("crd command doesn't accept any arguments")
	}
	verifyCrdFlags()
	verifyCrdDeployPath()

	log.Infof("Generating Custom Resource Definition (CRD) version %s for kind %s.", apiVersion, kind)

	// generate CR/CRD file
	resource, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		log.Fatal(err)
	}

	s := scaffold.Scaffold{}
	err = s.Execute(cfg,
		&scaffold.Crd{Resource: resource},
		&scaffold.Cr{Resource: resource},
	)
	if err != nil {
		log.Fatalf("add scaffold failed: (%v)", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, cfg.AbsProjectPath); err != nil {
		log.Fatalf("failed to update the RBAC manifest for the resource (%v, %v): (%v)", resource.APIVersion, resource.Kind, err)
	}

	log.Info("CRD generation complete.")
}

func verifyCrdFlags() {
	if len(apiVersion) == 0 {
		log.Fatal("--api-version must not have empty value")
	}
	if len(kind) == 0 {
		log.Fatal("--kind must not have empty value")
	}
	kindFirstLetter := string(kind[0])
	if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
		log.Fatal("--kind must start with an uppercase letter")
	}
	if strings.Count(apiVersion, "/") != 1 {
		log.Fatalf("api-version has wrong format (%v); format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", apiVersion)
	}
}

// verifyCrdDeployPath checks if the path <project-name>/deploy sub-directory is exists, and that is rooted under $GOPATH
func verifyCrdDeployPath() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to determine the full path of the current directory: (%v)", err)
	}
	// check if the deploy sub-directory exist
	_, err = os.Stat(filepath.Join(wd, scaffold.DeployDir))
	if err != nil {
		log.Fatalf("the path (./%v) does not exist. run this command in your project directory", scaffold.DeployDir)
	}
}
