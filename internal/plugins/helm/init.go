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
package helm

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"

	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"
	cmdutil "github.com/operator-framework/operator-sdk/internal/plugins/helm/internal"
	sdkscaffold "github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

type initPlugin struct {
	config *config.Config

	// For help text.
	commandName string

	// Helm flags
	Domain           string
	Group            string
	Version          string
	Kind             string
	CRDVersion       string
	HelmChartRef     string
	HelmChartVersion string
	HelmChartRepo    string
}

var (
	_ plugin.Init        = &initPlugin{}
	_ cmdutil.RunOptions = &initPlugin{}
)

func (p *initPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Initialize a Helm project with the Helm Chart package directories.`
	ctx.Examples = fmt.Sprintf(`  # Scaffold a Helm project 
	
  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --domain=app.example \	
  --group=app \
  --version=v1alpha1 \	
  --kind=AppService

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --domain=app.example \	
  --group=app \
  --version=v1alpha1 \	
  --kind=AppService \
  --helm-chart=myrepo/app

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --helm-chart=myrepo/app

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --helm-chart=myrepo/app \
  --helm-chart-version=1.2.3

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/ \
  --helm-chart-version=1.2.3

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --helm-chart=/path/to/local/chart-directories/app/

  $ operator-sdk init --plugins=helm.kubebuilder.io/v1-alpha \
  --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`)

	p.commandName = ctx.CommandName
}

func (p *initPlugin) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.Domain, "domain", "", "Kubernetes domain for groups. (e.g example.com)")
	fs.StringVar(&p.Group, "group", "", "Kubernetes resource Kind Group. (e.g app)")
	fs.StringVar(&p.Version, "version", "", "Kubernetes resource Version. (e.g v1alpha1)")
	fs.StringVar(&p.Kind, "kind", "",
		"Kubernetes resource Kind name. (e.g AppService)")
	fs.StringVar(&p.CRDVersion, "crd-version", gencrd.DefaultCRDVersion,
		"CRD version to generate")
	fs.StringVar(&p.HelmChartRef, "helm-chart", "",
		"Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path).")
	fs.StringVar(&p.HelmChartVersion, "helm-chart-version", "",
		"Specific version of the helm chart (default is latest version)")
	fs.StringVar(&p.HelmChartRepo, "helm-chart-repo", "",
		"Chart repository URL for the requested helm chart")
}

func (p *initPlugin) InjectConfig(c *config.Config) {
	c.Layout = plugin.KeyFor(Plugin{})
	p.config = c
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

	if len(p.HelmChartRef) == 0 {
		if len(p.HelmChartRepo) != 0 {
			return fmt.Errorf("value of --helm-chart-repo can only be used with --helm-chart")
		} else if len(p.HelmChartVersion) != 0 {
			return fmt.Errorf("value of --helm-chart-version can only be used with --helm-chart")
		}
	}

	if len(p.HelmChartRef) == 0 {
		if len(p.Domain) == 0 {
			return fmt.Errorf("value of --domain must not have empty value")
		}
		if len(p.Group) == 0 {
			return fmt.Errorf("value of --group must not have empty value")
		}
		if len(p.Version) == 0 {
			return fmt.Errorf("value of --version must not have empty value")
		}
		if len(p.Kind) == 0 {
			return fmt.Errorf("value of --kind must not have empty value")
		}
		// To validate the resource
		_, err := sdkscaffold.NewResource(path.Join(p.Group+"."+p.Domain, p.Version), p.Kind)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *initPlugin) GetScaffolder() (scaffold.Scaffolder, error) {
	// Check if the project name is a valid namespace according to k8s
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error to get the current path: %v", err)
	}

	cfg := input.Config{
		AbsProjectPath: filepath.Join(projutil.MustGetwd()),
		ProjectName:    filepath.Base(dir),
	}

	createOpts := helm.CreateChartOptions{
		ResourceAPIVersion: path.Join(p.Group+"."+p.Domain, p.Version),
		ResourceKind:       p.Kind,
		Chart:              p.HelmChartRef,
		Version:            p.HelmChartVersion,
		Repo:               p.HelmChartRepo,
		CRDVersion:         p.CRDVersion,
	}

	if err := helm.Init(cfg, createOpts); err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *initPlugin) PostScaffold() error {
	return nil
}
