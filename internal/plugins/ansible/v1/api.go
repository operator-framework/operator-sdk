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

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/util"
)

const (
	crdVersionFlag       = "crd-version"
	generatePlaybookFlag = "generate-playbook"
	generateRoleFlag     = "generate-role"

	defaultCrdVersion = "v1"
)

type createAPIOptions struct {
	CRDVersion         string
	DoRole, DoPlaybook bool
}

func (opts createAPIOptions) UpdateResource(res *resource.Resource) {
	res.API = &resource.API{
		CRDVersion: opts.CRDVersion,
		Namespaced: true,
	}

	// Ensure that Path is empty and Controller false as this is not a Go project
	res.Path = ""
	res.Controller = false
}

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config   config.Config
	resource *resource.Resource
	options  createAPIOptions
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Scaffold a Kubernetes API in which the controller is an Ansible role or playbook.

    - generates a Custom Resource Definition and sample
    - Updates watches.yaml
    - optionally generates Ansible Role tree
    - optionally generates Ansible playbook

    For the scaffolded operator to be runnable with no changes, specify either --generate-role or --generate-playbook.

`
	subcmdMeta.Examples = fmt.Sprintf(`# Create a new API, without Ansible roles or playbooks
  $ %[1]s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService

  $ %[1]s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --generate-role

  $ %[1]s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --generate-playbook

  $ %[1]s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService
      --generate-playbook
      --generate-role
`, cliMeta.CommandName)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false
	fs.StringVar(&p.options.CRDVersion, crdVersionFlag, defaultCrdVersion, "crd version to generate")
	fs.BoolVar(&p.options.DoRole, generateRoleFlag, false, "Generate an Ansible role skeleton.")
	fs.BoolVar(&p.options.DoPlaybook, generatePlaybookFlag, false, "Generate an Ansible playbook. If passed with --generate-role, the playbook will invoke the role.")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	p.options.UpdateResource(p.resource)

	if err := p.resource.Validate(); err != nil {
		return err
	}

	// Check that resource doesn't have the API scaffolded
	if res, err := p.config.GetResource(p.resource.GVK); err == nil && res.HasAPI() {
		return errors.New("the API resource already exists")
	}

	// Check that the provided group can be added to the project
	if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group set 'multigroup: true' in your PROJECT file")
	}

	// Selected CRD version must match existing CRD versions.
	if pluginutil.HasDifferentCRDVersion(p.config, p.resource.API.CRDVersion) {
		return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q", p.resource.API.CRDVersion)
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := util.RemoveKustomizeCRDManifests(); err != nil {
		return fmt.Errorf("error removing kustomization CRD manifests: %v", err)
	}
	if err := util.UpdateKustomizationsCreateAPI(); err != nil {
		return fmt.Errorf("error updating kustomization.yaml files: %v", err)
	}

	scaffolder := scaffolds.NewCreateAPIScaffolder(p.config, *p.resource, p.options.DoRole, p.options.DoPlaybook)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return err
	}

	return nil
}
