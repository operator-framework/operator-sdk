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

package v1

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
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/scorecard"
	utilplugins "github.com/operator-framework/operator-sdk/internal/util/plugins"
)

type initPlugin struct {
	config        *config.Config
	apiPlugin     createAPIPlugin
	doAPIScaffold bool

	// For help text.
	commandName string
}

var (
	_ plugin.Init        = &initPlugin{}
	_ cmdutil.RunOptions = &initPlugin{}
)

// UpdateContext define plugin context
func (p *initPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Initialize a new Helm-based operator project.

Writes the following files:
- a helm-charts directory with the chart(s) to build releases from
- a watches.yaml file that defines the mapping between your API and a Helm chart
- a PROJECT file with the domain and project layout configuration
- a Makefile to build the project
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
`
	ctx.Examples = fmt.Sprintf(`  $ %s init --plugins=%s \
      --domain=example.com \
      --group=apps --version=v1alpha1 \
      --kind=AppService

  $ %s init --plugins=%s \
      --domain=example.com \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --helm-chart=myrepo/app

  $ %s init --plugins=%s \
      --domain=example.com \
      --helm-chart=myrepo/app

  $ %s init --plugins=%s \
      --domain=example.com \
      --helm-chart=myrepo/app \
      --helm-chart-version=1.2.3

  $ %s init --plugins=%s \
      --domain=example.com \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/

  $ %s init --plugins=%s \
      --domain=example.com \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/ \
      --helm-chart-version=1.2.3

  $ %s init --plugins=%s \
      --domain=example.com \
      --helm-chart=/path/to/local/chart-directories/app/

  $ %s init --plugins=%s \
      --domain=example.com \
      --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`,
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
		ctx.CommandName, plugin.KeyFor(Plugin{}),
	)

	p.commandName = ctx.CommandName
}

// BindFlags will set the flags for the plugin
func (p *initPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false
	fs.StringVar(&p.config.Domain, "domain", "my.domain", "domain for groups")
	p.apiPlugin.BindFlags(fs)
}

// InjectConfig will inject the PROJECT file/config in the plugin
func (p *initPlugin) InjectConfig(c *config.Config) {
	// v3 project configs get a 'layout' value.
	c.Layout = plugin.KeyFor(Plugin{})
	p.config = c
	p.apiPlugin.config = p.config
}

// Run will call the plugin actions
func (p *initPlugin) Run() error {
	if err := cmdutil.Run(p); err != nil {
		return err
	}

	// Run the scorecard "phase 2" plugin.
	if err := scorecard.RunInit(p.config); err != nil {
		return err
	}

	return nil
}

// Validate perform the required validations for this plugin
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

	defaultOpts := chartutil.CreateOptions{CRDVersion: "v1"}
	if !p.apiPlugin.createOptions.GVK.Empty() || p.apiPlugin.createOptions != defaultOpts {
		p.doAPIScaffold = true
		return p.apiPlugin.Validate()
	}

	return nil
}

// GetScaffolder returns scaffold.Scaffolder which will be executed due the RunOptions interface implementation
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

// PostScaffold will run the required actions after the default plugin scaffold
func (p *initPlugin) PostScaffold() error {
	// runs the SDK customizations (wrappers)
	if err := utilplugins.UpdateMakefile(p.config); err != nil {
		return err
	}

	if !p.doAPIScaffold {
		fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	} else {
		return p.apiPlugin.PostScaffold()
	}
	return nil
}
