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

	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"
)

type packagemanifestsCmd struct {
	olmoperator.PackageManifestsCmd
}

func NewCmd() *cobra.Command {
	c := &packagemanifestsCmd{}

	cmd := &cobra.Command{
		Use:   "packagemanifests",
		Short: "Deploy an Operator in the package manifests format with OLM",
		Long: `'run packagemanifests' deploys an Operator's package manifests with OLM. The command's argument
must be set to a valid package manifests root directory, ex. '<project-root>/packagemanifests'.`,
		Aliases: []string{"pm"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := c.validate(args)
			if err != nil {
				log.Fatalf("Failed to validate input: %v", err)
			}
			c.setDefaults(args)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Infof("Running operator from directory %s", c.ManifestsDir)

			if err := c.Run(); err != nil {
				log.Fatalf("Failed to run operator: %v", err)
			}
			return nil
		},
	}

	c.PackageManifestsCmd.AddToFlagSet(cmd.Flags())

	return cmd
}

func (c *packagemanifestsCmd) validate(args []string) error {
	if len(args) > 0 {
		if len(args) > 1 {
			return fmt.Errorf("exactly one argument is required")
		}
	}

	return nil
}

func (c *packagemanifestsCmd) setDefaults(args []string) {
	if len(args) != 0 {
		c.ManifestsDir = args[0]
	} else {
		c.ManifestsDir = "packagemanifests"
	}
}
