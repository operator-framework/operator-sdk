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

package kustomize

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/yaml"

	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2/templates/config/manifests"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const longHelp = `
Running 'generate kustomize manifests' will (re)generate kustomize bases and a kustomization.yaml in
'config/manifests', which are used to build operator-framework manifests by other operator-sdk commands.
This command will interactively ask for UI metadata, an important component of manifest bases,
by default unless a base already exists or you set '--interactive=false'.
`

const examples = `
  $ operator-sdk generate kustomize manifests

  Display name for the operator (required):
  > memcached-operator
  ...

  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml

  # After generating kustomize bases and a kustomization.yaml, you can generate a bundle or package manifests.

  # To generate a bundle:
  $ kustomize build config/manifests | operator-sdk generate bundle --version 0.0.1

  # To generate package manifests:
  $ kustomize build config/manifests | operator-sdk generate packagemanifests --version 0.0.1
`

//nolint:maligned
type manifestsCmd struct {
	packageName string
	inputDir    string
	outputDir   string
	apisDir     string
	quiet       bool

	// Interactive options.
	interactiveLevel projutil.InteractiveLevel
	interactive      bool
}

// newManifestsCmd returns the 'manifests' command configured for the new project layout.
func newManifestsCmd() *cobra.Command {
	c := &manifestsCmd{}
	cmd := &cobra.Command{
		Use:     "manifests",
		Short:   "Generates kustomize bases and a kustomization.yaml for operator-framework manifests",
		Long:    longHelp,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			// Check if the user has any specific preference to enable/disable interactive prompts.
			// Default behaviour is to disable the prompt unless a base does not exist.
			if cmd.Flags().Changed("interactive") {
				if c.interactive {
					c.interactiveLevel = projutil.InteractiveOnAll
				} else {
					c.interactiveLevel = projutil.InteractiveHardOff
				}
			}

			cfg, err := projutil.ReadConfig()
			if err != nil {
				return fmt.Errorf("error reading configuration: %v", err)
			}

			if err = c.setDefaults(cfg); err != nil {
				return err
			}

			// Run command logic.
			if err = c.run(cfg); err != nil {
				log.Fatalf("Error generating kustomize files: %v", err)
			}

			return nil
		},
	}

	c.addFlagsTo(cmd.Flags())

	return cmd
}

func (c *manifestsCmd) addFlagsTo(fs *pflag.FlagSet) {
	fs.StringVar(&c.packageName, "package", "", "Package name")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory containing existing kustomize files")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory to write kustomize files")
	fs.StringVar(&c.apisDir, "apis-dir", "", "Root directory for API type defintions")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.interactive, "interactive", false, "When set to false, if no kustomize base exists, an interactive "+
		"command prompt will be presented to accept non-inferrable metadata")
}

// defaultDir is the default directory in which to generate kustomize bases and the kustomization.yaml.
var defaultDir = filepath.Join("config", "manifests")

// setDefaults sets command defaults.
func (c *manifestsCmd) setDefaults(cfg config.Config) error {
	if c.packageName == "" {
		c.packageName = cfg.GetProjectName()
	}

	if c.inputDir == "" {
		c.inputDir = defaultDir
	}
	if c.outputDir == "" {
		c.outputDir = defaultDir
	}

	if c.apisDir == "" {
		if cfg.IsMultiGroup() {
			c.apisDir = "apis"
		} else {
			c.apisDir = "api"
		}
	}
	return nil
}

// run generates kustomize bundle bases and a kustomization.yaml if one does not exist.
func (c manifestsCmd) run(cfg config.Config) error {

	if !c.quiet {
		fmt.Println("Generating kustomize files in", c.outputDir)
	}

	// Older config layouts do not have a ProjectName field, so use the current directory name.
	if c.packageName == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %v", err)
		}
		c.packageName = strings.ToLower(filepath.Base(dir))
		if err := validation.IsDNS1123Label(c.packageName); err != nil {
			return fmt.Errorf("project name (%s) is invalid: %v", c.packageName, err)
		}
	}

	operatorType := projutil.PluginChainToOperatorType(cfg.GetPluginChain())
	relBasePath := filepath.Join("bases", c.packageName+".clusterserviceversion.yaml")
	basePath := filepath.Join(c.inputDir, relBasePath)
	gvks, err := getGVKs(cfg)
	if err != nil {
		return err
	}
	base := bases.ClusterServiceVersion{
		OperatorName: c.packageName,
		OperatorType: operatorType,
		APIsDir:      c.apisDir,
		Interactive:  requiresInteraction(basePath, c.interactiveLevel),
		GVKs:         gvks,
	}
	// Set BasePath only if it exists. If it doesn't, a new base will be generated
	// if BasePath is empty.
	if genutil.IsExist(basePath) {
		base.BasePath = basePath
	}
	csv, err := base.GetBase()
	if err != nil {
		return fmt.Errorf("error getting ClusterServiceVersion base: %v", err)
	}

	csvBytes, err := k8sutil.GetObjectBytes(csv, yaml.Marshal)
	if err != nil {
		return fmt.Errorf("error marshaling CSV base: %v", err)
	}

	// todo: remove it when the OLM starts to support https://github.com/operator-framework/api/pull/100
	const cleanup = "cleanup:\n    enabled: false\n  "
	csvBytes = bytes.ReplaceAll(csvBytes, []byte(cleanup), []byte(""))

	if err = os.MkdirAll(filepath.Join(c.outputDir, "bases"), 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(c.outputDir, relBasePath)
	if err = ioutil.WriteFile(outputPath, csvBytes, 0644); err != nil {
		return fmt.Errorf("error writing CSV base: %v", err)
	}

	// Write a kustomization.yaml to outputDir if one does not exist.
	kustomization := manifests.Kustomization{SupportsWebhooks: operatorType == projutil.OperatorTypeGo}
	// Ensure the path to the manifest directory is correctly carried through
	kustomization.Path = c.outputDir
	err = machinery.NewScaffold(machinery.Filesystem{FS: afero.NewOsFs()}, machinery.WithConfig(cfg)).Execute(
		&kustomization,
	)
	if err != nil {
		return fmt.Errorf("error scaffolding manifests: %v", err)
	}

	if !c.quiet {
		fmt.Println("Kustomize files generated successfully")
	}

	return nil
}

// requiresInteraction checks if the combination of ilvl and basePath existence
// requires the generator prompt a user interactively.
func requiresInteraction(basePath string, ilvl projutil.InteractiveLevel) bool {
	return (ilvl == projutil.InteractiveSoftOff && genutil.IsNotExist(basePath)) || ilvl == projutil.InteractiveOnAll
}

func getGVKs(cfg config.Config) ([]schema.GroupVersionKind, error) {
	resources, err := cfg.GetResources()
	if err != nil {
		return nil, err
	}
	gvks := make([]schema.GroupVersionKind, len(resources))
	for i, gvk := range resources {
		// check if the resource has an specific domain
		// otherwise use the config.Domain.
		if cfg.GetVersion().Compare(cfgv2.Version) == 0 {
			gvk.Domain = cfg.GetDomain()
		}
		gvks[i].Group = gvk.QualifiedGroup()
		gvks[i].Version = gvk.Version
		gvks[i].Kind = gvk.Kind
	}
	return gvks, nil
}
