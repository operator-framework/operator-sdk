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

package helm_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/repo/repotest"
)

func TestCreateChart(t *testing.T) {
	srv, err := repotest.NewTempServer("testdata/testcharts/*.tgz")
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
		customAPIVersion   = "example.com/v1"
		customKind         = "MyApp"
		customExpectName   = "myapp"
		expectDerivedKind  = "TestChart"
	)

	testCases := []createChartTestCase{
		{
			name:      "from scaffold no apiVersion",
			expectErr: true,
		},
		{
			name:      "from scaffold no kind",
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
			name:               "from scaffold with apiVersion and kind",
			apiVersion:         customAPIVersion,
			kind:               customKind,
			expectResource:     mustNewResource(t, customAPIVersion, customKind),
			expectChartName:    customExpectName,
			expectChartVersion: "0.1.0",
		},
		{
			name:               "from directory",
			helmChart:          filepath.Join(".", "testdata", "testcharts", chartName),
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from archive",
			helmChart:          filepath.Join(".", "testdata", "testcharts", fmt.Sprintf("%s-%s.tgz", chartName, latestVersion)),
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from url",
			helmChart:          fmt.Sprintf("%s/%s-%s.tgz", srv.URL(), chartName, latestVersion),
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest",
			helmChart:          "test/" + chartName,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with apiVersion",
			helmChart:          "test/" + chartName,
			apiVersion:         customAPIVersion,
			expectResource:     mustNewResource(t, customAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with kind",
			helmChart:          "test/" + chartName,
			kind:               customKind,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, customKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with apiVersion and kind",
			helmChart:          "test/" + chartName,
			apiVersion:         customAPIVersion,
			kind:               customKind,
			expectResource:     mustNewResource(t, customAPIVersion, customKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit latest",
			helmChart:          "test/" + chartName,
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit previous",
			helmChart:          "test/" + chartName,
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
		{
			name:               "from name and repo url implicit latest",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit latest",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit previous",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
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

	apiVersion       string
	kind             string
	helmChart        string
	helmChartVersion string
	helmChartRepo    string

	expectResource     *scaffold.Resource
	expectChartName    string
	expectChartVersion string
	expectErr          bool
}

func mustNewResource(t *testing.T, apiVersion, kind string) *scaffold.Resource {
	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		t.Fatalf("Could not create resource for apiVersion=%s kind=%s: %s", apiVersion, kind, err)
	}
	return r
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

	opts := helm.CreateChartOptions{
		ResourceAPIVersion: tc.apiVersion,
		ResourceKind:       tc.kind,
		Chart:              tc.helmChart,
		Version:            tc.helmChartVersion,
		Repo:               tc.helmChartRepo,
	}
	resource, chart, err := helm.CreateChart(outputDir, opts)
	if tc.expectErr {
		assert.Error(t, err)
		return
	}

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, tc.expectResource, resource)
	assert.Equal(t, tc.expectChartName, chart.Name())
	assert.Equal(t, tc.expectChartVersion, chart.Metadata.Version)

	loadedChart, err := loader.Load(filepath.Join(outputDir, helm.HelmChartsDir, chart.Name()))
	if err != nil {
		t.Fatalf("Could not load chart from expected location: %s", err)
	}

	assert.Equal(t, loadedChart, chart)
}
