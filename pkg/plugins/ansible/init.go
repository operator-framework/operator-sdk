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

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

type initPlugin struct {
	config *config.Config

	commandName string
}

var _ plugin.Init = &initPlugin{}

func (p *initPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Initialize a new Ansible Operator project.

Writes the following files:
- a PROJECT file with the domain and repo
- a Makefile to build the project
`
	ctx.Examples = fmt.Sprintf(`  # Scaffold a project with a domain and project name
  %s init --plugins ansible --domain example.org --repo my-operator
`,
		ctx.CommandName)

	p.commandName = ctx.CommandName
}

func (p *initPlugin) BindFlags(fs *pflag.FlagSet) {
	// project args
	fs.StringVar(&p.config.Repo, "repo", "", "name of the project, "+
		"defaults to the name of the current working directory.")
	fs.StringVar(&p.config.Domain, "domain", "my.domain", "domain for groups")
}

func (p *initPlugin) InjectConfig(c *config.Config) {
	c.Layout = plugin.KeyFor(Plugin{})
	p.config = c
}

func (p *initPlugin) Run() error {
	if err := p.Validate(); err != nil {
		return err
	}

	if err := p.Scaffold(); err != nil {
		return err
	}

	return p.PostScaffold()
}

func (p *initPlugin) Validate() error {
	projectName := p.config.Repo
	if projectName == "" {
		// Check if the project name is a valid namespace according to k8s
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error to get the current path: %v", err)
		}
		projectName = filepath.Base(wd)
	}
	if err := validation.IsDNS1123Label(strings.ToLower(projectName)); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", projectName, err)
	}

	if err := validation.IsDNS1123Subdomain(p.config.Domain); err != nil {
		return fmt.Errorf("domain (%s) is invalid: %v", p.config.Domain, err)
	}
	return nil
}

func (p initPlugin) Scaffold() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if p.config.Repo == "" {
		p.config.Repo = filepath.Base(wd)
	}

	cfg := &input.Config{
		AbsProjectPath: wd,
		ProjectName:    p.config.Repo,
	}

	err = (&scaffold.Scaffold{}).Execute(cfg,
		&ansible.RequirementsYml{},
		&ansible.Makefile{},
	)
	if err != nil {
		return fmt.Errorf("init failed: %v", err)
	}

	return nil
}

func (p *initPlugin) PostScaffold() error {
	fmt.Printf("\nNext: define a resource with:\n$ %s create api\n", p.commandName)
	return nil
}
