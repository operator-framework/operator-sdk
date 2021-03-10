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
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests"
	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
)

const defaultCRDVersion = "v1"

type createAPIPSubcommand struct {
	config  config.Config
	options createOptions

	resource *resource.Resource

	// Ansible-specific flags
	doRole, doPlaybook bool
}

var (
	_ plugin.CreateAPISubcommand = &createAPIPSubcommand{}
	_ cmdutil.RunOptions         = &createAPIPSubcommand{}
)

// UpdateContext injects documentation for the command
func (p *createAPIPSubcommand) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a Kubernetes API in which the controller is an Ansible role or playbook.

    - generates a Custom Resource Definition and sample
    - Updates watches.yaml
    - optionally generates Ansible Role tree
    - optionally generates Ansible playbook

    For the scaffolded operator to be runnable with no changes, specify either --generate-role or --generate-playbook.

`
	ctx.Examples = fmt.Sprintf(`# Create a new API, without Ansible roles or playbooks
  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService

  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --generate-role

  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --generate-playbook

  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService
      --generate-playbook
      --generate-role
`,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
	)
}

func (p *createAPIPSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false

	fs.StringVar(&p.options.Group, "group", "", "resource group")
	fs.StringVar(&p.options.Version, "version", "", "resource version")
	fs.StringVar(&p.options.Kind, "kind", "", "resource kind")
	fs.StringVar(&p.options.CRDVersion, "crd-version", defaultCRDVersion, "crd version to generate")

	fs.BoolVarP(&p.doPlaybook, "generate-playbook", "", false, "Generate an Ansible playbook. If passed with --generate-role, the playbook will invoke the role.")
	fs.BoolVarP(&p.doRole, "generate-role", "", false, "Generate an Ansible role skeleton.")
}

func (p *createAPIPSubcommand) InjectConfig(c config.Config) {
	p.config = c
}

func (p *createAPIPSubcommand) Run() error {
	if err := cmdutil.Run(p); err != nil {
		return err
	}

	// Run SDK phase 2 plugins.
	if err := p.runPhase2(); err != nil {
		return err
	}

	return nil
}

// SDK phase 2 plugins.
func (p *createAPIPSubcommand) runPhase2() error {
	if p.resource == nil {
		return errors.New("resource must not be nil")
	}

	// Initially the ansible/v1 plugin was written to not create a "plugins" config entry
	// for any phase 2 plugin because they did not have their own keys. Now there are phase 2
	// plugin keys, so those plugins should be run if keys exist. Otherwise, enact old behavior.

	if manifestsv2.HasPluginConfig(p.config) {
		if err := manifestsv2.RunCreateAPI(p.config, p.resource.GVK); err != nil {
			return err
		}
	} else {
		if err := manifests.RunCreateAPI(p.config, p.resource.GVK); err != nil {
			return err
		}
	}

	return nil
}

func (p *createAPIPSubcommand) Validate() error {
	if len(strings.TrimSpace(p.options.Group)) == 0 {
		return errors.New("value of --group must not have empty value")
	}
	if len(strings.TrimSpace(p.options.Version)) == 0 {
		return errors.New("value of --version must not have empty value")
	}
	if len(strings.TrimSpace(p.options.Kind)) == 0 {
		return errors.New("value of --kind must not have empty value")
	}

	// Create and validate the resource from CreateOptions.
	p.resource = newResource(p.config, p.options)
	if err := p.resource.Validate(); err != nil {
		return err
	}

	// Check that resource doesn't exist
	if p.config.HasResource(p.resource.GVK) {
		return errors.New("the API resource already exists")
	}

	// Check that the provided group can be added to the project
	if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.GVK.Group) {
		return errors.New("multiple groups are not allowed by default, to enable multi-group set 'multigroup: true' in your PROJECT file")
	}

	// Selected CRD version must match existing CRD versions.
	if pluginutil.HasDifferentCRDVersion(p.config, p.resource.API.CRDVersion) {
		return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q", p.resource.API.CRDVersion)
	}

	return nil
}

func (p *createAPIPSubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	return scaffolds.NewCreateAPIScaffolder(p.config, p.resource, p.doRole, p.doPlaybook), nil
}

func (p *createAPIPSubcommand) PostScaffold() error {
	return nil
}

type createOptions = golang.Options

func newResource(cfg config.Config, opts createOptions) *resource.Resource {
	opts.DoAPI = true
	opts.Namespaced = true

	r := opts.NewResource(cfg)
	r.Domain = cfg.GetDomain()
	// Remove the path since this is not a Golang project.
	r.Path = ""
	return &r
}
