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
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
	"strings"
)

const (
	redColor   = "31"
	greenColor = "32"
	noColor    = "%s\n"
)

func (s ScorecardOutput) MarshalText() (string, error) {
	var sb strings.Builder

	failColor := "\033[1;" + redColor + "m%s\033[0m\n"
	passColor := "\033[1;" + greenColor + "m%s\033[0m\n"

	// turn off colorization if not in a terminal
	if !isatty.IsTerminal(os.Stdout.Fd()) &&
		!isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		passColor = noColor
		failColor = noColor
	}

	var currentSuite string
	for _, result := range s.Results {
		if currentSuite != result.Labels["suite"] {
			sb.WriteString(fmt.Sprintf("%s:\n", result.Labels["suite"]))
			currentSuite = result.Labels["suite"]
		}
		sb.WriteString(fmt.Sprintf("\t%-35s: ", result.Name))

		if result.State == PassState {
			sb.WriteString(fmt.Sprintf(passColor, PassState))
		} else {
			sb.WriteString(fmt.Sprintf(failColor, FailState))
		}
	}

	for _, result := range s.Results {
		for _, suggestion := range result.Suggestions {
			// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
			sb.WriteString(fmt.Sprintf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion))
		}
	}

	for _, result := range s.Results {
		for _, err := range result.Errors {
			// 31 is red (specifically, the same shade of red that logrus uses for errors)
			sb.WriteString(fmt.Sprintf("\x1b[%dmERROR:\x1b[0m %s\n", 31, err))
		}
	}

	return sb.String(), nil
}
