// Copyright 2019 The Operator-SDK Authors
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
	"io/ioutil"

	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
)

const (
	failRequiredMessage = "A required test has failed."
	passRequiredMessage = "All required tests passed."
	failColor           = "\033[1;31m%s\033[0m\n"
	passColor           = "\033[1;32m%s\033[0m\n"
)

func printPluginOutputs(pluginOutputs []scapiv1alpha1.ScorecardOutput) error {

	totalScore := 0.0
	failedRequiredTests := 0
	requiredTestStatus := fmt.Sprintf(passColor, passRequiredMessage)

	// calculate failed required tests and status
	for _, plugin := range pluginOutputs {
		for _, suite := range plugin.Results {
			for _, result := range suite.Tests {
				if schelpers.IsV1alpha2() {
					if result.State != scapiv1alpha1.PassState {
						failedRequiredTests++
						requiredTestStatus = fmt.Sprintf(failColor, failRequiredMessage)
					}
				}
			}
		}
	}

	// produce text output
	if scViper.GetString(OutputFormatOpt) == TextOutputFormat {
		numSuites := 0
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				fmt.Printf("%s:\n", suite.Name)
				for _, result := range suite.Tests {
					if schelpers.IsV1alpha2() {
						fmt.Printf("\t%-35s: ", result.Name)

						if result.State == scapiv1alpha1.PassState {
							fmt.Printf(passColor, scapiv1alpha1.PassState)
						} else {
							fmt.Printf(failColor, scapiv1alpha1.FailState)
						}
						continue
					}

					// v1alpha1 case
					fmt.Printf("\t%s: %d/%d\n", result.Name, result.EarnedPoints, result.MaximumPoints)
				}
				totalScore += float64(suite.TotalScore)
				numSuites++
			}
		}

		if schelpers.IsV1alpha2() {
			fmt.Printf(requiredTestStatus)
		} else {
			totalScore = totalScore / float64(numSuites)
			fmt.Printf("\nTotal Score: %.0f%%\n", totalScore)
		}

		// TODO: We can probably use some helper functions to clean up these quadruple nested loops
		// Print suggestions
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				for _, result := range suite.Tests {
					for _, suggestion := range result.Suggestions {
						// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
						fmt.Printf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion)
					}
				}
			}
		}
		// Print errors
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				for _, result := range suite.Tests {
					for _, err := range result.Errors {
						// 31 is red (specifically, the same shade of red that logrus uses for errors)
						fmt.Printf("\x1b[%dmERROR:\x1b[0m %s\n", 31, err)
					}
				}
			}
		}
	}

	// produce json output
	if scViper.GetString(OutputFormatOpt) == JSONOutputFormat {
		log, err := ioutil.ReadAll(logReadWriter)
		if err != nil {
			return fmt.Errorf("failed to read log buffer: %v", err)
		}
		scTest := schelpers.CombineScorecardOutput(pluginOutputs, string(log))
		if schelpers.IsV1alpha2() {
			scV2Test := schelpers.ConvertScorecardOutputV1ToV2(scTest)
			scV2Test.FailedRequiredTests = failedRequiredTests
			scV2Test.RequiredTestStatus = requiredTestStatus
			bytes, err := json.MarshalIndent(scV2Test, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", string(bytes))
			return nil
		}

		// Pretty print so users can also read the json output
		bytes, err := json.MarshalIndent(scTest, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
	}
	return nil
}
