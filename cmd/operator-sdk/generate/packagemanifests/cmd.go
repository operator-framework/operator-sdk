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

package packagemanifests

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

//nolint:maligned
type packagemanifestsCmd struct {
	// Options to turn on different parts of packaging.
	kustomize bool
	manifests bool

	// Common options.
	operatorName string
	version      string
	fromVersion  string
	inputDir     string
	outputDir    string
	deployDir    string
	apisDir      string
	crdsDir      string
	stdout       bool
	quiet        bool

	// CRD update options
	updateCRDs     bool
	updateCRDsFlag *pflag.Flag

	// Interactive options.
	interactiveLevel projutil.InteractiveLevel
	interactive      bool

	// Package manifest options.
	channelName      string
	isDefaultChannel bool
}

// NewCmd returns the 'packagemanifests' command configured for the new project layout.
func NewCmd() *cobra.Command {
	c := &packagemanifestsCmd{}

	cmd := &cobra.Command{
		Use:   "packagemanifests",
		Short: "Generates a package manifests format",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			// Check if the user has any specific preference to enable/disable
			// interactive prompts. Default behaviour is to disable the prompt
			// unless a base package does not exist.
			if cmd.Flags().Changed("interactive") {
				if c.interactive {
					c.interactiveLevel = projutil.InteractiveOnAll
				} else {
					c.interactiveLevel = projutil.InteractiveHardOff
				}
			}

			// Generate kustomize bases and manifests by default if no flags are set
			// so the default behavior is "do everything".
			fs := cmd.Flags()
			if !fs.Changed("kustomize") && !fs.Changed("manifests") {
				c.kustomize = true
				c.manifests = true
			}

			cfg, err := kbutil.ReadConfig()
			if err != nil {
				log.Fatal(fmt.Errorf("error reading configuration: %v", err))
			}
			c.setCommonDefaults(cfg)

			if c.kustomize {
				if err = c.runKustomize(cfg); err != nil {
					log.Fatalf("Error generating package bases: %v", err)
				}
			}
			if c.manifests {
				if err = c.validateManifests(); err != nil {
					return fmt.Errorf("invalid command options: %v", err)
				}
				if err = c.runManifests(cfg); err != nil {
					log.Fatalf("Error generating package manifests: %v", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&c.kustomize, "kustomize", false, "Generate kustomize bases")
	cmd.Flags().BoolVar(&c.manifests, "manifests", false, "Generate package manifests")
	cmd.Flags().BoolVar(&c.stdout, "stdout", false, "Write package to stdout")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

func (c *packagemanifestsCmd) addCommonFlagsTo(fs *pflag.FlagSet) {
	fs.StringVar(&c.operatorName, "operator-name", "", "Name of the packaged operator")
	fs.StringVarP(&c.version, "version", "v", "", "Semantic version of the packaged operator")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory to read existing package manifests from. "+
		"This directory is the parent of individual versioned package directories, and different from --deploy-dir")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory in which to write package manifests")
	fs.StringVar(&c.deployDir, "deploy-dir", "", "Root directory for operator manifests such as "+
		"Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir")
	fs.StringVar(&c.apisDir, "apis-dir", "", "Root directory for API type defintions")
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Root directory for CustomResoureDefinition manifests")
	fs.StringVar(&c.channelName, "channel", "", "Channel name for the generated package")
	fs.BoolVar(&c.isDefaultChannel, "default-channel", false, "Use the channel passed to --channel "+
		"as the package manifest file's default channel")
	fs.BoolVar(&c.updateCRDs, "update-crds", false, "Update CustomResoureDefinition manifests in this package. "+
		"This will occur automatically when a new package is generated")
	c.updateCRDsFlag = fs.Lookup("update-crds")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.interactive, "interactive", false, "When set or no package base exists, an interactive "+
		"command prompt will be presented to accept package ClusterServiceVersion metadata")
}
