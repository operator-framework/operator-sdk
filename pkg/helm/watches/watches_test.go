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

package watches

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

type testCase struct {
	name            string
	data            string
	env             map[string]string
	expectLen       int
	expectErr       bool
	expectOverrides []map[string]string
}

func TestLoadWatches(t *testing.T) {
	testCases := []testCase{
		{
			name: "valid",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    key: value
`,
			expectLen:       1,
			expectErr:       false,
			expectOverrides: []map[string]string{{"key": "value"}},
		},
		{
			name: "valid with override expansion",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    key: $MY_VALUE
`,
			env:             map[string]string{"MY_VALUE": "value"},
			expectLen:       1,
			expectErr:       false,
			expectOverrides: []map[string]string{{"key": "value"}},
		},
		{
			name: "multiple gvk",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyFirstKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
- group: mygroup
  version: v1alpha1
  kind: MySecondKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectLen: 2,
			expectErr: false,
		},
		{
			name: "duplicate gvk",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectLen: 0,
			expectErr: true,
		},
		{
			name: "no version",
			data: `---
- group: mygroup
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectLen: 0,
			expectErr: true,
		},
		{
			name: "no kind",
			data: `---
- group: mygroup
  version: v1alpha1
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectLen: 0,
			expectErr: true,
		},
		{
			name: "bad chart path",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: nonexistent/path/to/chart
`,
			expectLen: 0,
			expectErr: true,
		},
		{
			name: "invalid overrides",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  overrideValues:
    key1:
		key2: value
`,
			expectLen: 0,
			expectErr: true,
		},
		{
			name: "invalid yaml",
			data: `---
foo: bar
`,
			expectLen: 0,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmp, err := ioutil.TempFile("", "watches.yaml")
			if err != nil {
				t.Fatalf("Failed to create temporary watches.yaml file: %w", err)
			}
			defer func() { _ = os.Remove(tmp.Name()) }()
			if _, err := tmp.WriteString(tc.data); err != nil {
				t.Fatalf("Failed to write data to temporary watches.yaml file: %w", err)
			}
			if err := tmp.Close(); err != nil {
				t.Fatalf("Failed to close temporary watches.yaml file: %w", err)
			}

			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %q: %w", k, err)
				}
			}

			doTest(t, tc, tmp.Name())

			for k := range tc.env {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("Failed to unset environment variable %q: %w", k, err)
				}
			}
		})
	}
}

func doTest(t *testing.T, tc testCase, watchesFile string) {
	watches, err := Load(watchesFile)
	if !tc.expectErr && err != nil {
		t.Fatalf("Expected no error; got error: %w", err)
	} else if tc.expectErr && err == nil {
		t.Fatalf("Expected error; got no error")
	}
	if len(watches) != tc.expectLen {
		t.Fatalf("Expected %d watches; got %d", tc.expectLen, len(watches))
	}

	for i, w := range watches {
		if len(tc.expectOverrides) <= i {
			if len(w.OverrideValues) > 0 {
				t.Fatalf("Expected no overides; got %#v", w.OverrideValues)
			} else {
				continue
			}
		}

		expectedOverrides := tc.expectOverrides[i]
		if !reflect.DeepEqual(expectedOverrides, w.OverrideValues) {
			t.Fatalf("Expected overrides %#v; got %#v", expectedOverrides, w.OverrideValues)
		}
	}
}
