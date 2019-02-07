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

import "testing"

// TestSuiteWeightsCheck makes sure that the combined weights of all
// the tests in a suite adds up to 100
func TestSuiteWeightsCheck(t *testing.T) {
	basicTests := NewBasicTestSuite(BasicTestConfig{})
	basicWeight := 0
	for _, weight := range basicTests.Weights {
		basicWeight += weight
	}
	if basicWeight != 100 {
		t.Errorf("Weights of Basic Tests add to %d", basicWeight)
	}
	olmTests := NewOLMTestSuite(OLMTestConfig{})
	OLMWeight := 0
	for _, weight := range olmTests.Weights {
		OLMWeight += weight
	}
	if OLMWeight != 100 {
		t.Errorf("Weights of OLM Tests add to %d", OLMWeight)
	}
}
