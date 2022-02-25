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
)

//nolint:maligned
type bundleCmd struct {
	// Options to turn on different parts of bundling.
	manifests bool
	metadata  bool

	// Common options.
	version      string
	inputDir     string
	outputDir    string
	kustomizeDir string
	deployDir    string
	crdsDir      string
	stdout       bool
	quiet        bool
	// ServiceAccount names to consider outside of the operator's service account.
	extraServiceAccounts []string

	// Metadata options.
	channels       string
	defaultChannel string
	overwrite      bool

	// These are set if a PROJECT config is not present.
	layout      string
	packageName string

	// Use Image Digests flag to toggle using traditional Image tags vs SHA Digests
	useImageDigests bool
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
				if c.version == "" {
					c.println("Generating bundle")
				} else {
					c.println("Generating bundle version", c.version)
				}
			}

			if err := c.setDefaults(); err != nil {
				return err
			}

			// Validate command args before running so a preceding mode doesn't run
			// before a following validation fails.
			if c.manifests {
				if err := c.validateManifests(); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
			}

			// Run command logic.
			if c.manifests {
				if err := c.runManifests(); err != nil {
					log.Fatalf("Error generating bundle manifests: %v", err)
				}
			}
			if c.metadata {
				if err := c.runMetadata(); err != nil {
					log.Fatalf("Error generating bundle metadata: %v", err)
				}
			}

			return nil
		},
	}

	c.addFlagsTo(cmd.Flags())

	return cmd
}

func (c *bundleCmd) addFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVar(&c.manifests, "manifests", false, "Generate bundle manifests")
	fs.BoolVar(&c.metadata, "metadata", false, "Generate bundle metadata and Dockerfile")

	fs.StringVarP(&c.version, "version", "v", "", "Semantic version of the operator in the generated bundle. "+
		"Only set if creating a new bundle or upgrading your operator")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory to read cluster-ready operator manifests from. "+
		"This option is mutually exclusive with --deploy-dir/--crds-dir and piping to stdin. "+
		"This option should not be passed an existing bundle directory, as this bundle will not contain the correct "+
		"set of manifests required to generate a CSV. Use --kustomize-dir to pass a base CSV")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory to write the bundle to")
	// TODO(estroz): deprecate this in favor of --intput-dir.
	fs.StringVar(&c.deployDir, "deploy-dir", "", "Directory to read cluster-ready operator manifests from. "+
		"If --crds-dir is not set, CRDs are ready from this directory. "+
		"This option is mutually exclusive with --input-dir and piping to stdin")
	// TODO(estroz): deprecate this in favor of --intput-dir.
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Directory to read cluster-ready CustomResoureDefinition manifests from. "+
		"This option can only be used if --deploy-dir is set")
	fs.StringVar(&c.kustomizeDir, "kustomize-dir", filepath.Join("config", "manifests"),
		"Directory containing kustomize bases in a \"bases\" dir and a kustomization.yaml for operator-framework manifests")
	fs.StringVar(&c.channels, "channels", "alpha", "A comma-separated list of channels the bundle belongs to")
	fs.StringVar(&c.defaultChannel, "default-channel", "", "The default channel for the bundle")
	fs.StringSliceVar(&c.extraServiceAccounts, "extra-service-accounts", nil,
		"Names of service accounts, outside of the operator's Deployment account, "+
			"that have bindings to {Cluster}Roles that should be added to the CSV")
	fs.BoolVar(&c.overwrite, "overwrite", true, "Overwrite the bundle's metadata and Dockerfile if they exist")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.stdout, "stdout", false, "Write bundle manifest to stdout")

	fs.StringVar(&c.packageName, "package", "", "Bundle's package name")

	fs.BoolVar(&c.useImageDigests, "use-image-digests", false, "Use SHA Digest for images")
}

func (c bundleCmd) println(a ...interface{}) {
	if !(c.quiet || c.stdout) {
		fmt.Println(a...)
	}
}
