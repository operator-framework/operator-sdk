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
package helm

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"
	sdkscaffold "github.com/operator-framework/operator-sdk/internal/scaffold"
)

const (
	apiVersion       = "api-version"
	kind             = "kind"
	crdVersion       = "crd-version"
	helmChart        = "helm-chart"
	helmChartVersion = "helm-chart-version"
	helmChartRepo    = "helm-chart-repo"
)

type APIFlags struct {
	APIVersion       string
	Kind             string
	CRDVersion       string
	HelmChartRef     string
	HelmChartVersion string
	HelmChartRepo    string
}

// AddTo will add the API flags
func (f *APIFlags) AddTo(fs *pflag.FlagSet) {
	fs.StringVar(&f.APIVersion, apiVersion, "",
		"Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	fs.StringVar(&f.Kind, kind, "", "Kubernetes resource Kind name. (e.g AppService)")
	fs.StringVar(&f.CRDVersion, crdVersion, gencrd.DefaultCRDVersion,
		"CRD version to generate")
	fs.StringVar(&f.HelmChartRef, helmChart, "",
		"Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path).")
	fs.StringVar(&f.HelmChartVersion, helmChartVersion, "",
		"Specific version of the helm chart (default is latest version)")
	fs.StringVar(&f.HelmChartRepo, helmChartRepo, "",
		"Chart repository URL for the requested helm chart")
}

// Validate will verify the helm API flags
func (f *APIFlags) Validate() error {
	if len(strings.TrimSpace(f.HelmChartRef)) == 0 {
		if len(strings.TrimSpace(f.HelmChartRepo)) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartRepo, helmChart)
		} else if len(f.HelmChartVersion) != 0 {
			return fmt.Errorf("value of --%s can only be used with --%s", helmChartVersion, helmChart)
		}
	}

	if len(strings.TrimSpace(f.HelmChartRef)) == 0 {
		if len(strings.TrimSpace(f.APIVersion)) == 0 {
			return fmt.Errorf("value of --%s must not have empty value", apiVersion)
		}
		if len(strings.TrimSpace(f.Kind)) == 0 {
			return fmt.Errorf("value of --%s must not have empty value", kind)
		}
		// Validate the resource.
		_, err := sdkscaffold.NewResource(f.APIVersion, f.Kind)
		if err != nil {
			return err
		}
	}
	return nil
}
