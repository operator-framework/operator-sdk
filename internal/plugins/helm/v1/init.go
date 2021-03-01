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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds"
)

const (
	groupFlag   = "group"
	versionFlag = "version"
	kindFlag    = "kind"
)

type initSubcommand struct {
	apiSubcommand createAPISubcommand

	config config.Config

	// For help text.
	commandName string

	// Flags
	group       string
	domain      string
	version     string
	kind        string
	projectName string
}

var _ plugin.InitSubcommand = &initSubcommand{}

// UpdateContext define plugin context
func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize a new Helm-based operator project.

Writes the following files:
- a helm-charts directory with the chart(s) to build releases from
- a watches.yaml file that defines the mapping between your API and a Helm chart
- a PROJECT file with the domain and project layout configuration
- a Makefile to build the project
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
`
	subcmdMeta.Examples = fmt.Sprintf(`  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --group=apps \
      --version=v1alpha1 \
      --kind=AppService

  $ %[1]s init --plugins=%[2]s \
      --project-name=myapp
      --domain=example.com \
      --group=apps \
      --version=v1alpha1 \
      --kind=AppService

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --group=apps \
      --version=v1alpha1 \
      --kind=AppService \
      --helm-chart=myrepo/app

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=myrepo/app

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=myrepo/app \
      --helm-chart-version=1.2.3

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=app \
      --helm-chart-repo=https://charts.mycompany.com/ \
      --helm-chart-version=1.2.3

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=/path/to/local/chart-directories/app/

  $ %[1]s init --plugins=%[2]s \
      --domain=example.com \
      --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
`, cliMeta.CommandName, pluginKey)

	p.commandName = cliMeta.CommandName
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	fs.SortFlags = false
	fs.StringVar(&p.domain, "domain", "my.domain", "domain for groups")
	fs.StringVar(&p.projectName, "project-name", "", "name of this project, the default being directory name")

	fs.StringVar(&p.group, groupFlag, "", "resource Group")
	fs.StringVar(&p.version, versionFlag, "", "resource Version")
	fs.StringVar(&p.kind, kindFlag, "", "resource Kind")
	p.apiSubcommand.BindFlags(fs)
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	if err := p.config.SetDomain(p.domain); err != nil {
		return err
	}

	// Assign a default project name
	if p.projectName == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %v", err)
		}
		p.projectName = strings.ToLower(filepath.Base(dir))
	}
	// Check if the project name is a valid k8s namespace (DNS 1123 label).
	if err := validation.IsDNS1123Label(p.projectName); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", p.projectName, err)
	}
	if err := p.config.SetProjectName(p.projectName); err != nil {
		return err
	}

	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	scaffolder := scaffolds.NewInitScaffolder(p.config)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}

// PostScaffold will run the required actions after the default plugin scaffold
func (p *initSubcommand) PostScaffold() error {
	doAPI := p.group != "" || p.version != "" || p.kind != "" || p.apiSubcommand.options.chartOptions.Chart != defaultHelmChart
	if !doAPI {
		fmt.Printf("Next: define a resource with:\n$ %s create api\n", p.commandName)
	} else {
		args := []string{"create", "api"}
		// The following three checks should match the default values in sig.k8s.io/kubebuilder/v3/pkg/cli/resource.go
		if p.group != "" {
			args = append(args, fmt.Sprintf("--%s", groupFlag), p.group)
		}
		if p.version != "" {
			args = append(args, fmt.Sprintf("--%s", versionFlag), p.version)
		}
		if p.kind != "" {
			args = append(args, fmt.Sprintf("--%s", kindFlag), p.kind)
		}
		if p.apiSubcommand.options.CRDVersion != defaultCrdVersion {
			args = append(args, fmt.Sprintf("--%s", crdVersionFlag), p.apiSubcommand.options.CRDVersion)
		}
		if p.apiSubcommand.options.chartOptions.Chart != defaultHelmChart {
			args = append(args, fmt.Sprintf("--%s", helmChartFlag), p.apiSubcommand.options.chartOptions.Chart)
		}
		if p.apiSubcommand.options.chartOptions.Repo != defaultHelmChartRepo {
			args = append(args, fmt.Sprintf("--%s", helmChartRepoFlag), p.apiSubcommand.options.chartOptions.Repo)
		}
		if p.apiSubcommand.options.chartOptions.Version != defaultHelmChartVersion {
			args = append(args, fmt.Sprintf("--%s", helmChartVersionFlag), p.apiSubcommand.options.chartOptions.Version)
		}
		if err := util.RunCmd("Creating the API", os.Args[0], args...); err != nil {
			return err
		}
	}

	return nil
}
