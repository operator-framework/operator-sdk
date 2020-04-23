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

package scorecardutil

import (
	"testing"
)

func TestBundlePath(t *testing.T) {
	cases := []struct {
		bundlePath string
		wantError  bool
	}{
		{"../testdata/bundle", false},
		{"/foo", true},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {
			_, err := LoadBundleDirectory(c.bundlePath)
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
func TestBundleCRs(t *testing.T) {
	cases := []struct {
		bundlePath string
		crCount    int
	}{
		{"../testdata/bundle", 1},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {
			bundle, err := LoadBundleDirectory(c.bundlePath)
			if err != nil {
				t.Fatal(err)
			}

			examples, err := GetALMExamples(*bundle)
			if err != nil {
				t.Fatal(err)
			}

			if len(examples) != c.crCount {
				t.Errorf("Wanted %d CRs but got: %d", c.crCount, len(examples))
				return
			}

		})

	}
}
