// Copyright 2020 The Operator-SDK Authors
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

package ansible

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"
	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

type createAPIPlugin struct {
	config *config.Config

	// Flag values.
	// Contains group, version, and kind
	resource *scaffold.ResourceOptions
	// Either v1 or v1beta1
	crdVersion string
	// Create a playbook
	doPlaybook bool
	// Only create a CRD. Setting this emulates the old 'add crd' command.
	doCRDOnly bool
	// Force creation of a CRD/CR.
	force bool
}

var _ plugin.CreateAPI = &createAPIPlugin{}

func (p *createAPIPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a Kubernetes API by creating or updating an Ansible Resource definition.
`
	ctx.Examples = fmt.Sprintf(`  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
%s create api --group ship --version v1beta1 --kind Frigate
`,
		ctx.CommandName)
}

func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet) {
	// resource flags.
	p.resource = &scaffold.ResourceOptions{}
	fs.StringVar(&p.resource.Group, "group", "", "resource Group")
	fs.StringVar(&p.resource.Version, "version", "", "resource Version")
	fs.StringVar(&p.resource.Kind, "kind", "", "resource Kind")

	fs.StringVar(&p.crdVersion, "crd-version", gencrd.DefaultCRDVersion, "CRD version to generate")

	fs.BoolVar(&p.force, "force", false, "attempt to create resource even if it already exists")
	fs.BoolVar(&p.doPlaybook, "playbook", false, "Generate a playbook skeleton.")
	fs.BoolVar(&p.doCRDOnly, "crd-only", false, "only generate a CRD")
}

func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.config = c
}

func (p *createAPIPlugin) Run() error {
	if err := p.Validate(); err != nil {
		return err
	}

	return p.Scaffold()
}

func (p *createAPIPlugin) Validate() error {
	if err := p.resource.Validate(); err != nil {
		return err
	}

	// Check that resource doesn't exist or flag force was set
	if !p.force && p.config.HasResource(p.resource.GVK()) {
		return errors.New("resource for API already exists in configuration")
	}

	// Check that the provided group can be added to the project
	// TODO(estroz): change doc link.
	if !p.config.MultiGroup && len(p.config.Resources) != 0 && !p.config.HasGroup(p.resource.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group visit %s",
			"kubebuilder.io/migration/multi-group.html")
	}

	return nil
}

func (p *createAPIPlugin) Scaffold() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg := &input.Config{
		AbsProjectPath: wd,
		ProjectName:    p.config.Repo,
	}

	resource, err := p.resource.NewResource()
	if err != nil {
		return fmt.Errorf("invalid resource: %v", err)
	}
	p.config.AddResource(p.resource.GVK())

	if !p.doCRDOnly {
		if err = p.scaffoldNewProject(cfg, resource); err != nil {
			return err
		}
	}

	cr := &scaffold.CR{Resource: resource}
	if p.doCRDOnly {
		cr.IfExistsAction = input.Skip
	}

	log.Info("Generating CustomResourceDefinition manifests")

	if err = (&scaffold.Scaffold{}).Execute(cfg, cr); err != nil {
		return fmt.Errorf("error scaffolding Custom Resource: %v", err)
	}

	if err = generateCRD(resource, p.crdVersion); err != nil {
		return err
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, wd); err != nil {
		return fmt.Errorf("error updating Role for resource %s: %v", resource, err)
	}

	return nil
}

func (p *createAPIPlugin) scaffoldNewProject(cfg *input.Config, resource *scaffold.Resource) error {

	roleFiles := ansible.RolesFiles{Resource: *resource}
	roleTemplates := ansible.RolesTemplates{Resource: *resource}

	err := (&scaffold.Scaffold{}).Execute(cfg,
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.CR{Resource: resource},
		&ansible.BuildDockerfile{GeneratePlaybook: p.doPlaybook},
		&ansible.RolesReadme{Resource: *resource},
		&ansible.RolesMetaMain{Resource: *resource},
		&roleFiles,
		&roleTemplates,
		&ansible.RolesVarsMain{Resource: *resource},
		&ansible.MoleculeTestLocalConverge{Resource: *resource},
		&ansible.RolesDefaultsMain{Resource: *resource},
		&ansible.RolesTasksMain{Resource: *resource},
		&ansible.MoleculeDefaultMolecule{},
		&ansible.MoleculeDefaultPrepare{},
		&ansible.MoleculeDefaultConverge{
			GeneratePlaybook: p.doPlaybook,
			Resource:         *resource,
		},
		&ansible.MoleculeDefaultVerify{},
		&ansible.RolesHandlersMain{Resource: *resource},
		&ansible.Watches{
			GeneratePlaybook: p.doPlaybook,
			Resource:         *resource,
		},
		&ansible.DeployOperator{},
		&ansible.Travis{},
		&ansible.MoleculeTestLocalMolecule{},
		&ansible.MoleculeTestLocalPrepare{},
		&ansible.MoleculeTestLocalVerify{},
		&ansible.MoleculeClusterMolecule{Resource: *resource},
		&ansible.MoleculeClusterCreate{},
		&ansible.MoleculeClusterPrepare{Resource: *resource},
		&ansible.MoleculeClusterConverge{},
		&ansible.MoleculeClusterVerify{Resource: *resource},
		&ansible.MoleculeClusterDestroy{Resource: *resource},
		&ansible.MoleculeTemplatesOperator{},
	)
	if err != nil {
		return fmt.Errorf("error scaffolding project: %v", err)
	}

	// Remove placeholders from empty directories
	for _, file := range []string{roleFiles.Path, roleTemplates.Path} {
		err = os.Remove(filepath.Join(cfg.AbsProjectPath, file))
		if err != nil {
			return fmt.Errorf("error removing placeholder file %s: %v", file, err)
		}
	}

	// Decide on playbook.
	if p.doPlaybook {
		log.Infof("Generating Ansible playbook")

		err := (&scaffold.Scaffold{}).Execute(cfg,
			&ansible.Playbook{Resource: *resource},
		)
		if err != nil {
			return fmt.Errorf("error scaffolding ansible playbook: %v", err)
		}
	}

	return nil
}

func generateCRD(resource *scaffold.Resource, crdVersion string) error {
	cfg := gen.Config{
		Inputs:    map[string]string{gencrd.CRDsDirKey: scaffold.CRDsDir},
		OutputDir: scaffold.CRDsDir,
	}
	crd := gencrd.NewCRDNonGo(cfg, *resource, crdVersion)
	if err := crd.Generate(); err != nil {
		return fmt.Errorf("error generating CRD for %s: %w", resource, err)
	}
	return nil
}
