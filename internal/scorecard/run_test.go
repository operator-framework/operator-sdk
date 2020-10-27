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

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"k8s.io/apimachinery/pkg/labels"
)

// TODO(joelanford): rewrite to use ginkgo/gomega
func TestRun(t *testing.T) {
	cases := []struct {
		name            string
		configPathValue string
		selector        string
		timeout         time.Duration
		wantedError     error
		testRunner      FakeTestRunner
		expectedState   v1alpha3.State
	}{
		{
			name:            "should execute 1 fake test successfully",
			configPathValue: "testdata/bundle",
			selector:        "suite=basic",
			timeout:         time.Second * 7,
			testRunner:      FakeTestRunner{},
			expectedState:   v1alpha3.PassState,
		},
		{
			name:            "should fail to execute 1 test with short timeout",
			configPathValue: "testdata/bundle",
			selector:        "suite=basic",
			timeout:         time.Second * 0,
			wantedError:     context.DeadlineExceeded,
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

			ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
			defer cancel()

			scorecardOutput, err := o.Run(ctx)
			if err == nil {
				if c.wantedError != nil {
					t.Errorf("Wanted error %s but got no error", c.wantedError)
					return
				}
				if scorecardOutput.Items[0].Status.Results[0].State != c.expectedState {
					t.Errorf("Wanted state %v, got %v", c.expectedState, scorecardOutput.Items[0].Status.Results[0].State)
				}
			} else if err != nil {
				if c.wantedError == nil {
					t.Errorf("Wanted result but got error %v", err)
				} else if !errors.Is(err, c.wantedError) {
					t.Errorf("Wanted error %v but got error %v", c.wantedError, err)
				}
			}
		})

	}
}

// TODO(joelanford): rewrite to use ginkgo/gomega
func TestRunParallelPass(t *testing.T) {
	scorecard := getFakeScorecard(true)
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
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
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
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

	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()

	_, err := scorecard.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Expected deadline exceeded error, got:  %v", err)
	}
}

func getFakeScorecard(parallel bool) Scorecard {
	return Scorecard{
		Config: v1alpha3.Configuration{
			Stages: []v1alpha3.StageConfiguration{
				{
					Parallel: parallel,
					Tests: []v1alpha3.TestConfiguration{
						{},
						{},
					},
				},
			},
		},
		TestRunner: FakeTestRunner{
			Sleep: 50 * time.Millisecond,
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
