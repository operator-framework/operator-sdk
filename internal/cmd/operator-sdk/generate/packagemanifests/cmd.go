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

	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest"
)

//nolint:maligned
type packagemanifestsCmd struct {
	// Common options.
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

	// These are set if a PROJECT config is not present.
	layout      string
	packageName string
	// Backend generator
	generator packagemanifest.Generator
}

// NewCmd returns the 'packagemanifests' command configured for the new project layout.
func NewCmd() *cobra.Command {
	c := &packagemanifestsCmd{}

	cmd := &cobra.Command{
		Use: "packagemanifests",
		Deprecated: "support for the packagemanifests format will be removed in operator-sdk v2.0.0. Use bundles " +
			"to package your operator instead. Migrate your packagemanifest to a bundle using " +
			"'operator-sdk pkgman-to-bundle' command. Run 'operator-sdk pkgman-to-bundle --help' " +
			"for more details.",
		Short:   "Generates package manifests data for the operator",
		Long:    longHelp,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			if err := c.setDefaults(); err != nil {
				return err
			}

			if err := c.validate(); err != nil {
				return fmt.Errorf("invalid command options: %v", err)
			}
			if err := c.run(); err != nil {
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
	fs.StringVar(&c.inputDir, "input-dir", defaultRootDir, "Directory to read existing package manifests from. "+
		"This directory is the parent of individual versioned package directories, and different from --deploy-dir")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory in which to write package manifests")
	fs.StringVar(&c.kustomizeDir, "kustomize-dir", filepath.Join("config", "manifests"),
		"Directory containing kustomize bases in a \"bases\" dir and a kustomization.yaml for operator-framework manifests")
	fs.StringVar(&c.deployDir, "deploy-dir", "", "Directory to read cluster-ready operator manifests from. "+
		"If --crds-dir is not set, CRDs are ready from this directory")
	fs.StringVar(&c.crdsDir, "crds-dir", "", "Directory to read cluster-ready CustomResoureDefinition manifests from. "+
		"This option can only be used if --deploy-dir is set")
	fs.StringVar(&c.channelName, "channel", "", "Channel name for the generated package")
	fs.BoolVar(&c.isDefaultChannel, "default-channel", false, "Use the channel passed to --channel "+
		"as the package manifest file's default channel")
	fs.BoolVar(&c.updateObjects, "update-objects", true, "Update non-CSV objects in this package, "+
		"ex. CustomResoureDefinitions, Roles")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	fs.BoolVar(&c.stdout, "stdout", false, "Write package to stdout")

	fs.StringVar(&c.packageName, "package", "", "Package name")
}

func (c packagemanifestsCmd) println(a ...interface{}) {
	if !(c.quiet || c.stdout) {
		fmt.Println(a...)
	}
}
