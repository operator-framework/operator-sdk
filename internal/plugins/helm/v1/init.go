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

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds"
	sdkpluginutil "github.com/operator-framework/operator-sdk/internal/plugins/util"
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
	group   string
	version string
	kind    string
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
	fs.StringVar(&p.group, groupFlag, "", "resource Group")
	fs.StringVar(&p.version, versionFlag, "", "resource Version")
	fs.StringVar(&p.kind, kindFlag, "", "resource Kind")
	p.apiSubcommand.BindFlags(fs)
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {

	if err := addInitCustomizations(p.config.GetProjectName(), p.config.IsComponentConfig()); err != nil {
		return fmt.Errorf("error updating init manifests: %s", err)
	}

	scaffolder := scaffolds.NewInitScaffolder(p.config)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}

// PostScaffold will run the required actions after the default plugin scaffold
func (p *initSubcommand) PostScaffold() error {
	doAPI := p.group != "" || p.version != "" || p.kind != "" || p.apiSubcommand.options.chartOptions.Chart != ""
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
		if p.apiSubcommand.options.chartOptions.Chart != "" {
			args = append(args, fmt.Sprintf("--%s", helmChartFlag), p.apiSubcommand.options.chartOptions.Chart)
		}
		if p.apiSubcommand.options.chartOptions.Repo != "" {
			args = append(args, fmt.Sprintf("--%s", helmChartRepoFlag), p.apiSubcommand.options.chartOptions.Repo)
		}
		if p.apiSubcommand.options.chartOptions.Version != "" {
			args = append(args, fmt.Sprintf("--%s", helmChartVersionFlag), p.apiSubcommand.options.chartOptions.Version)
		}
		if err := util.RunCmd("Creating the API", os.Args[0], args...); err != nil {
			return err
		}
	}

	return nil
}

// addInitCustomizations will perform the required customizations for this plugin on the common base
func addInitCustomizations(projectName string, componentConfig bool) error {
	managerFile := filepath.Join("config", "manager", "manager.yaml")

	// todo: we ought to use afero instead. Replace this methods to insert/update
	// by https://github.com/kubernetes-sigs/kubebuilder/pull/2119

	// Add leader election arg in config/manager/manager.yaml and in config/default/manager_auth_proxy_patch.yaml
	if componentConfig {
		err := util.InsertCode(managerFile,
			"- /manager",
			fmt.Sprintf("\n        args:\n        - --leader-election-id=%s", projectName))
		if err != nil {
			return err
		}

		err = util.InsertCode(filepath.Join("config", "default", "manager_auth_proxy_patch.yaml"),
			"memory: 64Mi",
			fmt.Sprintf("\n      - name: manager\n        args:\n        - \"--leader-election-id=%s\"", projectName))
		if err != nil {
			return err
		}
		// Remove the webhook option for the componentConfig since webhooks are not supported by helm
		err = util.ReplaceInFile(filepath.Join("config", "manager", "controller_manager_config.yaml"),
			"webhook:\n  port: 9443", "")
		if err != nil {
			return err
		}
	} else {
		err := util.InsertCode(managerFile,
			"--leader-elect",
			fmt.Sprintf("\n        - --leader-election-id=%s", projectName))
		if err != nil {
			return err
		}
		err = util.InsertCode(filepath.Join("config", "default", "manager_auth_proxy_patch.yaml"),
			"- \"--leader-elect\"",
			fmt.Sprintf("\n        - \"--leader-election-id=%s\"", projectName))
		if err != nil {
			return err
		}
	}

	// Remove the call to the command as manager. Helm has not been exposing this entrypoint
	// todo: provide the manager entrypoint for helm and then remove it
	const command = `command:
        - /manager
        `
	err := util.ReplaceInFile(managerFile, command, "")
	if err != nil {
		return err
	}

	if err := sdkpluginutil.UpdateKustomizationsInit(); err != nil {
		return fmt.Errorf("error updating kustomization.yaml files: %v", err)
	}

	return nil
}
