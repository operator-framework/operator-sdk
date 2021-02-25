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
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests"
	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
)

type createAPISubcommand struct {
	config  config.Config
	options chartutil.CreateOptions

	resource *resource.Resource
	chrt     *chart.Chart
}

var (
	_ plugin.CreateAPISubcommand = &createAPISubcommand{}
	_ cmdutil.RunOptions         = &createAPISubcommand{}
)

// UpdateContext define plugin context
func (p createAPISubcommand) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Scaffold a Kubernetes API that is backed by a Helm chart.
`
	ctx.Examples = fmt.Sprintf(`  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService

  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --helm-chart=myrepo/app

  $ %s create api \
      --helm-chart=myrepo/app

  $ %s create api \
      --helm-chart=myrepo/app \
      --helm-chart-version=1.2.3

  $ %s create api \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/

  $ %s create api \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/ \
      --helm-chart-version=1.2.3

  $ %s create api \
      --helm-chart=/path/to/local/chart-directories/app/

  $ %s create api \
      --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
		ctx.CommandName,
	)
}

const (
	helmChartFlag        = "helm-chart"
	helmChartRepoFlag    = "helm-chart-repo"
	helmChartVersionFlag = "helm-chart-version"

	defaultCRDVersion = "v1"
)

// BindFlags will set the flags for the plugin
func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false

	fs.StringVar(&p.options.GVK.Group, "group", "", "resource group")
	fs.StringVar(&p.options.GVK.Version, "version", "", "resource version")
	fs.StringVar(&p.options.GVK.Kind, "kind", "", "resource kind")

	fs.StringVar(&p.options.Chart, helmChartFlag, "", "helm chart")
	fs.StringVar(&p.options.Repo, helmChartRepoFlag, "", "helm chart repository")
	fs.StringVar(&p.options.Version, helmChartVersionFlag, "", "helm chart version (default: latest)")

	fs.StringVar(&p.options.CRDVersion, "crd-version", defaultCRDVersion, "crd version to generate")
}

// InjectConfig will inject the PROJECT file/config in the plugin
func (p *createAPISubcommand) InjectConfig(c config.Config) {
	p.config = c
}

// Run will call the plugin actions according to the definitions done in RunOptions interface
func (p *createAPISubcommand) Run() error {
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
func (p *createAPISubcommand) runPhase2() error {
	// Initially the helm/v1 plugin was written to not create a "plugins" config entry
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

// Validate perform the required validations for this plugin
func (p *createAPISubcommand) Validate() (err error) {
	if len(strings.TrimSpace(p.options.Chart)) == 0 {
		if len(strings.TrimSpace(p.options.Repo)) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartRepoFlag, helmChartFlag)
		} else if len(p.options.Version) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartVersionFlag, helmChartFlag)
		}
	}

	if len(strings.TrimSpace(p.options.Chart)) == 0 {
		if len(strings.TrimSpace(p.options.GVK.Group)) == 0 {
			return errors.New("value of --group must not have empty value")
		}
		if len(strings.TrimSpace(p.options.GVK.Version)) == 0 {
			return errors.New("value of --version must not have empty value")
		}
		if len(strings.TrimSpace(p.options.GVK.Kind)) == 0 {
			return errors.New("value of --kind must not have empty value")
		}
	}

	// Create and validate the resource and chart from CreateOptions.
	p.resource, p.chrt, err = chartutil.CreateChart(p.config, p.options)
	if err != nil {
		return err
	}
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

	// Check CRDVersion against all other CRDVersions in p.config for compatibility.
	if !p.config.IsCRDVersionCompatible(p.resource.API.CRDVersion) {
		return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q", p.resource.API.CRDVersion)
	}

	return nil
}

// GetScaffolder returns cmdutil.Scaffolder which will be executed due the RunOptions interface implementation
func (p *createAPISubcommand) GetScaffolder() (cmdutil.Scaffolder, error) {
	return scaffolds.NewAPIScaffolder(p.config, p.resource, p.chrt), nil
}

// PostScaffold runs all actions that should be executed after the default plugin scaffold
func (p *createAPISubcommand) PostScaffold() error {
	return nil
}
