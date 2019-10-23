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

package schelpers

import (
	"testing"
)

func TestValidateVersions(t *testing.T) {
	cases := []struct {
		name      string
		version   string
		result    []string
		wantError bool
	}{
		{"empty", "", nil, true},
		{"invalidVersion", "invalidVersion", nil, true},
		{"v1alpha1", v1alpha1, nil, false},
		{"v1alpha2", v1alpha2, nil, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateVersion(c.version)
			if err != nil && !c.wantError {
				t.Errorf("Wanted result %+q, got error: %v", c.result, err)
			} else if err == nil && c.wantError {
				t.Errorf("Wanted error, got nil")
			}
		})
	}
}
