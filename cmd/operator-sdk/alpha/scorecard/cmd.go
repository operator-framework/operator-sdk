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
	"encoding/json"
	"fmt"

	"time"

	scorecard "github.com/operator-framework/operator-sdk/internal/scorecard/alpha"
	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
)

func NewCmd() *cobra.Command {
	var (
		config         string
		outputFormat   string
		bundle         string
		selector       string
		kubeconfig     string
		namespace      string
		serviceAccount string
		list           bool
		skipCleanup    bool
		waitTime       time.Duration
	)
	scorecardCmd := &cobra.Command{
		Use:    "scorecard",
		Short:  "Runs scorecard",
		Long:   `Has flags to configure dsl, bundle, and selector.`,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			var err error
			o := scorecard.Options{
				ServiceAccount: serviceAccount,
				Namespace:      namespace,
				BundlePath:     bundle,
				SkipCleanup:    skipCleanup,
				WaitTime:       waitTime,
			}
			o.Client, err = scorecard.GetKubeClient(kubeconfig)
			if err != nil {
				return fmt.Errorf("could not get Kube connection %s", err.Error())
			}
			o.Config, err = scorecard.LoadConfig(config)
			if err != nil {
				return fmt.Errorf("could not find config file %s", err.Error())
			}

			if bundle == "" {
				return fmt.Errorf("bundle flag required")
			}

			o.Selector, err = labels.Parse(selector)
			if err != nil {
				return fmt.Errorf("could not parse selector %s", err.Error())
			}

			var scorecardOutput v1alpha2.ScorecardOutput
			if list {
				scorecardOutput, err = scorecard.ListTests(o)
				if err != nil {
					return fmt.Errorf("error listing tests %s", err.Error())
				}
			} else {
				scorecardOutput, err = scorecard.RunTests(o)
				if err != nil {
					return fmt.Errorf("error running tests %s", err.Error())
				}
			}

			return printOutput(outputFormat, scorecardOutput)
		},
	}

	scorecardCmd.Flags().StringVarP(&config, "config", "c", "",
		"path to a new to be defined DSL yaml formatted file that configures what tests get executed")
	scorecardCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig path")

	scorecardCmd.Flags().StringVar(&bundle, "bundle", "", "path to the operator bundle contents on disk")
	scorecardCmd.Flags().StringVarP(&selector, "selector", "l", "", "label selector to determine which tests are run")
	scorecardCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace to run the test images in")
	scorecardCmd.Flags().StringVarP(&outputFormat, "output", "o", "text",
		"Output format for results.  Valid values: text, json")
	scorecardCmd.Flags().StringVarP(&serviceAccount, "service-account", "s", "default", "Service account to use for tests")
	scorecardCmd.Flags().BoolVarP(&list, "list", "L", false, "Option to enable listing which tests are run")
	scorecardCmd.Flags().BoolVarP(&skipCleanup, "skip-cleanup", "x", false, "Disable resource cleanup after tests are run")
	scorecardCmd.Flags().DurationVarP(&waitTime, "wait-time", "w", time.Duration(30*time.Second),
		"seconds to wait for tests to complete. Example: 35s")

	return scorecardCmd
}

func printOutput(outputFormat string, output v1alpha2.ScorecardOutput) error {
	switch outputFormat {
	case "text":
		o, err := output.MarshalText()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		fmt.Printf("%s\n", o)
	case "json":
		bytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		fmt.Printf("%s\n", string(bytes))
	default:
		return fmt.Errorf("invalid output format selected")
	}
	return nil

}
