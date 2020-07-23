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
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"

	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/plugins/util/kustomize"
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
	projectName string
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

			if err := c.setDefaults(cfg); err != nil {
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
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory containing existing kustomize files")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory to write kustomize files")
	fs.StringVar(&c.apisDir, "apis-dir", "", "Root directory for API type defintions")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.interactive, "interactive", false, "When set or no kustomize base exists, an interactive "+
		"command prompt will be presented to accept non-inferrable metadata")
}

// defaultDir is the default directory in which to generate kustomize bases and the kustomization.yaml.
var defaultDir = filepath.Join("config", "manifests")

// setDefaults sets command defaults.
func (c *manifestsCmd) setDefaults(cfg *config.Config) (err error) {
	if c.projectName, err = genutil.GetOperatorName(cfg); err != nil {
		return err
	}

	if c.inputDir == "" {
		c.inputDir = defaultDir
	}
	if c.outputDir == "" {
		c.outputDir = defaultDir
	}
	if c.apisDir == "" {
		if cfg.MultiGroup {
			c.apisDir = "apis"
		} else {
			c.apisDir = "api"
		}
	}
	return nil
}

// kustomization.yaml file contents for manifests. this should always be written to
// config/manifests/kustomization.yaml since it only references files in config.
const manifestsKustomization = `resources:
- ../default
- ../samples
- ../scorecard
`

// run generates kustomize bundle bases and a kustomization.yaml if one does not exist.
func (c manifestsCmd) run(cfg *config.Config) error {

	if !c.quiet {
		fmt.Println("Generating kustomize files in", c.outputDir)
	}

	csvGen := gencsv.Generator{
		OperatorName: c.projectName,
		OperatorType: projutil.PluginKeyToOperatorType(cfg.Layout),
	}
	opts := []gencsv.Option{
		gencsv.WithBase(c.inputDir, c.apisDir, c.interactiveLevel),
		gencsv.WithBaseWriter(c.outputDir),
	}
	if err := csvGen.Generate(cfg, opts...); err != nil {
		return fmt.Errorf("error generating kustomize bases: %v", err)
	}

	// Write a kustomization.yaml to outputDir if one does not exist.
	if err := kustomize.WriteIfNotExist(c.outputDir, manifestsKustomization); err != nil {
		return fmt.Errorf("error writing kustomization.yaml: %v", err)
	}

	if !c.quiet {
		fmt.Println("Kustomize files generated successfully")
	}

	return nil
}
