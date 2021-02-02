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
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests"
	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
)

const (
	groupFlag      = "group"
	versionFlag    = "version"
	kindFlag       = "kind"
	crdVersionFlag = "crd-version"

	crdVersionV1      = "v1"
	crdVersionV1beta1 = "v1beta1"
)

type createAPIPSubcommand struct {
	config        *config.Config
	createOptions scaffolds.CreateOptions
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

	fs.StringVar(&p.createOptions.GVK.Group, groupFlag, "", "resource group")
	fs.StringVar(&p.createOptions.GVK.Version, versionFlag, "", "resource version")
	fs.StringVar(&p.createOptions.GVK.Kind, kindFlag, "", "resource kind")
	fs.StringVar(&p.createOptions.CRDVersion, crdVersionFlag, crdVersionV1, "crd version to generate")
	fs.BoolVarP(&p.createOptions.GeneratePlaybook, "generate-playbook", "", false, "Generate an Ansible playbook. If passed with --generate-role, the playbook will invoke the role.")
	fs.BoolVarP(&p.createOptions.GenerateRole, "generate-role", "", false, "Generate an Ansible role skeleton.")
}

func (p *createAPIPSubcommand) InjectConfig(c *config.Config) {
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
	ogvk := p.createOptions.GVK
	gvk := config.GVK{Group: ogvk.Group, Version: ogvk.Version, Kind: ogvk.Kind}

	// Initially the ansible/v1 plugin was written to not create a "plugins" config entry
	// for any phase 2 plugin because they did not have their own keys. Now there are phase 2
	// plugin keys, so those plugins should be run if keys exist. Otherwise, enact old behavior.

	if manifestsv2.HasPluginConfig(p.config) {
		if err := manifestsv2.RunCreateAPI(p.config, gvk); err != nil {
			return err
		}
	} else {
		if err := manifests.RunCreateAPI(p.config, gvk); err != nil {
			return err
		}
	}

	return nil
}

func (p *createAPIPSubcommand) Validate() error {
	if p.createOptions.CRDVersion != crdVersionV1 && p.createOptions.CRDVersion != crdVersionV1beta1 {
		return fmt.Errorf("value of --%s must be either %q or %q", crdVersionFlag, crdVersionV1, crdVersionV1beta1)
	}

	if len(strings.TrimSpace(p.createOptions.GVK.Group)) == 0 {
		return fmt.Errorf("value of --%s must not have empty value", groupFlag)
	}
	if len(strings.TrimSpace(p.createOptions.GVK.Version)) == 0 {
		return fmt.Errorf("value of --%s must not have empty value", versionFlag)
	}
	if len(strings.TrimSpace(p.createOptions.GVK.Kind)) == 0 {
		return fmt.Errorf("value of --%s must not have empty value", kindFlag)
	}

	// Validate the resource.
	r := resource.Options{
		Namespaced: true,
		Group:      p.createOptions.GVK.Group,
		Version:    p.createOptions.GVK.Version,
		Kind:       p.createOptions.GVK.Kind,
	}
	if err := r.Validate(); err != nil {
		return err
	}

	return nil
}

func (p *createAPIPSubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	return scaffolds.NewCreateAPIScaffolder(p.config, p.createOptions), nil
}

func (p *createAPIPSubcommand) PostScaffold() error {
	return nil
}
