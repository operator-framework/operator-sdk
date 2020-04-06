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
	"log"

	scorecard "github.com/operator-framework/operator-sdk/internal/scorecard/alpha"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
)

func NewCmd() *cobra.Command {
	var (
		config string
		//bundle   string // TODO - to be implemented
		selector string
		//list     bool // TODO - to be implemented
	)
	scorecardCmd := &cobra.Command{
		Use:    "scorecard",
		Short:  "Runs scorecard",
		Long:   `Has flags to configure dsl, bundle, and selector.`,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			var err error
			o := scorecard.Options{}
			o.Config, err = scorecard.LoadConfig(config)
			if err != nil {
				return fmt.Errorf("could not find config file %s", err.Error())
			}

			o.Selector, err = labels.Parse(selector)
			if err != nil {
				return fmt.Errorf("could not parse selector %s", err.Error())
			}

			// TODO - process list and output formatting here?

			if err := scorecard.RunTests(o); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	scorecardCmd.Flags().StringVarP(&config, "config", "c", "",
		"path to a new to be defined DSL yaml formatted file that configures what tests get executed")
	//	scorecardCmd.Flags().StringVar(&bundle, "bundle", "", "path to the operator bundle contents on disk")
	scorecardCmd.Flags().StringVarP(&selector, "selector", "l", "", "label selector to determine which tests are run")
	//	scorecardCmd.Flags().BoolVarP(&list, "list", "L", false, "option to enable listing which tests are run")

	return scorecardCmd
}
