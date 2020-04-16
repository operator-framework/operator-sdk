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
	manifestRoot string
	apisDir      string
	crdsDir      string
	updateCRDs   bool
	stdout       bool
	quiet        bool

	// Package manifest options.
	channelName      string
	isDefaultChannel bool
}

//nolint:lll
func NewCmd() *cobra.Command {
	c := &packagemanifestsCmd{}

	cmd := &cobra.Command{
		Use:   "packagemanifests",
		Short: "Generates a package manifests format",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			// Turn them all on by default here to simplify individual flag usage.
			fs := cmd.Flags()
			if !fs.Changed("kustomize") && !fs.Changed("manifests") {
				c.kustomize = true
				c.manifests = true
			}

			cfg, err := kbutil.ReadConfig()
			if err != nil {
				log.Fatal(fmt.Errorf("error reading configuration: %v", err))
			}
			c.setDefaults(cfg)

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
	cmd.Flags().BoolVar(&c.manifests, "manifests", false, "Generate a package")
	cmd.Flags().BoolVar(&c.stdout, "stdout", false, "Write package to stdout")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

//nolint:lll
func NewCmdLegacy() *cobra.Command {
	c := &packagemanifestsCmd{}

	cmd := &cobra.Command{
		Use:   "packagemanifests",
		Short: "Generates a package manifests format",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			c.setDefaultsLegacy()
			if err := c.validateLegacy(); err != nil {
				return fmt.Errorf("invalid command options: %v", err)
			}
			if err := c.runLegacy(); err != nil {
				log.Fatalf("Error generating package manifests format: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&c.fromVersion, "from-version", "", "Semantic version of the existing ClusterServiceVersion to update")

	c.addCommonFlagsTo(cmd.Flags())

	return cmd
}

func (c *packagemanifestsCmd) addCommonFlagsTo(fs *pflag.FlagSet) {
	fs.StringVar(&c.operatorName, "operator-name", "", "Name of the operator to generate the package for")
	fs.StringVarP(&c.version, "version", "v", "", "Semantic version of the generated package")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory to read an existing package manifests format from")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory in which to write the package manifests format")
	fs.StringVar(&c.manifestRoot, "manifest-root", "", "Root directory for operator manifests, ex. Deployment and RBAC")
	fs.StringVar(&c.apisDir, "apis-dir", "", "Root directory for API type defintions")
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Root directory for CustomResoureDefinition and Custom Resource manifests")
	fs.StringVar(&c.channelName, "channel", "", "Channel name for the generated package")
	fs.BoolVar(&c.isDefaultChannel, "default-channel", false, "Use the channel passed to --channel "+
		"as the package manifests' default channel")
	fs.BoolVar(&c.updateCRDs, "update-crds", false, "Update CustomResoureDefinition manifests "+
		"in the package for this version")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
}
