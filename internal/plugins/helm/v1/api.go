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
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests"
)

type createAPIPlugin struct {
	config *config.Config

	createOptions chartutil.CreateOptions
}

var (
	_ plugin.CreateAPI   = &createAPIPlugin{}
	_ cmdutil.RunOptions = &createAPIPlugin{}
)

// UpdateContext define plugin context
func (p createAPIPlugin) UpdateContext(ctx *plugin.Context) {
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
	groupFlag            = "group"
	versionFlag          = "version"
	kindFlag             = "kind"
	helmChartFlag        = "helm-chart"
	helmChartRepoFlag    = "helm-chart-repo"
	helmChartVersionFlag = "helm-chart-version"
	crdVersionFlag       = "crd-version"

	crdVersionV1      = "v1"
	crdVersionV1beta1 = "v1beta1"
)

// BindFlags will set the flags for the plugin
func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet) {
	p.createOptions = chartutil.CreateOptions{}
	fs.SortFlags = false

	fs.StringVar(&p.createOptions.GVK.Group, groupFlag, "", "resource group")
	fs.StringVar(&p.createOptions.GVK.Version, versionFlag, "", "resource version")
	fs.StringVar(&p.createOptions.GVK.Kind, kindFlag, "", "resource kind")

	fs.StringVar(&p.createOptions.Chart, helmChartFlag, "", "helm chart")
	fs.StringVar(&p.createOptions.Repo, helmChartRepoFlag, "", "helm chart repository")
	fs.StringVar(&p.createOptions.Version, helmChartVersionFlag, "", "helm chart version (default: latest)")

	fs.StringVar(&p.createOptions.CRDVersion, crdVersionFlag, crdVersionV1, "crd version to generate")
}

// InjectConfig will inject the PROJECT file/config in the plugin
func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.config = c
}

// Run will call the plugin actions according to the definitions done in RunOptions interface
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

// Validate perform the required validations for this plugin
func (p *createAPIPlugin) Validate() error {
	if p.createOptions.CRDVersion != crdVersionV1 && p.createOptions.CRDVersion != crdVersionV1beta1 {
		return fmt.Errorf("value of --%s must be either %q or %q", crdVersionFlag, crdVersionV1, crdVersionV1beta1)
	}

	if len(strings.TrimSpace(p.createOptions.Chart)) == 0 {
		if len(strings.TrimSpace(p.createOptions.Repo)) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartRepoFlag, helmChartFlag)
		} else if len(p.createOptions.Version) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartVersionFlag, helmChartFlag)
		}
	}

	if len(strings.TrimSpace(p.createOptions.Chart)) == 0 {
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
	}

	return nil
}

// GetScaffolder returns scaffold.Scaffolder which will be executed due the RunOptions interface implementation
func (p *createAPIPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	return scaffolds.NewAPIScaffolder(p.config, p.createOptions), nil
}

// PostScaffold runs all actions that should be executed after the default plugin scaffold
func (p *createAPIPlugin) PostScaffold() error {
	return nil
}
