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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds"
	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
	scorecardv2 "github.com/operator-framework/operator-sdk/internal/plugins/scorecard/v2"
)

type initSubcommand struct {
	config    *config.Config
	apiPlugin createAPIPSubcommand

	// If true, run the `create api` plugin.
	doCreateAPI bool

	// For help text.
	commandName string
}

var (
	_ plugin.InitSubcommand = &initSubcommand{}
	_ cmdutil.RunOptions    = &initSubcommand{}
)

// UpdateContext injects documentation for the command
func (p *initSubcommand) UpdateContext(ctx *plugin.Context) {
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
  $ %[1]s init --plugins=%[2]s --domain=my.domain \

  # Invokes "create api"
  $ %[1]s init --plugins=%[2]s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService

  $ %[1]s init --plugins=%[2]s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService \
      --generate-role

  $ %[1]s init --plugins=%[2]s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService \
      --generate-playbook

  $ %[1]s init --plugins=%[2]s \
      --domain=my.domain \
      --group=apps --version=v1alpha1 --kind=AppService \
      --generate-playbook \
      --generate-role
`,
		ctx.CommandName, pluginKey,
	)
	p.commandName = ctx.CommandName
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false
	fs.StringVar(&p.config.Domain, "domain", "my.domain", "domain for groups")
	fs.StringVar(&p.config.ProjectName, "project-name", "", "name of this project, the default being directory name")
	p.apiPlugin.BindFlags(fs)
}

func (p *initSubcommand) InjectConfig(c *config.Config) {
	c.Layout = pluginKey
	p.config = c
	p.apiPlugin.config = p.config
}

func (p *initSubcommand) Run() error {
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
func (p *initSubcommand) runPhase2() error {
	if err := manifestsv2.RunInit(p.config); err != nil {
		return err
	}
	if err := scorecardv2.RunInit(p.config); err != nil {
		return err
	}

	if p.doCreateAPI {
		if err := p.apiPlugin.runPhase2(); err != nil {
			return err
		}
	}

	return nil
}

func (p *initSubcommand) Validate() error {
	// Check if the project name is a valid k8s namespace (DNS 1123 label).
	if p.config.ProjectName == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %v", err)
		}
		p.config.ProjectName = strings.ToLower(filepath.Base(dir))
	}
	if err := validation.IsDNS1123Label(p.config.ProjectName); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", p.config.ProjectName, err)
	}

	defaultOpts := scaffolds.CreateOptions{CRDVersion: "v1"}
	if !p.apiPlugin.createOptions.GVK.Empty() || p.apiPlugin.createOptions != defaultOpts {
		p.doCreateAPI = true
		return p.apiPlugin.Validate()
	}
	return nil
}

func (p *initSubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	var (
		apiScaffolder cmdutil.Scaffolder
		err           error
	)
	if p.doCreateAPI {
		apiScaffolder, err = p.apiPlugin.GetScaffolder()
		if err != nil {
			return nil, err
		}
	}
	return scaffolds.NewInitScaffolder(p.config, apiScaffolder), nil
}

func (p *initSubcommand) PostScaffold() error {
	if !p.doCreateAPI {
		fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	}

	return nil
}
