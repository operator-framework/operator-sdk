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
	"errors"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
)

// TODO(joelanford): rewrite to use ginkgo/gomega
func TestRunTests(t *testing.T) {
	cases := []struct {
		name            string
		configPathValue string
		selector        string
		wantError       bool
		testRunner      FakeTestRunner
		expectedState   v1alpha3.State
	}{
		{
			name:            "should execute 1 fake test",
			configPathValue: "testdata/bundle",
			selector:        "suite=basic",
			wantError:       false,
			testRunner:      FakeTestRunner{},
			expectedState:   v1alpha3.PassState,
		},
		{
			name:            "should execute 1 fake test",
			configPathValue: "testdata/bundle",
			selector:        "suite=basic",
			wantError:       false,
			testRunner:      FakeTestRunner{},
			expectedState:   v1alpha3.PassState,
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

			mockResult := v1alpha3.TestResult{}
			mockResult.Name = "mocked test"
			mockResult.State = v1alpha3.PassState
			mockResult.Errors = make([]string, 0)
			mockResult.Suggestions = make([]string, 0)
			mockStatus := v1alpha3.TestStatus{Results: []v1alpha3.TestResult{mockResult}}

			c.testRunner.TestStatus = &mockStatus
			o.TestRunner = c.testRunner

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(7*time.Second))
			defer cancel()
			var scorecardOutput v1alpha3.TestList
			scorecardOutput, err = o.Run(ctx)

			if scorecardOutput.Items[0].Status.Results[0].State != c.expectedState {
				t.Fatalf("Wanted state %v, got %v", c.expectedState, scorecardOutput.Items[0].Status.Results[0].State)
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

// TODO(joelanford): rewrite to use ginkgo/gomega
func TestRunParallelPass(t *testing.T) {
	scorecard := getFakeScorecard(true)
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Millisecond)
	defer cancel()

	tests, err := scorecard.Run(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got error: %v", err)
	}
	if len(tests.Items) != 2 {
		t.Fatalf("Expected 2 tests, got %d", len(tests.Items))
	}
	for _, test := range tests.Items {
		expectPass(t, test)
	}
}

// TODO(joelanford): rewrite to use ginkgo/gomega
func TestRunSequentialPass(t *testing.T) {
	scorecard := getFakeScorecard(false)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Millisecond)
	defer cancel()

	tests, err := scorecard.Run(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got error: %v", err)
	}
	if len(tests.Items) != 2 {
		t.Fatalf("Expected 2 tests, got %d", len(tests.Items))
	}
	for _, test := range tests.Items {
		expectPass(t, test)
	}
}

// TODO(joelanford): rewrite to use ginkgo/gomega
func TestRunSequentialFail(t *testing.T) {
	scorecard := getFakeScorecard(false)

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Millisecond)
	defer cancel()

	_, err := scorecard.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Expected deadline exceeded error, got:  %v", err)
	}
}

func getFakeScorecard(parallel bool) Scorecard {
	return Scorecard{
		Config: Config{
			Stages: []Stage{
				{
					Parallel: parallel,
					Tests: []Test{
						{},
						{},
					},
				},
			},
		},
		TestRunner: FakeTestRunner{
			Sleep: 5 * time.Millisecond,
			TestStatus: &v1alpha3.TestStatus{
				Results: []v1alpha3.TestResult{
					{
						State: v1alpha3.PassState,
					},
				},
			},
		},
	}
}

func expectPass(t *testing.T, test v1alpha3.Test) {
	if len(test.Status.Results) != 1 {
		t.Fatalf("Expected 1 results, got %d", len(test.Status.Results))
	}
	for _, r := range test.Status.Results {
		if len(r.Errors) > 0 {
			t.Fatalf("Expected no errors, got %v", r.Errors)
		}
		if r.State != v1alpha3.PassState {
			t.Fatalf("Expected result state %q, got %q", v1alpha3.PassState, r.State)
		}
	}
}
