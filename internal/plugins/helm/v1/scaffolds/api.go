/*
Copyright 2019 The Kubernetes Authors.
Modifications copyright 2020 The Operator-SDK Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaffolds

import (
	"fmt"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
	internalchartutil "github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/crd"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/samples"
)

var _ cmdutil.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config   config.Config
	resource *resource.Resource
	chrt     *chart.Chart
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(config config.Config, res *resource.Resource, chrt *chart.Chart) cmdutil.Scaffolder {
	return &apiScaffolder{
		config:   config,
		resource: res,
		chrt:     chrt,
	}
}

// Scaffold implements Scaffolder
func (s *apiScaffolder) Scaffold() error {
	return s.scaffold()
}

func (s *apiScaffolder) newUniverse(r *resource.Resource) *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithResource(r),
	)
}

func (s *apiScaffolder) scaffold() error {
	if err := s.config.UpdateResource(*s.resource); err != nil {
		return err
	}
	// Path for file builders.
	chartPath := filepath.Join(internalchartutil.HelmChartsDir, s.chrt.Name())

	// Write the chart to disk.
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}
	absChartDir := filepath.Join(projectDir, internalchartutil.HelmChartsDir)
	if err := chartutil.SaveDir(s.chrt, absChartDir); err != nil {
		return err
	}
	fmt.Println("Created", chartPath)

	if err := machinery.NewScaffold().Execute(
		s.newUniverse(s.resource),
		&templates.WatchesUpdater{ChartPath: chartPath},
		&crd.CRD{CRDVersion: s.resource.API.CRDVersion},
		&crd.Kustomization{},
		&rbac.CRDEditorRole{},
		&rbac.CRDViewerRole{},
		&rbac.ManagerRoleUpdater{Chart: s.chrt},
		&samples.CRDSample{ChartPath: chartPath, Chart: s.chrt},
	); err != nil {
		return fmt.Errorf("error scaffolding APIs: %v", err)
	}

	return nil
}
