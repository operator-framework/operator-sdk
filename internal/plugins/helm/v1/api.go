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

	"github.com/iancoleman/strcase"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds"
	"github.com/operator-framework/operator-sdk/internal/plugins/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

const (
	crdVersionFlag       = "crd-version"
	helmChartFlag        = "helm-chart"
	helmChartRepoFlag    = "helm-chart-repo"
	helmChartVersionFlag = "helm-chart-version"

	defaultCrdVersion = "v1"
	legacyCrdVersion  = "v1beta1"

	// defaultGroup is the Kubernetes CRD API Group used for fetched charts when the --group flag is not specified
	defaultGroup = "charts"
	// defaultVersion is the Kubernetes CRD API Version used for fetched charts when the --version flag is not specified
	defaultVersion = "v1alpha1"
)

type createAPIOptions struct {
	// CRDVersion is the version of the `apiextensions.k8s.io` API which will be used to generate the CRD.
	CRDVersion string

	chartOptions chartutil.Options
}

// UpdateResource updates the base resource with the information obtained from the flags
func (opts createAPIOptions) UpdateResource(res *resource.Resource) {
	res.API = &resource.API{
		CRDVersion: opts.CRDVersion,
		Namespaced: true,
	}

	// Ensure that Path is empty and Controller false
	res.Path = ""
	res.Controller = false
}

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config   config.Config
	resource *resource.Resource
	chart    *chart.Chart
	options  createAPIOptions
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Scaffold a Kubernetes API that is backed by a Helm chart.
`
	subcmdMeta.Examples = fmt.Sprintf(`  $ %s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService

  $ %[1]s create api \
      --group=apps --version=v1alpha1 \
      --kind=AppService \
      --helm-chart=myrepo/app

  $ %[1]s create api \
      --helm-chart=myrepo/app

  $ %[1]s create api \
      --helm-chart=myrepo/app \
      --helm-chart-version=1.2.3

  $ %[1]s create api \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/

  $ %[1]s create api \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/ \
      --helm-chart-version=1.2.3

  $ %[1]s create api \
      --helm-chart=/path/to/local/chart-directories/app/

  $ %[1]s create api \
      --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz

  $ %[1]s create api \
      --helm-chart=oci://charts.mycompany.com/example-namespace/app:1.2.3
`, cliMeta.CommandName)
}

// BindFlags will set the flags for the plugin
func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false

	fs.StringVar(&p.options.chartOptions.Chart, helmChartFlag, "", "helm chart")
	fs.StringVar(&p.options.chartOptions.Repo, helmChartRepoFlag, "", "helm chart repository")
	fs.StringVar(&p.options.chartOptions.Version, helmChartVersionFlag, "", "helm chart version (default: latest)")

	fs.StringVar(&p.options.CRDVersion, crdVersionFlag, defaultCrdVersion, "crd version to generate")
	// (not required raise an error in this case)
	// nolint:errcheck,gosec
	fs.MarkDeprecated(crdVersionFlag, util.WarnMessageRemovalV1beta1)
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	if p.options.CRDVersion == legacyCrdVersion {
		logrus.Warn(util.WarnMessageRemovalV1beta1)
	}
	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// The following checks and the chart creation would be a better fit for PreScaffold method
	// but, as having a chart sets some default values for the resource's GVK, we need to do it here.
	var err error
	if len(strings.TrimSpace(p.options.chartOptions.Chart)) == 0 {
		// Chart repo and version can only be provided if chart was provided.
		if len(strings.TrimSpace(p.options.chartOptions.Repo)) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartRepoFlag, helmChartFlag)
		}
		if len(p.options.chartOptions.Version) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartVersionFlag, helmChartFlag)
		}

		// Kind is required if no chart was provided as it is used for the chart name.
		// While the resource validation will detect this, the error yielded would not
		// mention the option of providing the chart flag. Additionally, by checking it
		// here we can create the new chart before resource validation.
		if len(p.resource.Kind) == 0 {
			return fmt.Errorf("either --%s or --%s need to be provided", kindFlag, helmChartFlag)
		}

		p.chart, err = chartutil.NewChart(strings.ToLower(p.resource.Kind))
		if err != nil {
			return err
		}
	} else {
		p.chart, err = chartutil.LoadChart(p.options.chartOptions)
		if err != nil {
			return err
		}

		// In case we loaded a chart and some resource flags were not set we will set defaults.
		if p.resource.Group == "" {
			p.resource.Group = defaultGroup
		}
		if p.resource.Version == "" {
			p.resource.Version = defaultVersion
		}
		if p.resource.Kind == "" {
			p.resource.Kind = strcase.ToCamel(p.chart.Name())
			if p.resource.Plural == "" {
				p.resource.Plural = resource.RegularPlural(p.resource.Kind)
			}
		}
	}

	p.options.UpdateResource(p.resource)

	if err := p.resource.Validate(); err != nil {
		return err
	}

	// Check that resource doesn't have the API scaffolded
	if res, err := p.config.GetResource(p.resource.GVK); err == nil && res.HasAPI() {
		return errors.New("the API resource already exists")
	}

	// Check that the provided group can be added to the project
	// nolint:staticcheck
	if !p.config.IsMultiGroup() && p.config.ResourcesLength() != 0 && !p.config.HasGroup(p.resource.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group set 'multigroup: true' in your PROJECT file")
	}

	// Selected CRD version must match existing CRD versions.
	// nolint:staticcheck
	if hasDifferentAPIVersion(p.config.ListCRDVersions(), p.resource.API.CRDVersion) {
		return fmt.Errorf("only one CRD version can be used for all resources, cannot add %q", p.resource.API.CRDVersion)
	}

	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := util.RemoveKustomizeCRDManifests(); err != nil {
		return fmt.Errorf("error removing kustomization CRD manifests: %v", err)
	}
	if err := util.UpdateKustomizationsCreateAPI(); err != nil {
		return fmt.Errorf("error updating kustomization.yaml files: %v", err)
	}

	scaffolder := scaffolds.NewAPIScaffolder(p.config, *p.resource, p.chart)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return err
	}
	// NOTE: previous step fetches the dependencies of the chart.Chart, so reloading may be needed if used afterwards

	return nil
}

// hasDifferentCRDVersion returns true if any other CRD version is tracked in the project configuration.
func hasDifferentAPIVersion(versions []string, version string) bool {
	return !(len(versions) == 0 || (len(versions) == 1 && versions[0] == version))
}
