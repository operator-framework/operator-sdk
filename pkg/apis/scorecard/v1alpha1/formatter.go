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

package v1alpha1

import (
	"fmt"
	"strings"
)

func (s ScorecardOutput) MarshalText() (string, error) {
	var sb strings.Builder
	totalScore := 0.0

	numSuites := 0
	for _, suite := range s.Results {
		sb.WriteString(fmt.Sprintf("%s:\n", suite.Name))
		for _, result := range suite.Tests {

			sb.WriteString(fmt.Sprintf("\t%s: %d/%d\n", result.Name, result.EarnedPoints, result.MaximumPoints))
		}
		totalScore += float64(suite.TotalScore)
		numSuites++
	}

	totalScore = totalScore / float64(numSuites)
	sb.WriteString(fmt.Sprintf("\nTotal Score: %.0f%%\n", totalScore))

	// TODO: We can probably use some helper functions to clean up these quadruple nested loops
	// Print suggestions
	for _, suite := range s.Results {
		for _, result := range suite.Tests {
			for _, suggestion := range result.Suggestions {
				// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
				sb.WriteString(fmt.Sprintf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion))
			}
		}
	}

	// Print errors
	for _, suite := range s.Results {
		for _, result := range suite.Tests {
			for _, err := range result.Errors {
				// 31 is red (specifically, the same shade of red that logrus uses for errors)
				sb.WriteString(fmt.Sprintf("\x1b[%dmERROR:\x1b[0m %s\n", 31, err))
			}
		}
	}

	return sb.String(), nil
}
