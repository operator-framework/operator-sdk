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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/iancoleman/strcase"
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

// HelmChartsDir is the relative directory within an SDK project where Helm
// charts are stored.
const HelmChartsDir string = "helm-charts"

func CreateChart(apiVersion, kind, source, sourceRepo, projectDir string) (*scaffold.Resource, string, error) {
	chartDir := filepath.Join(projectDir, HelmChartsDir)
	err := os.MkdirAll(chartDir, 0755)
	if err != nil {
		return nil, "", err
	}

	// If we don't have a source helm chart, scaffold the standard Nginx chart
	// from Helm's default template.
	if len(source) == 0 {
		r, err := scaffold.NewResource(apiVersion, kind)
		if err != nil {
			return nil, "", err
		}
		log.Infof("Create %s/%s/", HelmChartsDir, r.LowerKind)

		if err := createChartForResource(r, chartDir); err != nil {
			return nil, "", err
		}
		return r, "", nil
	}

	var (
		stat  os.FileInfo
		chart *chart.Chart
	)

	if stat, err = os.Stat(source); err == nil {
		chart, err = createChartFromDisk(source, chartDir, stat.IsDir())
	} else {
		chart, err = createChartFromRepo(source, sourceRepo, chartDir)
	}
	if err != nil {
		return nil, "", err
	}

	chartName := chart.GetMetadata().GetName()
	if len(apiVersion) == 0 {
		apiVersion = "charts.helm.k8s.io/v1alpha1"
	}
	if len(kind) == 0 {
		kind = strcase.ToCamel(chartName)
	}

	r, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		return nil, "", err
	}
	log.Infof("Create %s/%s/", HelmChartsDir, chartName)
	return r, chartName, nil
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

func createChartFromRepo(source, sourceRepo, destDir string) (*chart.Chart, error) {
	if strings.HasSuffix(source, ".tgz") {
		tmpDir, err := ioutil.TempDir("", "osdk-helm-chart")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpDir)

		args := []string{"fetch", source, "--destination", tmpDir}
		if len(sourceRepo) != 0 {
			args = append(args, "--repo", sourceRepo)
		}
		cmd := exec.Command("helm", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		chartArchive := filepath.Join(tmpDir, filepath.Base(source))
		return createChartFromDisk(chartArchive, destDir, false)
	}

	args := []string{"fetch", source, "--untar", "--untardir", destDir}
	if len(sourceRepo) != 0 {
		args = append(args, "--repo", sourceRepo)
	}
	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	chartDir := filepath.Join(destDir, filepath.Base(source))
	return chartutil.LoadDir(chartDir)
}

// createChartForResource creates a new helm chart in the SDK project for the
// provided resource.
func createChartForResource(r *scaffold.Resource, chartDir string) error {
	chartfile := &chart.Metadata{
		Name:        r.LowerKind,
		Description: "A Helm chart for Kubernetes",
		Version:     "0.1.0",
		AppVersion:  "1.0",
		ApiVersion:  chartutil.ApiVersionV1,
	}
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		return err
	}
	_, err := chartutil.Create(chartfile, chartDir)
	return err
}
