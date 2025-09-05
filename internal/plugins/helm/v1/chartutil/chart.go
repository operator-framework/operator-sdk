// Copyright 2018 The Operator-SDK Authors
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

package chartutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	// HelmChartsDir is the relative directory within a SDK project where Helm charts are stored.
	HelmChartsDir = "helm-charts"
)

// Options is used to configure how a Helm chart is scaffolded
// for a new Helm operator project.
type Options struct {
	// Chart is a chart reference for a local or remote chart.
	Chart string

	// Repo is a URL to a custom chart repository.
	Repo string

	// Version is the version of the chart to fetch.
	Version string
}

// NewChart creates a new helm chart for the project from helm's default template.
// It returns a chart.Chart that references the newly created chart or an error.
func NewChart(name string) (*chart.Chart, error) {
	tmpDir, err := os.MkdirTemp("", "osdk-helm-chart")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Errorf("Failed to remove temporary directory %s: %v", tmpDir, err)
		}
	}()

	// Create a new chart
	chartPath, err := chartutil.Create(name, tmpDir)
	if err != nil {
		return nil, err
	}

	return loader.Load(chartPath)
}

// LoadChart creates a new helm chart for the project based on the passed opts.
// It returns a chart.Chart that references the newly created chart or an error.
//
// If opts.Chart is a local file, it verifies that it is a valid helm chart
// archive and returns its chart.Chart representation.
//
// If opts.Chart is a local directory, it verifies that it is a valid helm chart
// directory and returns its chart.Chart representation.
//
// For any other value of opts.Chart, it attempts to fetch the helm chart from a
// remote repository.
//
// If opts.Repo is not specified, the following chart reference formats are supported:
//
//   - <repoName>/<chartName>: Fetch the helm chart named chartName from the helm
//     chart repository named repoName, as specified in the
//     $HELM_HOME/repositories/repositories.yaml file.
//
//   - <url>: Fetch the helm chart archive at the specified URL.
//
// If opts.Repo is specified, only one chart reference format is supported:
//
//   - <chartName>: Fetch the helm chart named chartName in the helm chart repository
//     specified by opts.Repo
//
// If opts.Version is not set, it will fetch the latest available version of the helm
// chart. Otherwise, it will fetch the specified version.
// opts.Version is not used when opts.Chart itself refers to a specific version, for
// example when it is a local path or a URL.
func LoadChart(opts Options) (*chart.Chart, error) {
	tmpDir, err := os.MkdirTemp("", "osdk-helm-chart")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Errorf("Failed to remove temporary directory %s: %v", tmpDir, err)
		}
	}()

	chartPath := opts.Chart

	// If it is a remote chart, download it to a temp dir first
	if _, err := os.Stat(opts.Chart); err != nil {
		chartPath, err = downloadChart(tmpDir, opts)
		if err != nil {
			return nil, err
		}
	}

	return loader.Load(chartPath)
}

func downloadChart(destDir string, opts Options) (string, error) {
	settings := cli.New()
	getters := getter.All(settings)

	// Create registry client for OCI registry support
	registryClient, err := registry.NewClient()
	if err != nil {
		return "", fmt.Errorf("failed to create registry client: %w", err)
	}

	c := downloader.ChartDownloader{
		Out:              os.Stderr,
		Getters:          getters,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
		RegistryClient:   registryClient,
	}

	if opts.Repo != "" {
		chartURL, err := repo.FindChartInRepoURL(opts.Repo, opts.Chart, opts.Version, "", "", "", getters)
		if err != nil {
			return "", err
		}
		opts.Chart = chartURL
	}

	chartArchive, _, err := c.DownloadTo(opts.Chart, opts.Version, destDir)
	if err != nil {
		return "", err
	}

	return chartArchive, nil
}

// ScaffoldChart scaffolds the provided chart.Chart to a known directory relative to projectDir
//
// # It also fetches the dependencies and reloads the chart.Chart
//
// It returns the reloaded chart, the relative path, or an error.
func ScaffoldChart(chrt *chart.Chart, projectDir string) (*chart.Chart, string, error) {
	chartsPath := filepath.Join(projectDir, HelmChartsDir)

	// Save it into our project's helm-charts directory.
	if err := chartutil.SaveDir(chrt, chartsPath); err != nil {
		return chrt, "", err
	}

	chartPath := filepath.Join(chartsPath, chrt.Name())

	// Fetch dependencies
	if err := fetchChartDependencies(chartPath); err != nil {
		return chrt, "", fmt.Errorf("failed to fetch chart dependencies: %w", err)
	}

	// Reload chart in case dependencies changed
	chrt, err := loader.Load(chartPath)
	if err != nil {
		return chrt, "", fmt.Errorf("failed to reload chart: %w", err)
	}

	return chrt, filepath.Join(HelmChartsDir, chrt.Name()), nil
}

func fetchChartDependencies(chartPath string) error {
	settings := cli.New()
	getters := getter.All(settings)

	// Create registry client for OCI registry support
	registryClient, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}

	out := &bytes.Buffer{}
	man := &downloader.Manager{
		Out:              out,
		ChartPath:        chartPath,
		Getters:          getters,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
		RegistryClient:   registryClient,
	}
	if err := man.Build(); err != nil {
		fmt.Println(out.String())
		return err
	}
	return nil
}
