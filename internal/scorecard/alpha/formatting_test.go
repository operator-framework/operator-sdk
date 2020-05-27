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
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"k8s.io/apimachinery/pkg/labels"
)

func TestList(t *testing.T) {

	cases := []struct {
		bundlePathValue string
		selector        string
		wantError       bool
		resultCount     int
	}{
		{"testdata/bundle", "suite=basic", false, 1},
	}

	for _, c := range cases {
		t.Run(c.bundlePathValue, func(t *testing.T) {
			o := Scorecard{}
			runner := PodTestRunner{}
			var err error
			configPath := filepath.Join(c.bundlePathValue, "tests", "scorecard", "config.yaml")

			o.Config, err = LoadConfig(configPath)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}
			o.Selector, err = labels.Parse(c.selector)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}
			runner.BundlePath = c.bundlePathValue
			o.TestRunner = &runner
			var output v1alpha2.ScorecardOutput
			output, err = o.ListTests()
			if err == nil && c.wantError {
				t.Fatalf("Wanted error but got no error")
			} else if err != nil {
				if !c.wantError {
					t.Fatalf("Wanted result but got error: %v", err)
				}
				return
			}
			actualResultCount := len(output.Results)
			if c.resultCount != actualResultCount {
				t.Fatalf("Wanted result count %d but got : %d", c.resultCount, actualResultCount)
			}

		})

	}
}
