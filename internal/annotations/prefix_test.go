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

package annotations

import (
	"testing"
)

func TestSplitPrefix(t *testing.T) {
	cases := []struct {
		name      string
		prefix    string
		result    []string
		wantError bool
	}{
		{"empty", "", nil, true},
		{"no prefix separator or use case prefix", "+operator-sdk", nil, true},
		{"no use case prefix", "+operator-sdk:", nil, true},
		{"use case prefix", "+operator-sdk:foo", []string{"foo"}, false},
		{"use case prefix and one path token", "+operator-sdk:foo:bar", []string{"foo", "bar"}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SplitPrefix(c.prefix)
			if err != nil && !c.wantError {
				t.Errorf("Wanted result %+q, got error: %v", c.result, err)
			} else if err == nil && c.wantError {
				t.Errorf("Wanted error, got result %+q", got)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		result    []string
		wantError bool
	}{
		{"empty", "", nil, true},
		{"no prefix separator or use case prefix", "+operator-sdk", nil, true},
		{"use case prefix", "+operator-sdk:foo", nil, true},
		{"use case prefix and path element with empty child path element", "+operator-sdk:foo:bar.", nil, true},
		{"use case prefix and path elements", "+operator-sdk:foo:bar.baz", []string{"+operator-sdk:foo:bar", "baz"}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SplitPath(c.path)
			if err != nil && !c.wantError {
				t.Errorf("Wanted result %+q, got error: %v", c.result, err)
			} else if err == nil && c.wantError {
				t.Errorf("Wanted error, got result %+q", got)
			}
		})
	}
}

func TestSplitAnnotation(t *testing.T) {
	cases := []struct {
		name       string
		annotation string
		path, val  string
		wantError  bool
	}{
		{"empty", "", "", "", true},
		{"no prefix separator or use case prefix", "+operator-sdk", "", "", true},
		{"prefixed path with empty value", "+operator-sdk:foo:bar.baz=", "", "", true},
		{"prefixed path with value", "+operator-sdk:foo:bar.baz=value", "+operator-sdk:foo:bar.baz", "value", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPath, gotVal, err := SplitAnnotation(c.annotation)
			if err != nil && !c.wantError {
				t.Errorf("Wanted path %s and val %s, got error: %v", c.path, c.val, err)
			} else if err == nil && c.wantError {
				t.Errorf("Wanted error, got path %s and val %s", gotPath, gotVal)
			}
		})
	}
}
