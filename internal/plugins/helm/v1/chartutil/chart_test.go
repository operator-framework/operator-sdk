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
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/repo/repotest"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
)

func TestCreateChart(t *testing.T) {
	srv, err := repotest.NewTempServer("testdata/*.tgz")
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
		customGroup        = "example.com"
		customVersion      = "v1"
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
			group:              customGroup,
			version:            customVersion,
			kind:               customKind,
			expectResource:     mustNewResource(customGroup, customVersion, customKind),
			expectChartName:    customExpectName,
			expectChartVersion: "0.1.0",
		},
		{
			name:               "from directory",
			helmChart:          filepath.Join(".", "testdata", chartName),
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from archive",
			helmChart:          filepath.Join(".", "testdata", fmt.Sprintf("%s-%s.tgz", chartName, latestVersion)),
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from url",
			helmChart:          fmt.Sprintf("%s/%s-%s.tgz", srv.URL(), chartName, latestVersion),
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest",
			helmChart:          "test/" + chartName,
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with apiVersion",
			helmChart:          "test/" + chartName,
			group:              customGroup,
			version:            customVersion,
			expectResource:     mustNewResource(customGroup, customVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with kind",
			helmChart:          "test/" + chartName,
			kind:               customKind,
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, customKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with apiVersion and kind",
			helmChart:          "test/" + chartName,
			group:              customGroup,
			version:            customVersion,
			kind:               customKind,
			expectResource:     mustNewResource(customGroup, customVersion, customKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit latest",
			helmChart:          "test/" + chartName,
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit previous",
			helmChart:          "test/" + chartName,
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
		{
			name:               "from name and repo url implicit latest",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit latest",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit previous",
			helmChart:          chartName,
			helmChartRepo:      srv.URL(),
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(chartutil.DefaultGroup, chartutil.DefaultVersion, expectDerivedKind),
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

	group            string
	version          string
	kind             string
	helmChart        string
	helmChartVersion string
	helmChartRepo    string

	expectResource     *resource.Options
	expectChartName    string
	expectChartVersion string
	expectErr          bool
}

func mustNewResource(group, version, kind string) *resource.Options {
	r := &resource.Options{
		Namespaced: true,
		Group:      group,
		Version:    version,
		Kind:       kind,
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

	opts := chartutil.CreateOptions{
		GVK: schema.GroupVersionKind{
			Group:   tc.group,
			Version: tc.version,
			Kind:    tc.kind,
		},
		Chart:   tc.helmChart,
		Version: tc.helmChartVersion,
		Repo:    tc.helmChartRepo,
	}
	resource, chrt, err := chartutil.CreateChart(outputDir, opts)
	if tc.expectErr {
		assert.Error(t, err)
		return
	}

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, tc.expectResource, resource)
	assert.Equal(t, tc.expectChartName, chrt.Name())
	assert.Equal(t, tc.expectChartVersion, chrt.Metadata.Version)

	loadedChart, err := loader.Load(filepath.Join(outputDir, chartutil.HelmChartsDir, chrt.Name()))
	if err != nil {
		t.Fatalf("Could not load chart from expected location: %s", err)
	}

	assert.Equal(t, loadedChart, chrt)
}
