// Copyright 2018 The Operator-SDK Authors
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

package scorecard2

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	dsl         string
	bundle      string
	bundleImage string
	selector    string
)

//scorecard2 alpha command stub
func NewCmd() *cobra.Command {
	scorecardCmd := &cobra.Command{
		Use:   "scorecard",
		Short: "Runs scorecard",
		Long:  `Has flags to configure dsl, bundle, bundle-image, and selector.`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("TODO")
		},
	}

	scorecardCmd.Flags().StringVar(&dsl, "dsl", "", "path to a new to be defined DSL yaml formatted file that configures what tests get executed")
	scorecardCmd.Flags().StringVar(&bundle, "bundle", "", "path to the operator bundle contents on disk")
	scorecardCmd.Flags().StringVar(&bundleImage, "bundle-image", "", "name of a bundle image not on disk but in a registry")
	scorecardCmd.Flags().StringVar(&selector, "selector", "", "label selector to determine which tests are run")

	return scorecardCmd
}
