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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/helm"

	"github.com/stretchr/testify/assert"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
)

const (
	repoServerAddr = "127.0.0.1:8879"
)

func TestCreateChart(t *testing.T) {
	const (
		chartName          = "test-chart"
		latestVersion      = "1.2.3"
		previousVersion    = "1.2.0"
		nonExistentVersion = "0.0.1"
		customApiVersion   = "example.com/v1"
		customKind         = "MyApp"
		customExpectName   = "myapp"
		expectDerivedKind  = "TestChart"
	)

	testDir, err := ioutil.TempDir("", "osdk-test")
	if err != nil {
		t.Fatalf("Failed to create temp test directory: %s", err)
	}
	defer os.RemoveAll(testDir)

	helmHomeDir := filepath.Join(testDir, "helmhome")
	if err := os.Mkdir(helmHomeDir, 0755); err != nil {
		t.Fatalf("Failed to create temp helm home directory: %s", err)
	}

	latest, previous, localDir, err := createLocalChartRepo(helmHomeDir, chartName, latestVersion, previousVersion)
	if err != nil {
		t.Fatalf("Failed to create local chart repo: %s", err)
	}

	if err := chartutil.SaveDir(latest.chart, localDir); err != nil {
		t.Fatalf("Failed to save latest chart as directory: %s", err)
	}
	latestDirectory := filepath.Join(localDir, latest.chart.GetMetadata().GetName())

	testRepo := http.Server{
		Addr:    repoServerAddr,
		Handler: &repo.RepositoryServer{RepoPath: localDir},
	}

	var (
		repoURL       = fmt.Sprintf("http://%s/", testRepo.Addr)
		repoURLCharts = fmt.Sprintf("http://%s/charts/", testRepo.Addr)
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := testRepo.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Fatalf("Failed to run test repo server: %s", err)
		}
		wg.Done()
	}()

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
			helmChartRepo: repoURL,
			expectErr:     true,
		},
		{
			name:             "non-existent version",
			helmChart:        latest.repoAndName,
			helmChartVersion: nonExistentVersion,
			expectErr:        true,
		},
		{
			name:               "from scaffold with apiVersion and kind",
			apiVersion:         customApiVersion,
			kind:               customKind,
			expectResource:     mustNewResource(t, customApiVersion, customKind),
			expectChartName:    customExpectName,
			expectChartVersion: "0.1.0",
		},
		{
			name:               "from directory",
			helmChart:          latestDirectory,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from archive",
			helmChart:          latest.archive,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from url",
			helmChart:          latest.url,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest",
			helmChart:          latest.repoAndName,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with apiVersion",
			helmChart:          latest.repoAndName,
			apiVersion:         customApiVersion,
			expectResource:     mustNewResource(t, customApiVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with kind",
			helmChart:          latest.repoAndName,
			kind:               customKind,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, customKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name implicit latest with apiVersion and kind",
			helmChart:          latest.repoAndName,
			apiVersion:         customApiVersion,
			kind:               customKind,
			expectResource:     mustNewResource(t, customApiVersion, customKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit latest",
			helmChart:          latest.repoAndName,
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from repo and name explicit previous",
			helmChart:          previous.repoAndName,
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
		{
			name:               "from name and repo url implicit latest",
			helmChart:          chartName,
			helmChartRepo:      repoURL,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit latest",
			helmChart:          chartName,
			helmChartRepo:      repoURL,
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and repo url explicit previous",
			helmChart:          chartName,
			helmChartRepo:      repoURL,
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
		{
			name:               "from name and charts repo url implicit latest",
			helmChart:          chartName,
			helmChartRepo:      repoURLCharts,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and charts repo url explicit latest",
			helmChart:          chartName,
			helmChartRepo:      repoURLCharts,
			helmChartVersion:   latestVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: latestVersion,
		},
		{
			name:               "from name and charts repo url explicit previous",
			helmChart:          chartName,
			helmChartRepo:      repoURLCharts,
			helmChartVersion:   previousVersion,
			expectResource:     mustNewResource(t, helm.DefaultAPIVersion, expectDerivedKind),
			expectChartName:    chartName,
			expectChartVersion: previousVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runTestCase(t, testDir, tc)
		})
	}

	if err := testRepo.Close(); err != nil {
		t.Fatalf("Failed to close test repo server: %s", err)
	}
	wg.Wait()
}

type testChart struct {
	chart       *chart.Chart
	archive     string
	repoAndName string
	url         string
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
	} else {
		assert.NoError(t, err)
	}

	assert.Equal(t, tc.expectResource, resource)
	assert.Equal(t, tc.expectChartName, chart.GetMetadata().GetName())
	assert.Equal(t, tc.expectChartVersion, chart.GetMetadata().GetVersion())

	loadedChart, err := chartutil.Load(filepath.Join(outputDir, helm.HelmChartsDir, chart.GetMetadata().GetName()))
	if err != nil {
		t.Fatalf("Could not load chart from expected location: %s", err)
	}

	assert.Equal(t, loadedChart, chart)
}

