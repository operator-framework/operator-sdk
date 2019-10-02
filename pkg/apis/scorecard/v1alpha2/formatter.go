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

package v1alpha2

import (
	"encoding/json"
	"fmt"
	"github.com/mattn/go-isatty"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	failRequiredMessage = "A required test has failed."
	passRequiredMessage = "All required tests passed."
	redColor            = "31"
	greenColor          = "32"
	noColor             = "%s\n"
)

func (s ScorecardOutputList) MarshalText() (string, error) {
	var sb strings.Builder
	pluginOutputs := s.Items

	failColor := "\033[1;" + redColor + "m%s\033[0m\n"
	passColor := "\033[1;" + greenColor + "m%s\033[0m\n"

	failedRequiredTests := 0

	// turn off colorization if not in a terminal
	if !isatty.IsTerminal(os.Stdout.Fd()) &&
		!isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		passColor = noColor
		failColor = noColor
	}

	requiredTestStatus := fmt.Sprintf(passColor, passRequiredMessage)

	// calculate failed required tests and status
	for _, plugin := range pluginOutputs {
		for _, suite := range plugin.Results {
			for _, result := range suite.Tests {
				if result.State != PassState {
					failedRequiredTests++
					requiredTestStatus = fmt.Sprintf(failColor, failRequiredMessage)
				}
			}
		}
	}

	for _, plugin := range pluginOutputs {
		for _, suite := range plugin.Results {
			sb.WriteString(fmt.Sprintf("%s:\n", suite.Name))
			for _, result := range suite.Tests {
				sb.WriteString(fmt.Sprintf("\t%-35s: ", result.Name))

				if result.State == PassState {
					sb.WriteString(fmt.Sprintf(passColor, PassState))
				} else {
					sb.WriteString(fmt.Sprintf(failColor, FailState))
				}
			}
		}
	}

	sb.WriteString(fmt.Sprintf(requiredTestStatus))

	// TODO: We can probably use some helper functions to clean up these quadruple nested loops
	// Print suggestions
	for _, plugin := range pluginOutputs {
		for _, suite := range plugin.Results {
			for _, result := range suite.Tests {
				for _, suggestion := range result.Suggestions {
					// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
					sb.WriteString(fmt.Sprintf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion))
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
					sb.WriteString(fmt.Sprintf("\x1b[%dmERROR:\x1b[0m %s\n", 31, err))
				}
			}
		}
	}

	return sb.String(), nil
}

func (s ScorecardOutputList) MarshalJSONOutput(logReadWriter io.ReadWriter) ([]byte, error) {
	pluginOutputs := s.Items

	failedRequiredTests := 0

	requiredTestStatus := fmt.Sprintf(noColor, passRequiredMessage)

	// calculate failed required tests and status
	for _, plugin := range pluginOutputs {
		for _, suite := range plugin.Results {
			for _, result := range suite.Tests {
				if result.State != PassState {
					failedRequiredTests++
					requiredTestStatus = failRequiredMessage
				}
			}
		}
	}

	log, err := ioutil.ReadAll(logReadWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to read log buffer: %v", err)
	}
	scTest := combineScorecardOutput(pluginOutputs, string(log))

	scTest.FailedRequiredTests = failedRequiredTests
	scTest.RequiredTestStatus = requiredTestStatus

	// Pretty print so users can also read the json output
	bytes, err := json.MarshalIndent(scTest, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func combineScorecardOutput(outputs []ScorecardOutput, log string) ScorecardOutput {
	output := ScorecardOutput{
		Log: log,
	}
	for _, item := range outputs {
		output.Results = append(output.Results, item.Results...)
	}
	return output
}
