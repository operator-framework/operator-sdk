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

// These functions should be in the public test definitions file, but they are not complete/stable,
// so we'll keep these here until they get fully implemented

// ResultsPassFail will be used when multiple CRs are supported
func ResultsPassFail(results []TestResult) (earned, max int) {
	for _, result := range results {
		if result.EarnedPoints != result.MaximumPoints {
			return 0, 1
		}
	}
	return 1, 1
}

// ResultsCumulative will be used when multiple CRs are supported
func ResultsCumulative(results []TestResult) (earned, max int) {
	for _, result := range results {
		earned += result.EarnedPoints
		max += result.MaximumPoints
	}
	return earned, max
}
