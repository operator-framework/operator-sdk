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
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"k8s.io/apimachinery/pkg/labels"
)

func TestFakeRunner(t *testing.T) {

	cases := []struct {
		configPathValue string
		selector        string
		wantError       bool
	}{
		{"testdata/bundle", "suite=basic", false},
	}

	for _, c := range cases {
		t.Run(c.configPathValue, func(t *testing.T) {
			o := Scorecard{}
			var err error
			o.Config, err = LoadConfig(c.configPathValue)
			o.Selector, err = labels.Parse(c.selector)
			o.SkipCleanup = true

			mockResult := v1alpha2.ScorecardTestResult{}
			mockResult.Name = "mocked test"
			mockResult.Description = "mocked test description"
			mockResult.State = v1alpha2.PassState
			mockResult.Errors = make([]string, 0)
			mockResult.Suggestions = make([]string, 0)

			r := FakePodTestRunner{}
			r.TestConfiguration = o
			r.TestResult = &mockResult
			o.TestRunner = r

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(7*time.Second))
			defer cancel()
			_, err = o.RunTests(ctx)

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
