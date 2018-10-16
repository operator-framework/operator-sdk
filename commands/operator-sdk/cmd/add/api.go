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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/generate"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
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
	// Create and validate new resource
	cmdutil.MustInProjectRoot()
	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		log.Fatal(err)
	}

	absProjectPath := cmdutil.MustGetwd()

	cfg := &input.Config{
		Repo:           cmdutil.CheckAndGetCurrPkg(),
		AbsProjectPath: absProjectPath,
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(cfg,
		&scaffold.Types{Resource: r},
		&scaffold.AddToScheme{Resource: r},
		&scaffold.Register{Resource: r},
		&scaffold.Doc{Resource: r},
		&scaffold.Cr{Resource: r},
		&scaffold.Crd{Resource: r},
	)
	if err != nil {
		log.Fatalf("add scaffold failed: (%v)", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := updateRoleForResource(r, absProjectPath); err != nil {
		log.Fatalf("failed to update the RBAC manifest for the resource (%v, %v): %v", r.APIVersion, r.Kind, err)
	}

	// Run k8s codegen for deepcopy
	generate.K8sCodegen()
}

func updateRoleForResource(r *scaffold.Resource, absProjectPath string) error {
	// append rbac rule to deploy/role.yaml
	roleFilePath := filepath.Join(absProjectPath, "deploy", "role.yaml")
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
	case *rbacv1.Role:
		pr := &rbacv1.PolicyRule{}
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
			// Using "*" to allow access to the resource and all its subresources e.g "memcacheds" and "memcacheds/finalizers"
			// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			pr.Resources = []string{"*"}
			pr.Verbs = []string{"*"}
			role.Rules = append(role.Rules, *pr)
		}
		// update role.yaml
		d, err := json.Marshal(&role)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		m := &map[string]interface{}{}
		err = yaml.Unmarshal(d, m)
		data, err := yaml.Marshal(m)
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
