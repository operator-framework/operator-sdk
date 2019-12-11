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
	"testing"
)

type testCase struct {
	name      string
	data      string
	expectErr bool
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
`,
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
			expectErr: true,
		},
		{
			name: "no version",
			data: `---
- group: mygroup
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectErr: true,
		},
		{
			name: "no kind",
			data: `---
- group: mygroup
  version: v1alpha1
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
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
			expectErr: true,
		},
		{
			name: "invalid yaml",
			data: `---
foo: bar
`,
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

			_, err = Load(tmp.Name())
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %w", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}
		})
	}
}
