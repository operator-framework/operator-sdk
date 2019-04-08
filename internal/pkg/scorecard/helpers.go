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

import "fmt"

// ResultsPassFail will be used when multiple CRs are supported
func ResultsPassFail(results []*TestResult) (*TestResult, error) {
	var name string
	finalResult := TestResult{}
	if len(results) > 0 {
		name = results[0].Test.GetName()
		// all results have the same test
		finalResult.Test = results[0].Test
		finalResult.MaximumPoints = 1
		finalResult.EarnedPoints = 1
	}
	for _, result := range results {
		if result.Test.IsCumulative() {
			return nil, fmt.Errorf("cumulative test passed to ResultsPassFail: name (%s)", result.Test.GetName())
		}
		if result.Test.GetName() != name {
			return nil, fmt.Errorf("test name mismatch in ResultsPassFail: %s != %s", result.Test.GetName(), name)
		}
		if result.EarnedPoints != result.MaximumPoints {
			finalResult.EarnedPoints = 0
		}
		finalResult.Suggestions = append(finalResult.Suggestions, result.Suggestions...)
		finalResult.Errors = append(finalResult.Errors, result.Errors...)
	}
	return &finalResult, nil
}

// ResultsCumulative will be used when multiple CRs are supported
func ResultsCumulative(results []*TestResult) (*TestResult, error) {
	var name string
	finalResult := TestResult{}
	if len(results) > 0 {
		name = results[0].Test.GetName()
		// all results have the same test
		finalResult.Test = results[0].Test
	}
	for _, result := range results {
		if !result.Test.IsCumulative() {
			return nil, fmt.Errorf("non-cumulative test passed to ResultsCumulative: name (%s)", result.Test.GetName())
		}
		if result.Test.GetName() != name {
			return nil, fmt.Errorf("test name mismatch in ResultsCumulative: %s != %s", result.Test.GetName(), name)
		}
		finalResult.EarnedPoints += result.EarnedPoints
		finalResult.MaximumPoints += result.MaximumPoints
		finalResult.Suggestions = append(finalResult.Suggestions, result.Suggestions...)
		finalResult.Errors = append(finalResult.Errors, result.Errors...)
	}
	return &finalResult, nil
}