func createLocalChartRepo(helmHomeDir, chartName, latestVersion, previousVersion string) (*testChart, *testChart, string, error) {
	if err := os.Setenv("HELM_HOME", helmHomeDir); err != nil {
		return nil, nil, "", err
	}

	var (
		localURL = fmt.Sprintf("http://%s", repoServerAddr)

		repoDir  = filepath.Join(helmHomeDir, "repository")
		cacheDir = filepath.Join(helmHomeDir, "repository", "cache")
		localDir = filepath.Join(helmHomeDir, "repository", "local")

		cacheFilePath  = filepath.Join(helmHomeDir, "repository", "cache", "local-index.yaml")
		localIndexPath = filepath.Join(helmHomeDir, "repository", "local", "index.yaml")
		repoFilePath   = filepath.Join(helmHomeDir, "repository", "repositories.yaml")
	)

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return nil, nil, "", err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, nil, "", err
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return nil, nil, "", err
	}
	repoFile := repo.RepoFile{
		APIVersion: "v1",
		Generated:  time.Now(),
		Repositories: []*repo.Entry{
			{
				Name:  "local",
				Cache: cacheFilePath,
				URL:   localURL,
			},
		},
	}
	if err := repoFile.WriteFile(repoFilePath, 0644); err != nil {
		return nil, nil, "", err
	}

	latest, err := createTestChart(localDir, chartName, latestVersion)
	if err != nil {
		return nil, nil, "", err
	}

	previous, err := createTestChart(localDir, chartName, previousVersion)
	if err != nil {
		return nil, nil, "", err
	}

	localIndex, err := repo.IndexDirectory(localDir, localURL+"/charts")
	if err != nil {
		return nil, nil, "", err
	}
	if err := localIndex.WriteFile(cacheFilePath, 0644); err != nil {
		return nil, nil, "", err
	}
	if err := localIndex.WriteFile(localIndexPath, 0644); err != nil {
		return nil, nil, "", err
	}
	return latest, previous, localDir, nil
}

func createTestChart(chartsDir, name, version string) (*testChart, error) {
	dir, err := chartutil.Create(&chart.Metadata{
		Name:    name,
		Version: version,
	}, chartsDir)
	if err != nil {
		return nil, fmt.Errorf("could not create test chart directory: %s", err)
	}
	defer os.RemoveAll(dir)

	chart, err := chartutil.LoadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("could not load chart from directory: %s", err)
	}

	archive, err := chartutil.Save(chart, chartsDir)
	if err != nil {
		return nil, fmt.Errorf("could not save chart archive: %s", err)
	}

	return &testChart{
		chart:       chart,
		archive:     archive,
		repoAndName: "local/" + chart.GetMetadata().GetName(),
		url:         fmt.Sprintf("http://%s/charts/%s", repoServerAddr, filepath.Base(archive)),
	}, nil
}
