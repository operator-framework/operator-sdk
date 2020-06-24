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

package bundle

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

//nolint:maligned
type bundleCmd struct {
	// Options to turn on different parts of bundling.
	manifests bool
	metadata  bool

	// Common options.
	operatorName string
	version      string
	inputDir     string
	outputDir    string
	kustomizeDir string
	deployDir    string
	crdsDir      string
	stdout       bool
	quiet        bool

	// Metadata options.
	channels       string
	defaultChannel string
	overwrite      bool
}

//nolint:maligned
type bundleCmdLegacy struct {
	bundleCmd

	apisDir          string
	interactiveLevel projutil.InteractiveLevel
	interactive      bool
}

// NewCmd returns the 'bundle' command configured for the new project layout.
func NewCmd() *cobra.Command {
	c := &bundleCmd{}
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Generates bundle data for the operator",
		Long:    longHelp,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			// Generate manifests and metadata by default if no flags are set so the default behavior is "do everything".
			fs := cmd.Flags()
			if !fs.Changed("metadata") && !fs.Changed("manifests") {
				c.manifests = true
				c.metadata = true
			}

			cfg, err := kbutil.ReadConfig()
			if err != nil {
				return fmt.Errorf("error reading configuration: %v", err)
			}
			c.setCommonDefaults(cfg)

			// Validate command args before running so a preceding mode doesn't run
			// before a following validation fails.
			if c.manifests {
				if err = c.validateManifests(cfg); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
			}
			if c.metadata {
				if err = c.validateMetadata(cfg); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
			}

			// Run command logic.
			if c.manifests {
				if err = c.runManifests(cfg); err != nil {
					log.Fatalf("Error generating bundle manifests: %v", err)
				}
			}
			if c.metadata {
				if err = c.runMetadata(); err != nil {
					log.Fatalf("Error generating bundle metadata: %v", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&c.kustomizeDir, "kustomize-dir", filepath.Join("config", "manifests"),
		"Directory containing kustomize bases and a kustomization.yaml for operator-framework manifests")
	cmd.Flags().BoolVar(&c.stdout, "stdout", false, "Write bundle manifest to stdout")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

// NewCmdLegacy returns the 'bundle' command configured for the legacy project layout.
func NewCmdLegacy() *cobra.Command {
	c := &bundleCmdLegacy{}
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Generates bundle data for the operator",
		Long:    longHelpLegacy,
		Example: examplesLegacy,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			// Check if the user has any specific preference to enable/disable
			// interactive prompts. Default behaviour is to disable the prompt
			// unless a base bundle does not exist.
			if cmd.Flags().Changed("interactive") {
				if c.interactive {
					c.interactiveLevel = projutil.InteractiveOnAll
				} else {
					c.interactiveLevel = projutil.InteractiveHardOff
				}
			}

			// Generate manifests and metadata by default if no flags are set so
			// the default behavior is "do everything".
			fs := cmd.Flags()
			if !fs.Changed("metadata") && !fs.Changed("manifests") {
				c.metadata = true
				c.manifests = true
			}

			c.setCommonDefaults()

			// Validate command args before running so a preceding mode doesn't run
			// before a following validation fails.
			if c.manifests {
				if err = c.validateManifests(); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
			}
			if c.metadata {
				if err = c.validateMetadata(); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
			}

			// Run command logic.
			if c.manifests {
				if err = c.runManifests(); err != nil {
					log.Fatalf("Error generating bundle manifests: %v", err)
				}
			}
			if c.metadata {
				if err = c.runMetadata(); err != nil {
					log.Fatalf("Error generating bundle metadata: %v", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&c.apisDir, "apis-dir", "", "Root directory for API type defintions")
	cmd.Flags().BoolVar(&c.interactive, "interactive", false, "When set or no bundle base exists, an interactive "+
		"command prompt will be presented to accept bundle ClusterServiceVersion metadata")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

// TODO(estroz): add flag to skip API metadata regeneration.
func (c *bundleCmd) addCommonFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVar(&c.manifests, "manifests", false, "Generate bundle manifests")
	fs.BoolVar(&c.metadata, "metadata", false, "Generate bundle metadata and Dockerfile")

	fs.StringVar(&c.operatorName, "operator-name", "", "Name of the bundle's operator")
	fs.StringVarP(&c.version, "version", "v", "", "Semantic version of the operator in the generated bundle. "+
		"Only set if creating a new bundle or upgrading your operator")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory to read an existing bundle from. "+
		"This directory is the parent of your bundle 'manifests' directory, and different from --deploy-dir")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory to write the bundle to")
	fs.StringVar(&c.deployDir, "deploy-dir", "", "Root directory for operator manifests such as "+
		"Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir")
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Root directory for CustomResoureDefinition manifests")
	fs.StringVar(&c.channels, "channels", "alpha", "A comma-separated list of channels the bundle belongs to")
	fs.StringVar(&c.defaultChannel, "default-channel", "", "The default channel for the bundle")
	fs.BoolVar(&c.overwrite, "overwrite", true, "Overwrite the bundle's metadata and Dockerfile if they exist")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
}
