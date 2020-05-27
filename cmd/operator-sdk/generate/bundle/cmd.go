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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

//nolint:maligned
type bundleCmd struct {
	// Options to turn on different parts of bundling.
	kustomize bool
	manifests bool
	metadata  bool

	// Common options.
	operatorName string
	version      string
	inputDir     string
	outputDir    string
	manifestRoot string
	apisDir      string
	crdsDir      string
	stdout       bool
	quiet        bool

	// Interactive options.
	interactiveLevel projutil.InteractiveLevel
	interactive      bool

	// Metadata options.
	channels       string
	defaultChannel string
	overwrite      bool
}

// NewCmd returns the 'bundle' command configured for the new project layout.
func NewCmd() *cobra.Command {
	c := &bundleCmd{}
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Generates bundle data for the operator",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// Generate kustomize bases, manifests, and metadata by default if no
			// flags are set so the default behavior is "do everything".
			fs := cmd.Flags()
			if !fs.Changed("kustomize") && !fs.Changed("metadata") && !fs.Changed("manifests") {
				c.kustomize = true
				c.manifests = true
				c.metadata = true
			}

			cfg, err := kbutil.ReadConfig()
			if err != nil {
				return fmt.Errorf("error reading configuration: %v", err)
			}
			c.setCommonDefaults(cfg)

			if c.kustomize {
				if err = c.runKustomize(cfg); err != nil {
					log.Fatalf("Error generating bundle bases: %v", err)
				}
			}
			if c.manifests {
				if err = c.validateManifests(cfg); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
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

	cmd.Flags().BoolVar(&c.kustomize, "kustomize", false, "Generate kustomize bases")
	cmd.Flags().BoolVar(&c.manifests, "manifests", false, "Generate bundle manifests")
	cmd.Flags().BoolVar(&c.metadata, "metadata", false, "Generate bundle metadata and Dockerfile")
	cmd.Flags().BoolVar(&c.stdout, "stdout", false, "Write bundle manifest to stdout")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

// NewCmdLegacy returns the 'bundle' command configured for the legacy project layout.
func NewCmdLegacy() *cobra.Command {
	c := &bundleCmd{}
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Generates bundle data for the operator",
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

			c.setCommonDefaultsLegacy()

			if c.manifests {
				if err = c.validateManifestsLegacy(); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
				if err = c.runManifestsLegacy(); err != nil {
					log.Fatalf("Error generating bundle manifests: %v", err)
				}
			}
			if c.metadata {
				if err = c.runMetadataLegacy(); err != nil {
					log.Fatalf("Error generating bundle metadata: %v", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&c.manifests, "manifests", false, "Generate bundle manifests")
	cmd.Flags().BoolVar(&c.metadata, "metadata", false, "Generate bundle metadata and Dockerfile")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

// TODO(estroz): add flag to skip API metadata regeneration.
func (c *bundleCmd) addCommonFlagsTo(fs *pflag.FlagSet) {
	fs.StringVar(&c.operatorName, "operator-name", "", "Name of the bundle's operator")
	fs.StringVarP(&c.version, "version", "v", "", "Semantic version of the operator in the generated bundle. "+
		"Only set if creating a new bundle or upgrading your operator")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory to read an existing bundle from. "+
		"This directory is the parent of your bundle 'manifests' directory, and different from --manifest-root")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory to write the bundle to")

	fs.StringVar(&c.manifestRoot, "manifest-root", "", "Root directory for operator manifests such as "+
		"Deployments and RBAC, ex. 'deploy' or 'config'. This directory is different from that passed to --input-dir")
	// NB(estroz): still debating the name of this flag. For now, hide it as an
	// "alpha" flag so we do not have to deprecate it if we change this name.
	// TODO(estroz): decide on this flag's name before making 'init' default.
	if err := fs.MarkHidden("manifest-root"); err != nil {
		panic(err)
	}

	fs.StringVar(&c.apisDir, "apis-dir", "", "Root directory for API type defintions")
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Root directory for CustomResoureDefinition manifests")
	fs.StringVar(&c.channels, "channels", "alpha", "A comma-separated list of channels the bundle belongs to")
	fs.StringVar(&c.defaultChannel, "default-channel", "", "The default channel for the bundle")
	fs.BoolVar(&c.overwrite, "overwrite", false, "Overwrite the bundle's metadata and Dockerfile if they exist")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.interactive, "interactive", false, "When set or no bundle base exists, an interactive "+
		"command prompt will be presented to accept bundle ClusterServiceVersion metadata")
}
