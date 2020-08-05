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
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests"
)

const (
	groupFlag      = "group"
	versionFlag    = "version"
	kindFlag       = "kind"
	crdVersionFlag = "crd-version"

	crdVersionV1      = "v1"
	crdVersionV1beta1 = "v1beta1"
)

type createAPIPlugin struct {
	config        *config.Config
	createOptions scaffolds.CreateOptions
}

var (
	_ plugin.CreateAPI   = &createAPIPlugin{}
	_ cmdutil.RunOptions = &createAPIPlugin{}
)

// UpdateContext injects documentation for the command
func (p *createAPIPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a Kubernetes API in which the controller is an Ansible role or playbook.

    - generates a Custom Resource Definition and sample
    - Updates watches.yaml
    - optionally generates Ansible Role tree
    - optionally generates Ansible playbook

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

func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false

	fs.StringVar(&p.createOptions.GVK.Group, groupFlag, "", "resource group")
	fs.StringVar(&p.createOptions.GVK.Version, versionFlag, "", "resource version")
	fs.StringVar(&p.createOptions.GVK.Kind, kindFlag, "", "resource kind")
	fs.StringVar(&p.createOptions.CRDVersion, crdVersionFlag, crdVersionV1, "crd version to generate")
	fs.BoolVarP(&p.createOptions.GeneratePlaybook, "generate-playbook", "", false, "Generate an Ansible playbook. If passed with --generate-role, the playbook will invoke the role.")
	fs.BoolVarP(&p.createOptions.GenerateRole, "generate-role", "", false, "Generate an Ansible role skeleton.")
}

func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.config = c
}

func (p *createAPIPlugin) Run() error {
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
func (p *createAPIPlugin) runPhase2() error {
	gvk := p.createOptions.GVK
	return manifests.RunCreateAPI(p.config, config.GVK{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind})
}

func (p *createAPIPlugin) Validate() error {
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

func (p *createAPIPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	return scaffolds.NewCreateAPIScaffolder(p.config, p.createOptions), nil
}

func (p *createAPIPlugin) PostScaffold() error {
	return nil
}
