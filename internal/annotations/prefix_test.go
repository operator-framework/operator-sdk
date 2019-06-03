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
		prefix    string
		result    []string
		wantError bool
	}{
		{"", nil, true},
		{"+operator-sdk", nil, true},
		{"+operator-sdk:", nil, true},
		{"+operator-sdk:foo", []string{"foo"}, false},
		{"+operator-sdk:foo:bar", []string{"foo", "bar"}, false},
	}

	for i, c := range cases {
		got, err := SplitPrefix(c.prefix)
		if err != nil && !c.wantError {
			t.Errorf("Case %d: wanted result %+q, got error: %v", i, c.result, err)
		} else if err == nil && c.wantError {
			t.Errorf("Case %d: wanted error, got result %+q", i, got)
		}
	}
}

func TestSplitPath(t *testing.T) {
	cases := []struct {
		path      string
		result    []string
		wantError bool
	}{
		{"", nil, true},
		{"+operator-sdk", nil, true},
		{"+operator-sdk:foo", nil, true},
		{"+operator-sdk:foo:bar.", nil, true},
		{"+operator-sdk:foo:bar.baz", []string{"+operator-sdk:foo:bar", "baz"}, false},
	}

	for i, c := range cases {
		got, err := SplitPath(c.path)
		if err != nil && !c.wantError {
			t.Errorf("Case %d: wanted result %+q, got error: %v", i, c.result, err)
		} else if err == nil && c.wantError {
			t.Errorf("Case %d: wanted error, got result %+q", i, got)
		}
	}
}

func TestSplitAnnotation(t *testing.T) {
	cases := []struct {
		annotation string
		path, val  string
		wantError  bool
	}{
		{"", "", "", true},
		{"+operator-sdk", "", "", true},
		{"+operator-sdk:foo:bar.baz=", "", "", true},
		{"+operator-sdk:foo:bar.baz=value", "+operator-sdk:foo:bar.baz", "value", false},
	}

	for i, c := range cases {
		gotPath, gotVal, err := SplitAnnotation(c.annotation)
		if err != nil && !c.wantError {
			t.Errorf("Case %d: wanted path %s and val %s, got error: %v", i, c.path, c.val, err)
		} else if err == nil && c.wantError {
			t.Errorf("Case %d: wanted error, got path %s and val %s", i, gotPath, gotVal)
		}
	}
}
