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
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

//nolint:maligned
type packagemanifestsCmd struct {
	// Common options.
	projectName   string
	version       string
	fromVersion   string
	inputDir      string
	outputDir     string
	kustomizeDir  string
	deployDir     string
	crdsDir       string
	updateObjects bool
	stdout        bool
	quiet         bool

	// Package manifest options.
	channelName      string
	isDefaultChannel bool
}

// NewCmd returns the 'packagemanifests' command configured for the new project layout.
func NewCmd() *cobra.Command {
	c := &packagemanifestsCmd{}

	cmd := &cobra.Command{
		Use:     "packagemanifests",
		Short:   "Generates package manifests data for the operator",
		Long:    longHelp,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			cfg, err := projutil.ReadConfig()
			if err != nil {
				log.Fatal(fmt.Errorf("error reading configuration: %v", err))
			}
			if err := c.setDefaults(cfg); err != nil {
				return err
			}

			if err = c.validate(); err != nil {
				return fmt.Errorf("invalid command options: %v", err)
			}
			if err = c.run(cfg); err != nil {
				log.Fatalf("Error generating package manifests: %v", err)
			}

			return nil
		},
	}

	c.addFlagsTo(cmd.Flags())

	return cmd
}

func (c *packagemanifestsCmd) addFlagsTo(fs *pflag.FlagSet) {
	fs.StringVarP(&c.version, "version", "v", "", "Semantic version of the packaged operator")
	fs.StringVar(&c.fromVersion, "from-version", "", "Semantic version of the operator being upgraded from")
	fs.StringVar(&c.inputDir, "input-dir", "", "Directory to read existing package manifests from. "+
		"This directory is the parent of individual versioned package directories, and different from --deploy-dir")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory in which to write package manifests")
	fs.StringVar(&c.kustomizeDir, "kustomize-dir", filepath.Join("config", "manifests"),
		"Directory containing kustomize bases and a kustomization.yaml for operator-framework manifests")
	fs.StringVar(&c.deployDir, "deploy-dir", "", "Root directory for operator manifests such as "+
		"Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir")
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Root directory for CustomResoureDefinition manifests")
	fs.StringVar(&c.channelName, "channel", "", "Channel name for the generated package")
	fs.BoolVar(&c.isDefaultChannel, "default-channel", false, "Use the channel passed to --channel "+
		"as the package manifest file's default channel")
	fs.BoolVar(&c.updateObjects, "update-objects", true, "Update non-CSV objects in this package, "+
		"ex. CustomResoureDefinitions, Roles")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.stdout, "stdout", false, "Write package to stdout")
}
