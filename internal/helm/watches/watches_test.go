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
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestLoadReader(t *testing.T) {
	trueVal, falseVal := true, false
	testCases := []struct {
		name          string
		data          string
		env           map[string]string
		expectWatches []Watch
		expectErr     bool
	}{
		{
			name: "valid",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  watchDependentResources: false
  overrideValues:
    key: value
`,
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
				},
			},
			expectErr: false,
		},
		{
			name: "valid with override env expansion",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  watchDependentResources: false
  overrideValues:
    key: $MY_VALUE
`,
			env: map[string]string{"MY_VALUE": "value"},
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
				},
			},
			expectErr: false,
		},
		{
			name: "valid with override template expansion",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  watchDependentResources: false
  overrideValues:
    repo: '{{ ("$MY_IMAGE" | split ":")._0 }}'
    tag: '{{ ("$MY_IMAGE" | split ":")._1 }}'
`,
			env: map[string]string{"MY_IMAGE": "quay.io/operator-framework/helm-operator:latest"},
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues: map[string]string{
						"repo": "quay.io/operator-framework/helm-operator",
						"tag":  "latest",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "valid with dry run option",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  watchDependentResources: false
  overrideValues:
    key: $MY_VALUE
  dryRunOption: server
`,
			env: map[string]string{"MY_VALUE": "value"},
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
					DryRunOption:            "server",
				},
			},
			expectErr: false,
		},
		{
			name: "invalid with override template expansion",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  watchDependentResources: false
  overrideValues:
    repo: '{{ ("$MY_IMAGE" | split ":")._0 }}'
    tag: '{{ ("$MY_IMAGE" | split ":")._1'
`,
			env:       map[string]string{"MY_IMAGE": "quay.io/operator-framework/helm-operator:latest"},
			expectErr: true,
		},
		{
			name: "multiple gvk",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyFirstKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
- group: mygroup
  version: v1alpha1
  kind: MySecondKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
`,
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyFirstKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &trueVal,
				},
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MySecondKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &trueVal,
				},
			},
			expectErr: false,
		},
		{
			name: "duplicate gvk",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
`,
			expectErr: true,
		},
		{
			name: "no version",
			data: `---
- group: mygroup
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
`,
			expectErr: true,
		},
		{
			name: "no kind",
			data: `---
- group: mygroup
  version: v1alpha1
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
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
			name: "invalid overrides",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  overrideValues:
    key1:
		key2: value
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
			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %q: %v", k, err)
				}
			}

			watchesData := bytes.NewBufferString(tc.data)
			watches, err := LoadReader(watchesData)
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %v", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}
			assert.Equal(t, tc.expectWatches, watches)

			for k := range tc.env {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("Failed to unset environment variable %q: %v", k, err)
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	falseVal := false
	testCases := []struct {
		name          string
		data          string
		env           map[string]string
		expectWatches []Watch
		expectErr     bool
	}{
		{
			name: "valid",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/plugins/helm/v1/chartutil/testdata/test-chart
  watchDependentResources: false
  overrideValues:
    key: value
`,
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/plugins/helm/v1/chartutil/testdata/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %q: %v", k, err)
				}
			}

			f, err := os.CreateTemp("", "osdk-test-load")
			if err != nil {
				t.Fatalf("Failed to create temporary watches file: %v", err)
			}
			defer removeFile(t, f)
			if _, err := f.WriteString(tc.data); err != nil {
				t.Fatalf("Failed to write temporary watches file: %v", err)
			}
			watches, err := Load(f.Name())
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %v", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}
			assert.Equal(t, tc.expectWatches, watches)

			for k := range tc.env {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("Failed to unset environment variable %q: %v", k, err)
				}
			}
		})
	}
}

// remove removes path from disk. Used in defer statements.
func removeFile(t *testing.T, f *os.File) {
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.Name()); err != nil {
		t.Fatal(err)
	}
}
