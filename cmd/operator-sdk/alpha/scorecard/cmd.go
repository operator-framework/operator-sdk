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
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"time"

	scorecard "github.com/operator-framework/operator-sdk/internal/scorecard/alpha"
	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
)

func NewCmd() *cobra.Command {
	var (
		outputFormat   string
		bundle         string
		config         string
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
			o := scorecard.Scorecard{
				SkipCleanup: skipCleanup,
			}

			if bundle == "" {
				return fmt.Errorf("bundle flag required")
			}

			runner := scorecard.PodTestRunner{
				ServiceAccount: serviceAccount,
				Namespace:      namespace,
				BundlePath:     bundle,
			}

			runner.Client, err = scorecard.GetKubeClient(kubeconfig)
			if err != nil {
				return fmt.Errorf("could not get kubernetes client: %w", err)
			}

			configPath := filepath.Join(bundle, "tests", "scorecard", "config.yaml")
			if config != "" {
				configPath = config
			}
			o.Config, err = scorecard.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("could not find config file %w", err)
			}

			o.Selector, err = labels.Parse(selector)
			if err != nil {
				return fmt.Errorf("could not parse selector %w", err)
			}

			var scorecardOutput v1alpha2.ScorecardOutput
			if list {
				scorecardOutput, err = o.ListTests()
				if err != nil {
					return fmt.Errorf("error listing tests %w", err)
				}
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), waitTime)
				defer cancel()

				o.TestRunner = &runner

				scorecardOutput, err = o.RunTests(ctx)
				if err != nil {
					return fmt.Errorf("error running tests %w", err)
				}

			}

			return printOutput(outputFormat, scorecardOutput)
		},
	}

	scorecardCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig path")

	scorecardCmd.Flags().StringVar(&bundle, "bundle", "", "path to the operator bundle contents on disk")
	scorecardCmd.Flags().StringVarP(&selector, "selector", "l", "", "label selector to determine which tests are run")
	scorecardCmd.Flags().StringVarP(&config, "config", "c", "", "path to scorecard config file")
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
		if len(output.Results) == 0 {
			fmt.Println("0 tests selected")
			return nil
		}
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
