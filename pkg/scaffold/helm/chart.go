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

package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/iancoleman/strcase"
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
)

const (

	// HelmChartsDir is the relative directory within an SDK project where Helm
	// charts are stored.
	HelmChartsDir string = "helm-charts"

	// DefaultAPIVersion is the Kubernetes CRD API Version used for fetched
	// charts when the --api-version flag is not specified
	DefaultAPIVersion string = "charts.helm.k8s.io/v1alpha1"
)

type FetchChartOptions struct {
	Chart   string
	Repo    string
	Version string
}

func CreateChart(apiVersion, kind string, opts FetchChartOptions, projectDir string) (*scaffold.Resource, *chart.Chart, error) {
	chartsDir := filepath.Join(projectDir, HelmChartsDir)
	err := os.MkdirAll(chartsDir, 0755)
	if err != nil {
		return nil, nil, err
	}

	var (
		r *scaffold.Resource
		c *chart.Chart
	)

	// If we don't have a helm chart reference, scaffold the default chart
	// from Helm's default template. Otherwise, fetch it.
	if len(opts.Chart) == 0 {
		r, c, err = scaffoldChart(apiVersion, kind, chartsDir)
	} else {
		r, c, err = fetchChart(apiVersion, kind, opts, chartsDir)
	}
	if err != nil {
		return nil, nil, err
	}
	log.Infof("Create %s/%s/", HelmChartsDir, c.GetMetadata().GetName())
	return r, c, nil
}

func scaffoldChart(apiVersion, kind, destDir string) (*scaffold.Resource, *chart.Chart, error) {
	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		return nil, nil, err
	}

	chartfile := &chart.Metadata{
		Name:        r.LowerKind,
		Description: "A Helm chart for Kubernetes",
		Version:     "0.1.0",
		AppVersion:  "1.0",
		ApiVersion:  chartutil.ApiVersionV1,
	}
	chartPath, err := chartutil.Create(chartfile, destDir)
	if err != nil {
		return nil, nil, err
	}

	chart, err := chartutil.LoadDir(chartPath)
	if err != nil {
		return nil, nil, err
	}
	return r, chart, nil
}

func fetchChart(apiVersion, kind string, opts FetchChartOptions, destDir string) (*scaffold.Resource, *chart.Chart, error) {
	var (
		stat  os.FileInfo
		chart *chart.Chart
		err   error
	)

	if stat, err = os.Stat(opts.Chart); err == nil {
		chart, err = createChartFromDisk(opts.Chart, destDir, stat.IsDir())
	} else {
		chart, err = createChartFromRemote(opts, destDir)
	}
	if err != nil {
		return nil, nil, err
	}

	chartName := chart.GetMetadata().GetName()
	if len(apiVersion) == 0 {
		apiVersion = DefaultAPIVersion
	}
	if len(kind) == 0 {
		kind = strcase.ToCamel(chartName)
	}

	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		return nil, nil, err
	}
	return r, chart, nil
}

func createChartFromDisk(source, destDir string, isDir bool) (*chart.Chart, error) {
	var (
		chart *chart.Chart
		err   error
	)

	// If source is a file or directory, attempt to load it
	if isDir {
		chart, err = chartutil.LoadDir(source)
	} else {
		chart, err = chartutil.LoadFile(source)
	}
	if err != nil {
		return nil, err
	}

	// Save it into our project's helm-charts directory.
	if err := chartutil.SaveDir(chart, destDir); err != nil {
		return nil, err
	}
	return chart, nil
}

func createChartFromRemote(opts FetchChartOptions, destDir string) (*chart.Chart, error) {
	helmHome, ok := os.LookupEnv(environment.HomeEnvVar)
	if !ok {
		helmHome = environment.DefaultHelmHome
	}
	getters := getter.All(environment.EnvSettings{})
	c := downloader.ChartDownloader{
		HelmHome: helmpath.Home(helmHome),
		Out:      os.Stderr,
		Getters:  getters,
	}

	if opts.Repo != "" {
		chartURL, err := repo.FindChartInRepoURL(opts.Repo, opts.Chart, opts.Version, "", "", "", getters)
		if err != nil {
			return nil, err
		}
		opts.Chart = chartURL
	}

	tmpDir, err := ioutil.TempDir("", "osdk-helm-chart")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	chartArchive, _, err := c.DownloadTo(opts.Chart, opts.Version, tmpDir)
	if err != nil {
		// One of Helm's error messages directs users to run `helm init`, which
		// installs tiller in a remote cluster. Since that's unnecessary and
		// unhelpful, modify the error message to be relevant for operator-sdk.
		if strings.Contains(err.Error(), "Couldn't load repositories file") {
			return nil, fmt.Errorf("failed to load repositories file %s "+
				"(you might need to run `helm init --client-only` "+
				"to create and initialize it)", c.HelmHome.RepositoryFile())
		}
		return nil, err
	}

	return createChartFromDisk(chartArchive, destDir, false)
}
