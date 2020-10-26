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
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/labels"
)

func TestList(t *testing.T) {

	cases := []struct {
		bundlePathValue string
		selector        string
		resultCount     int
	}{
		{"testdata/bundle", "suite=basic", 1},
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
			output := o.List()
			actualResultCount := len(output.Items)
			if c.resultCount != actualResultCount {
				t.Fatalf("Wanted result count %d but got : %d", c.resultCount, actualResultCount)
			}

		})

	}
}
