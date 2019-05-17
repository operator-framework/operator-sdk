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

package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/genutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
)

type AddCmd struct {
	APIVersion string
	Kind       string
}

type AddAPICmd struct {
	AddCmd
}

func (c *AddAPICmd) Run() error {

	if err := c.verifyFlags(); err != nil {
		return err
	}

	log.Infof("Generating api version %s for kind %s.", c.APIVersion, c.Kind)

	// Create and validate new resource.
	r, err := scaffold.NewResource(c.APIVersion, c.Kind)
	if err != nil {
		return err
	}

	absProjectPath := projutil.MustGetwd()
	s := &scaffold.Scaffold{
		Repo:           projutil.CheckAndGetProjectGoPkg(),
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	err = s.Execute(&input.Config{},
		&scaffold.Types{Resource: r},
		&scaffold.AddToScheme{Resource: r},
		&scaffold.Register{Resource: r},
		&scaffold.Doc{Resource: r},
		&scaffold.CR{Resource: r},
		&scaffold.CRD{Resource: r, IsOperatorGo: projutil.IsOperatorGo()},
	)
	if err != nil {
		return fmt.Errorf("api scaffold failed: (%v)", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(r, absProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): (%v)", r.APIVersion, r.Kind, err)
	}

	// Run k8s codegen for deepcopy
	if err := genutil.K8sCodegen(); err != nil {
		return err
	}

	// Generate a validation spec for the new CRD.
	if err := genutil.OpenAPIGen(); err != nil {
		return err
	}

	log.Info("API generation complete.")
	return nil
}

type AddControllerCmd struct {
	AddCmd

	CustomAPIImport string
}

func (c *AddControllerCmd) Run() error {

	if err := c.verifyFlags(); err != nil {
		return err
	}

	log.Infof("Generating controller version %s for kind %s.", c.APIVersion, c.Kind)

	// Create and validate new resource
	r, err := scaffold.NewResource(c.APIVersion, c.Kind)
	if err != nil {
		return err
	}

	absProjectPath := projutil.MustGetwd()
	s := &scaffold.Scaffold{
		Repo:           projutil.CheckAndGetProjectGoPkg(),
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	err = s.Execute(&input.Config{},
		&scaffold.ControllerKind{Resource: r, CustomImport: c.CustomAPIImport},
		&scaffold.AddController{Resource: r},
	)
	if err != nil {
		return fmt.Errorf("controller scaffold failed: (%v)", err)
	}

	log.Info("Controller generation complete.")
	return nil
}

type AddCRDCmd struct {
	AddCmd
}

func (c *AddCRDCmd) Run() error {

	if err := c.verifyFlags(); err != nil {
		return err
	}

	log.Infof("Generating Custom Resource Definition (CRD) version %s for kind %s.", c.APIVersion, c.Kind)

	// generate CR/CRD file
	resource, err := scaffold.NewResource(c.APIVersion, c.Kind)
	if err != nil {
		return err
	}

	absProjectPath := projutil.MustGetwd()
	s := &scaffold.Scaffold{
		Repo:           projutil.CheckAndGetProjectGoPkg(),
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	err = s.Execute(&input.Config{},
		&scaffold.CRD{
			Input:        input.Input{IfExistsAction: input.Skip},
			Resource:     resource,
			IsOperatorGo: projutil.IsOperatorGo(),
		},
		&scaffold.CR{
			Input:    input.Input{IfExistsAction: input.Skip},
			Resource: resource,
		},
	)
	if err != nil {
		return fmt.Errorf("crd scaffold failed: (%v)", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, s.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): (%v)", resource.APIVersion, resource.Kind, err)
	}

	log.Info("CRD generation complete.")
	return nil
}

func (c *AddCmd) verifyFlags() error {
	if len(c.APIVersion) == 0 {
		return fmt.Errorf("value of --api-version must not have empty value")
	}
	if len(c.Kind) == 0 {
		return fmt.Errorf("value of --kind must not have empty value")
	}
	kindFirstLetter := string(c.Kind[0])
	if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
		return fmt.Errorf("value of --kind must start with an uppercase letter")
	}
	if strings.Count(c.APIVersion, "/") != 1 {
		return fmt.Errorf("value of --api-version has wrong format (%v); format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", c.APIVersion)
	}
	return nil
}
