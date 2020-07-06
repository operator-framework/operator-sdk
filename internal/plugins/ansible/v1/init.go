/*
Copyright 2020 The Kubernetes Authors.
Modifications copyright 2020 The Operator-SDK Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ansible

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds"
)

type initPlugin struct {
	config    *config.Config
	apiPlugin createAPIPlugin

	doAPIScaffold bool

	// For help text.
	commandName string
}

var (
	_ plugin.Init        = &initPlugin{}
	_ cmdutil.RunOptions = &initPlugin{}
)

// UpdateContext injects documentation for the command
func (p *initPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `
Initialize a new Ansible-based operator project.

Writes the following files
- a kubebuilder PROJECT file with the domain and project layout configuration
- a Makefile that provides an interface for building and managing the operator
- Kubernetes manifests and kustomize configuration
- a watches.yaml file that defines the mapping between APIs and Roles/Playbooks

Optionally creates a new API, using the same flags as "create api"
`
	ctx.Examples = fmt.Sprintf(`
  # Scaffold a project with no API
  $ %s init --plugins=%s --domain=my.domain \

  # Invokes "create api"
  $ %s init --plugins=%s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService

  $ %s init --plugins=%s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService \
      --generate-role

  $ %s init --plugins=%s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService \
      --generate-playbook

  $ %s init --plugins=%s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService \
      --generate-playbook \
      --generate-role
`,
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
	)
	p.commandName = ctx.CommandName
}

func (p *initPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false
	fs.StringVar(&p.config.Domain, "domain", "my.domain", "domain for groups")
	p.apiPlugin.BindFlags(fs)
}

func (p *initPlugin) InjectConfig(c *config.Config) {
	c.Layout = plugin.KeyFor(Plugin{})
	p.config = c
	p.apiPlugin.config = p.config
}

func (p *initPlugin) Run() error {
	return cmdutil.Run(p)
}

func (p *initPlugin) Validate() error {
	// Check if the project name is a valid namespace according to k8s
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error to get the current path: %v", err)
	}
	projectName := filepath.Base(dir)
	if err := validation.IsDNS1123Label(strings.ToLower(projectName)); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", projectName, err)
	}

	defaultOpts := scaffolds.CreateOptions{CRDVersion: "v1"}
	if !p.apiPlugin.createOptions.GVK.Empty() || p.apiPlugin.createOptions != defaultOpts {
		p.doAPIScaffold = true
		return p.apiPlugin.Validate()
	}
	return nil
}

func (p *initPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	var (
		apiScaffolder scaffold.Scaffolder
		err           error
	)
	if p.doAPIScaffold {
		apiScaffolder, err = p.apiPlugin.GetScaffolder()
		if err != nil {
			return nil, err
		}
	}
	return scaffolds.NewInitScaffolder(p.config, apiScaffolder), nil
}

func (p *initPlugin) PostScaffold() error {
	if !p.doAPIScaffold {
		fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	}

	return nil
}
