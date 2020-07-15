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

package v1alpha3

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

const (
	redColor    = "31"
	greenColor  = "32"
	yellowColor = "33"
	noColor     = "%s\n"
)

func (s Test) MarshalText() string {
	var sb strings.Builder

	failColor := "\033[1;" + redColor + "m%s\033[0m"
	passColor := "\033[1;" + greenColor + "m%s\033[0m"
	warnColor := "\033[1;" + yellowColor + "m%s\033[0m"

	// turn off colorization if not in a terminal
	if !isatty.IsTerminal(os.Stdout.Fd()) &&
		!isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		passColor = noColor
		failColor = noColor
		warnColor = noColor
	}

	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("-", 80)))
	sb.WriteString(fmt.Sprintf("Image:      %s\n", s.Spec.Image))

	if len(s.Spec.Entrypoint) > 0 {
		sb.WriteString(fmt.Sprintf("Entrypoint: %s\n", s.Spec.Entrypoint))
	}

	if len(s.Spec.Labels) > 0 {
		sb.WriteString("Labels:\n")
		for labelKey, labelValue := range s.Spec.Labels {
			sb.WriteString(fmt.Sprintf("\t%q:%q\n", labelKey, labelValue))
		}
	}
	if len(s.Status.Results) > 0 {
		sb.WriteString("Results:\n")
		for _, result := range s.Status.Results {
			if len(result.Name) > 0 {
				sb.WriteString(fmt.Sprintf("\tName: %s\n", result.Name))
			}
			sb.WriteString("\tState: ")
			if result.State == PassState {
				sb.WriteString(fmt.Sprintf(passColor, PassState))
			} else if result.State == FailState {
				sb.WriteString(fmt.Sprintf(failColor, FailState))
			} else if result.State == ErrorState {
				sb.WriteString(fmt.Sprintf(failColor, ErrorState))
			} else {
				sb.WriteString("unknown")
			}
			sb.WriteString("\n")

			if len(result.Suggestions) > 0 {
				sb.WriteString(fmt.Sprintf(warnColor, "\tSuggestions:\n"))
				for _, suggestion := range result.Suggestions {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", suggestion))
				}
			}

			if len(result.Errors) > 0 {
				sb.WriteString(fmt.Sprintf(failColor, "\tErrors:\n"))
				for _, err := range result.Errors {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", err))
				}
			}

			if result.Log != "" {
				sb.WriteString("\tLog:\n")
				scanner := bufio.NewScanner(strings.NewReader(result.Log))
				for scanner.Scan() {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", scanner.Text()))
				}
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
