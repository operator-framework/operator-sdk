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

package chartutil_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo/repotest"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
)

func TestChart(t *testing.T) {
	srv, err := repotest.NewTempServerWithCleanup(t, "testdata/*.tgz")
	if err != nil {
		t.Fatalf("Failed to create new temp server: %s", err)
	}
	defer srv.Stop()

	if err := srv.LinkIndices(); err != nil {
		t.Fatalf("Failed to link server indices: %s", err)
	}

	const (
		chartName          = "test-chart"
		latestVersion      = "1.2.3"
		previousVersion    = "1.2.0"
		nonExistentVersion = "0.0.1"
		customKind         = "MyApp"
		customExpectName   = "myapp"
	)

	testCases := []createChartTestCase{
		{
			name:      "from scaffold no apiVersion",
			expectErr: true,
		},
		{
			name:             "version without helm chart",
			helmChartVersion: latestVersion,
			expectErr:        true,
		},
		{
			name:          "repo without helm chart",
			helmChartRepo: srv.URL(),
			expectErr:     true,
		},
		{
			name:             "non-existent version",
			helmChart:        "test/" + chartName,
			helmChartVersion: nonExistentVersion,
			expectErr:        true,
		},
		{
			name:               "from scaffold with kind",
			kind:               customKind,
			expectChartName:    customExpectName,
			expectChartVersion: "0.1.0",
		},
		{
			name:               "from directory",
			helmChart:          filepath.Join(".", "testdata", chartName),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from archive",
			helmChart:          filepath.Join(".", "testdata", fmt.Sprintf("%s-%s.tgz", chartName, latestVersion)),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from url",
			helmChart:          fmt.Sprintf("%s/%s-%s.tgz", srv.URL(), chartName, latestVersion),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest",
			helmChart:          "test/" + chartName,
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with kind",
			helmChart:          "test/" + chartName,
			kind:               customKind,
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit latest",
			helmChart:          "test/" + chartName,
			helmChartVersion:   latestVersion,
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit previous",
			helmChart:          "test/" + chartName,
			helmChartVersion:   previousVersion,
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
		{
			name:               "from name and repo url implicit latest",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit latest",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			helmChartVersion:   latestVersion,
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit previous",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			helmChartVersion:   previousVersion,
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, srv.Root(), tc)
		})
	}
}

type createChartTestCase struct {
	name string

	kind             string
	helmChart        string
	helmChartVersion string
	helmChartRepo    string

	expectChartName    string
	expectChartVersion string
	expectErr          bool
}

func runTestCase(t *testing.T, testDir string, tc createChartTestCase) {
	outputDir := filepath.Join(testDir, "output")
	assert.NoError(t, os.Mkdir(outputDir, 0755))
	defer os.RemoveAll(outputDir)

	os.Setenv("XDG_CONFIG_HOME", filepath.Join(testDir, ".config"))
	os.Setenv("XDG_CACHE_HOME", filepath.Join(testDir, ".cache"))
	os.Setenv("HELM_REPOSITORY_CONFIG", filepath.Join(testDir, "repositories.yaml"))
	os.Setenv("HELM_REPOSITORY_CACHE", filepath.Join(testDir))
	defer os.Unsetenv("XDG_CONFIG_HOME")
	defer os.Unsetenv("XDG_CACHE_HOME")
	defer os.Unsetenv("HELM_REPOSITORY_CONFIG")
	defer os.Unsetenv("HELM_REPOSITORY_CACHE")

	var (
		chrt *chart.Chart
		err  error
	)
	if tc.helmChart != "" {
		opts := chartutil.Options{
			Chart:   tc.helmChart,
			Version: tc.helmChartVersion,
			Repo:    tc.helmChartRepo,
		}
		chrt, err = chartutil.LoadChart(opts)
	} else {
		chrt, err = chartutil.NewChart(strings.ToLower(tc.kind))
	}

	if tc.expectErr {
		assert.Error(t, err)
		return
	}

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, tc.expectChartName, chrt.Name())
	assert.Equal(t, tc.expectChartVersion, chrt.Metadata.Version)

	_, chartPath, err := chartutil.ScaffoldChart(chrt, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(chartutil.HelmChartsDir, tc.expectChartName), chartPath)
}
