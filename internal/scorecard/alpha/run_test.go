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

package alpha

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"k8s.io/apimachinery/pkg/labels"
)

func TestRunTests(t *testing.T) {

	cases := []struct {
		name            string
		configPathValue string
		selector        string
		wantError       bool
		testRunner      FakeTestRunner
		expectedState   v1alpha2.State
	}{
		{
			name:            "should execute 1 fake test",
			configPathValue: "testdata/bundle",
			selector:        "suite=basic",
			wantError:       false,
			testRunner:      FakeTestRunner{},
			expectedState:   v1alpha2.PassState,
		},
		{
			name:            "should execute 1 fake test",
			configPathValue: "testdata/bundle",
			selector:        "suite=basic",
			wantError:       false,
			testRunner:      FakeTestRunner{},
			expectedState:   v1alpha2.PassState,
		},
	}

	for _, c := range cases {
		t.Run(c.configPathValue, func(t *testing.T) {
			o := Scorecard{}
			var err error
			configPath := filepath.Join(c.configPathValue, "tests", "scorecard", "config.yaml")
			o.Config, err = LoadConfig(configPath)
			if err != nil {
				t.Fatalf("Unexpected error loading config %v", err)
			}
			o.Selector, err = labels.Parse(c.selector)
			if err != nil {
				t.Fatalf("Unexpected error parsing selector %v", err)
			}
			o.SkipCleanup = true

			mockResult := v1alpha2.ScorecardTestResult{}
			mockResult.Name = "mocked test"
			mockResult.Description = "mocked test description"
			mockResult.State = v1alpha2.PassState
			mockResult.Errors = make([]string, 0)
			mockResult.Suggestions = make([]string, 0)

			c.testRunner.TestResult = &mockResult
			o.TestRunner = c.testRunner

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(7*time.Second))
			defer cancel()
			var scorecardOutput v1alpha2.ScorecardOutput
			scorecardOutput, err = o.RunTests(ctx)

			if scorecardOutput.Results[0].State != c.expectedState {
				t.Fatalf("Wanted state %v, got %v", c.expectedState, scorecardOutput.Results[0].State)
			}

			if err == nil && c.wantError {
				t.Fatalf("Wanted error but got no error")
			} else if err != nil {
				if !c.wantError {
					t.Fatalf("Wanted result but got error: %v", err)
				}
				return
			}

		})

	}
}
