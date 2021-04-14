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

	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/crd"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/samples"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	fs machinery.Filesystem

	config   config.Config
	resource resource.Resource
	chrt     *chart.Chart
}

// NewAPIScaffolder returns a new plugins.Scaffolder for API/controller creation operations
func NewAPIScaffolder(cfg config.Config, res resource.Resource, chrt *chart.Chart) plugins.Scaffolder {
	return &apiScaffolder{
		config:   cfg,
		resource: res,
		chrt:     chrt,
	}
}

// InjectFS implements plugins.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements plugins.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	if err := s.config.UpdateResource(s.resource); err != nil {
		return err
	}

	// Get current directory
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Save the loaded chart.Chart
	var chartPath string
	s.chrt, chartPath, err = chartutil.ScaffoldChart(s.chrt, projectDir)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s\n", chartPath)

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		// NOTE: kubebuilder's default permissions are only for root users
		machinery.WithDirectoryPermissions(0755),
		machinery.WithFilePermissions(0644),
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	if err := scaffold.Execute(
		&templates.WatchesUpdater{ChartPath: chartPath},
		&crd.CRD{},
		&crd.Kustomization{},
		&rbac.ManagerRoleUpdater{Chart: s.chrt},
		&samples.CustomResource{ChartPath: chartPath, Chart: s.chrt},
	); err != nil {
		return fmt.Errorf("error scaffolding APIs: %w", err)
	}

	return nil
}
