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

package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFuncs(t *testing.T) {
	cases := []struct {
		name           string
		paths          []string
		wantedSatPaths map[string]bool
	}{
		{
			"empty filters with one path",
			nil,
			map[string]bool{"notexist": false},
		},
		{
			"two filters with no matching paths",
			[]string{"key1", "key2"},
			map[string]bool{"notexist": false},
		},
		{
			"multiple pathed filters with multiple matching paths",
			[]string{"key1/key2", "key3", "/abs/path/to/something"},
			map[string]bool{
				"key3":                        true,
				"/abs/path/to/something/else": true,
				"key2":                        false,
				"path/to":                     false,
			},
		},
	}
	for _, c := range cases {
		filters := MakeFilters(c.paths...)
		for path, wantedSat := range c.wantedSatPaths {
			t.Run(c.name+": "+path, func(t *testing.T) {
				isSat := filters.SatisfiesAny(path)
				if wantedSat {
					assert.True(t, isSat)
				} else {
					assert.False(t, isSat)
				}
			})
		}
	}
}
