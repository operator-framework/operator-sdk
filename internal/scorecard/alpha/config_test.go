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
	"testing"
)

func TestInvalidConfigPath(t *testing.T) {

	cases := []struct {
		configPathValue string
		wantError       bool
	}{
		{"", true},
	}

	for _, c := range cases {
		t.Run(c.configPathValue, func(t *testing.T) {
			_, err := LoadConfig(c.configPathValue)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			} else if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}

		})

	}
}
