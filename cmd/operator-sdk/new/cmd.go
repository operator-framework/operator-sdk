// Copyright 2018 The Operator-SDK Authors
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

package new

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	catalog "github.com/operator-framework/operator-sdk/internal/pkg/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/pkg/config"
	scaffoldcmd "github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewCmd() *cobra.Command {
	c := &scaffoldcmd.NewCmd{}

	newCmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Creates a new operator application",
		Long: `The operator-sdk new command creates a new operator application and
generates a default directory layout based on the input <project-name>.

<project-name> is the project name of the new operator. (e.g app-operator)

For example:
	$ mkdir $GOPATH/src/github.com/example.com/
	$ cd $GOPATH/src/github.com/example.com/
	$ operator-sdk new app-operator
generates a skeletal app-operator application in $GOPATH/src/github.com/example.com/app-operator.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("command %s requires exactly one argument", cmd.CommandPath())
			}
			c.ProjectName = args[0]
			if c.ProjectName == "" {
				return fmt.Errorf("project name must not be empty")
			}

			// --repo must be set if the project is not under $GOPATH/src.
			if err := config.CheckRepo(); err != nil {
				return fmt.Errorf("%v: projects not in $GOPATH/src require --%s=<repo> be set", err, config.RepoOpt)
			}
			if !cmd.Flags().Changed(config.RepoOpt) {
				repo, err := config.GetGoPathRepo()
				if err != nil {
					return err
				}
				viper.Set(config.RepoOpt, filepath.Join(repo, c.ProjectName))
			}

			// By default, both CRDs and OLM manifest dirs are contained in the
			// deploy dir. If --deploy-dir is set and any child dirs in the config
			// are not, make sure they have the correct parent. Otherwise update
			// config values that contain CRDs or OLM dir paths.
			if cmd.Flags().Changed(config.DeployDirOpt) {
				deployDir := viper.GetString(config.DeployDirOpt)
				setDirChildren(scaffold.DeployDir, deployDir)
				// Update child dirs if the deploy dir has been changed.
				if f := cmd.Flags().Lookup(config.CRDsDirOpt); f != nil && f.Changed {
					setDirChildren(filepath.Join(deployDir, filepath.Base(scaffold.CRDsDir)), f.Value.String())
				}
				if f := cmd.Flags().Lookup(config.OLMCatalogDirOpt); f != nil && f.Changed {
					setDirChildren(filepath.Join(deployDir, filepath.Base(catalog.OLMCatalogDir)), f.Value.String())
				}
			} else {
				if f := cmd.Flags().Lookup(config.CRDsDirOpt); f != nil && f.Changed {
					setDirChildren(scaffold.CRDsDir, f.Value.String())
				}
				if f := cmd.Flags().Lookup(config.OLMCatalogDirOpt); f != nil && f.Changed {
					setDirChildren(catalog.OLMCatalogDir, f.Value.String())
				}
			}

			// If --config was not set, write a new config file for the project.
			c.WriteConfig = !cmd.Flags().Changed(config.ConfigOpt)

			return c.Run()
		},
	}

	newCmd.Flags().StringVar(&c.APIVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1) - used with \"ansible\" or \"helm\" types")
	newCmd.Flags().StringVar(&c.Kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService) - used with \"ansible\" or \"helm\" types")
	newCmd.Flags().StringVar(&c.OperatorType, "type", "go", "Type of operator to initialize (choices: \"go\", \"ansible\" or \"helm\")")
	newCmd.Flags().StringVar(&c.DepManager, "dep-manager", "modules", `Dependency manager the new project will use (choices: "dep", "modules")`)
	newCmd.Flags().BoolVar(&c.SkipGit, "skip-git-init", false, "Do not init the directory as a git repository")
	newCmd.Flags().StringVar(&c.HeaderFile, "header-file", "", "Path to file containing headers for generated Go files. Copied to hack/boilerplate.go.txt")

	newCmd.Flags().BoolVar(&c.Ansible.GeneratePlaybook, "generate-playbook", false, "Generate a playbook skeleton. (Only used for --type ansible)")

	newCmd.Flags().StringVar(&c.Helm.ChartRef, "helm-chart", "", "Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path)")
	newCmd.Flags().StringVar(&c.Helm.ChartVersion, "helm-chart-version", "", "Specific version of the helm chart (default is latest version)")
	newCmd.Flags().StringVar(&c.Helm.ChartRepo, "helm-chart-repo", "", "Chart repository URL for the requested helm chart")

	fset := pflag.NewFlagSet("", pflag.ExitOnError)
	fset.String(config.RepoOpt, "", "Project repository path, ex. github.com/operator-framework/operator-sdk. This flag is required if the project is not in $GOPATH/src")
	fset.String(config.DeployDirOpt, scaffold.DeployDir, "Directory to write deployment manifests. This flag is optional")
	fset.String(config.CRDsDirOpt, scaffold.CRDsDir, "Directory to write CRD and CR manifests. This flag is optional")
	fset.String(config.APIsDirOpt, scaffold.APIsDir, "Directory to write Kubernetes resource API code. This flag is optional")
	fset.String(catalog.OLMCatalogDirOpt, catalog.OLMCatalogDir, "Directory to write OLM manifests. This flag is optional")
	viper.BindPFlags(fset)
	newCmd.Flags().AddFlagSet(fset)

	return newCmd
}

func setDirChildren(fromDir, toDir string) {
	configKeys := viper.AllKeys()
	lenConfigKeys := len(configKeys)
	for lenConfigKeys > 0 {
		for _, k := range configKeys {
			v := viper.Get(k)
			switch t := v.(type) {
			case string:
				if strings.HasPrefix(t, fromDir) {
					viper.Set(k, strings.Replace(t, fromDir, toDir, 1))
				}
			case []string, []interface{}:
				vs := []string{}
				for _, tv := range t.([]interface{}) {
					if s, ok := tv.(string); ok {
						if strings.HasPrefix(s, fromDir) {
							vs = append(vs, strings.Replace(s, fromDir, toDir, 1))
						}
					}
				}
				viper.Set(k, vs)
			case map[string]interface{}, map[string]string:
				for mk := range t.(map[string]interface{}) {
					configKeys = append(configKeys, k+"."+mk)
				}
			}
		}
		configKeys = configKeys[lenConfigKeys:]
		lenConfigKeys = len(configKeys)
	}
}
