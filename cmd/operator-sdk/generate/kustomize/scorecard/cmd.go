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

package scorecard

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
)

const scorecardLongHelp = `
Running 'generate kustomize scorecard' will (re)generate scorecard configuration kustomize bases,
default test patches, and a kustomization.yaml in 'config/scorecard'.
`

const scorecardExamples = `
  $ operator-sdk generate kustomize scorecard
  Generating kustomize files in config/scorecard
  Kustomize files generated successfully
  $ tree ./config/scorecard
  ./config/scorecard/
  ├── bases
  │   └── config.yaml
  ├── kustomization.yaml
  └── patches
      ├── basic.config.yaml
      └── olm.config.yaml
`

// defaultTestImageTag points to the latest-released image.
// TODO: change the tag to "latest" once config scaffolding is in a release,
// as the new config spec won't work with the current latest image.
const defaultTestImageTag = "quay.io/operator-framework/scorecard-test:master"

type scorecardCmd struct {
	operatorName string
	outputDir    string
	testImageTag string
	quiet        bool
}

// NewCmd returns the `scorecard` subcommand.
func NewCmd() *cobra.Command {
	c := &scorecardCmd{}
	cmd := &cobra.Command{
		Use:     "scorecard",
		Short:   "Generates scorecard configuration files",
		Long:    scorecardLongHelp,
		Example: scorecardExamples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}

			cfg, err := kbutil.ReadConfig()
			if err != nil {
				return fmt.Errorf("error reading configuration: %v", err)
			}
			c.setDefaults(cfg)

			// Run command logic.
			if err = c.run(); err != nil {
				log.Fatalf("Error generating kustomize files: %v", err)
			}

			return nil
		},
	}

	c.addFlagsTo(cmd.Flags())

	return cmd
}

func (c *scorecardCmd) addFlagsTo(fs *pflag.FlagSet) {
	fs.StringVar(&c.operatorName, "operator-name", "", "Name of the operator")
	fs.StringVar(&c.outputDir, "output-dir", "", "Directory to write kustomize files")
	fs.StringVar(&c.testImageTag, "image", defaultTestImageTag,
		"Image to use for default tests; this image must contain the `/scorecard-test` binary")
	fs.BoolVarP(&c.quiet, "quiet", "q", false, "Run in quiet mode")
	// NB(estroz): might be nice to have an --overwrite flag to explicitly turn on overwrite behavior (the current default).
}

// defaultDir is the default directory in which to generate kustomize bases and the kustomization.yaml.
var defaultDir = filepath.Join("config", "scorecard")

// setDefaults sets command defaults.
func (c *scorecardCmd) setDefaults(cfg *config.Config) {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(cfg.Repo)
	}

	if c.outputDir == "" {
		c.outputDir = defaultDir
	}
}

// run scaffolds kustomize files for kustomizing a scorecard componentconfig.
func (c scorecardCmd) run() error {

	if !c.quiet {
		fmt.Println("Generating kustomize files in", c.outputDir)
	}

	err := generate(c.operatorName, c.testImageTag, c.outputDir)
	if err != nil {
		return err
	}

	if !c.quiet {
		fmt.Println("Kustomize files generated successfully")
	}

	return nil
}
