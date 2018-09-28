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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/generate"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	apiVersion string
	kind       string
)

func NewApiCmd() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Adds a new api definition under pkg/apis",
		Long: `operator-sdk add api --kind=<kind> --api-version=<group/version> creates the
api definition for a new custom resource under pkg/apis. This command must be run from the project root directory.
If the api already exists at pkg/apis/<group>/<version> then the command will not overwrite and return an error.

Example:
	$ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService
	$ tree pkg/apis
	pkg/apis/
	├── addtoscheme_app_appservice.go
	├── apis.go
	└── app
		└── v1alpha1
			├── doc.go
			├── register.go
			├── types.go

`,
		Run: apiRun,
	}

	apiCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes APIVersion that has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	apiCmd.MarkFlagRequired("api-version")
	apiCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes resource Kind name. (e.g AppService)")
	apiCmd.MarkFlagRequired("kind")

	return apiCmd
}

func apiRun(cmd *cobra.Command, args []string) {
	projectPath := cmdutil.MustInProjectRoot()
	fullProjectPath := mustGetwd()

	// Create and validate new resource
	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		log.Fatal(err)
	}

	// Must be new Kind: pkg/apis/<group>/<version>/<kind>_types.go shouldn't exist
	typesFileName := r.LowerKind + "_types.go"
	pkgApisDir := filepath.Join(fullProjectPath, "pkg", "apis", r.Group, r.Version)
	mustNotExist(filepath.Join(pkgApisDir, typesFileName))

	// TODO: Don't rewrite existing boilerplate files(register.go, doc.go) for new Kind
	// scaffold pkg/apis/<group>/<version> directory
	if err := os.MkdirAll(pkgApisDir, cmdutil.DefaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", pkgApisDir, err)
	}

	// scaffold pkg/apis/addtoscheme_<group>_<version>.go
	fileName := "addtoscheme_" + r.Group + "_" + r.Version + ".go"
	filePath := filepath.Join(fullProjectPath, "pkg", "apis", fileName)
	codeGen := scaffold.NewAddToSchemeCodegen(&scaffold.AddToSchemeInput{ProjectPath: projectPath, Resource: r})
	buf := &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// scaffold pkg/apis/<group>/<version>/<kind>_types.go
	filePath = filepath.Join(pkgApisDir, typesFileName)
	codeGen = scaffold.NewTypesCodegen(&scaffold.TypesInput{ProjectPath: projectPath, Resource: r})
	buf = &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// scaffold pkg/apis/<group>/<version>/register.go
	filePath = filepath.Join(pkgApisDir, "register.go")
	codeGen = scaffold.NewRegisterCodegen(&scaffold.RegisterInput{ProjectPath: projectPath, Resource: r})
	buf = &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// scaffold pkg/apis/<group>/<version>/doc.go
	filePath = filepath.Join(pkgApisDir, "doc.go")
	codeGen = scaffold.NewDocCodegen(&scaffold.DocInput{ProjectPath: projectPath, Resource: r})
	buf = &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// Scaffold crd.yaml and cr.yaml
	// mkdir deploy/crds
	crdsDir := filepath.Join(fullProjectPath, "deploy", "crds")
	if err := os.MkdirAll(crdsDir, cmdutil.DefaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", crdsDir, err)
	}

	// scaffold deploy/crds/<group>_<version>_<kind>_crd.yaml
	crdFileName := r.Group + "_" + r.Version + "_" + r.LowerKind + "_crd.yaml"
	filePath = filepath.Join(crdsDir, crdFileName)
	codeGen = scaffold.NewCrdCodegen(&scaffold.CrdInput{Resource: r})
	buf = &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// scaffold deploy/crds/<group>-<version>-<kind>_cr.yaml
	crFileName := r.Group + "_" + r.Version + "_" + r.LowerKind + "_cr.yaml"
	filePath = filepath.Join(crdsDir, crFileName)
	codeGen = scaffold.NewCrCodegen(&scaffold.CrInput{Resource: r})
	buf = &bytes.Buffer{}
	if err := codeGen.Render(buf); err != nil {
		log.Fatalf("failed to render the template for (%v): %v", filePath, err)
	}
	if err := writeFileAndPrint(filePath, buf.Bytes(), cmdutil.DefaultFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", filePath, err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := updateRoleForResource(r, fullProjectPath); err != nil {
		log.Fatalf("failed to update the RBAC manifest for the resource (%v, %v): %v", r.APIVersion, r.Kind, err)
	}

	// Run k8s codegen for deepcopy
	generate.K8sCodegen()
}

func updateRoleForResource(r *scaffold.Resource, fullProjectPath string) error {
	// append rbac rule to deploy/role.yaml
	roleFilePath := filepath.Join(fullProjectPath, "deploy", "role.yaml")
	roleYAML, err := ioutil.ReadFile(roleFilePath)
	if err != nil {
		return fmt.Errorf("failed to read role manifest %v: %v", roleFilePath, err)
	}
	obj, _, err := cgoscheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	// TODO: use rbac/v1.
	case *rbacv1beta1.Role:
		pr := &rbacv1beta1.PolicyRule{}
		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.FullGroup {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}
		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Resource {
				log.Printf("deploy/role.yaml RBAC rules already up to date for the resource (%v, %v)", r.APIVersion, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Resource)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.FullGroup}
			pr.Resources = []string{r.Resource}
			pr.Verbs = []string{"*"}
			role.Rules = append(role.Rules, *pr)
		}
		// update role.yaml
		data, err := yaml.Marshal(&role)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		if err := ioutil.WriteFile(roleFilePath, data, cmdutil.DefaultFileMode); err != nil {
			return fmt.Errorf("failed to update %v: %v", roleFilePath, err)
		}
	default:
		return errors.New("failed to parse role.yaml as a role")
	}
	// not reachable
	return nil
}

// TODO: Make the utils below common in pkg/cmd/util
// Writes file to a given path and data buffer, as well as prints out a message confirming creation of a file
func writeFileAndPrint(filePath string, data []byte, fileMode os.FileMode) error {
	if err := ioutil.WriteFile(filePath, data, fileMode); err != nil {
		return err
	}
	fmt.Printf("Create %v \n", filePath)
	return nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to determine current working directory: %v", err)
	}
	return wd
}

// TODO: combine with writeFileAndPrint to return error if file already exists
func mustNotExist(path string) {
	_, err := os.Stat(path)
	if err == nil {
		log.Fatalf("%v already exists", path)
	}
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("failed to stat %v: %v", path, err)
	}
}
