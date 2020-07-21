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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates/crd"
)

var _ scaffold.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config *config.Config
	opts   chartutil.CreateOptions
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(config *config.Config, opts chartutil.CreateOptions) scaffold.Scaffolder {
	return &apiScaffolder{
		config: config,
		opts:   opts,
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
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}
	r, chrt, err := chartutil.CreateChart(projectDir, s.opts)
	if err != nil {
		return err
	}

	// Check that resource doesn't exist
	if s.config.HasResource(r.GVK()) {
		return errors.New("the API resource already exists")
	}
	// Check that the provided group can be added to the project
	if !s.config.MultiGroup && len(s.config.Resources) != 0 && !s.config.HasGroup(r.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group visit %s",
			"kubebuilder.io/migration/multi-group.html")
	}

	res := r.NewResource(s.config, true)
	s.config.AddResource(res.GVK())

	chartPath := filepath.Join(chartutil.HelmChartsDir, chrt.Metadata.Name)
	if err := machinery.NewScaffold().Execute(
		s.newUniverse(res),
		&templates.CRDSample{ChartPath: chartPath, Chart: chrt},
		&templates.CRDEditorRole{},
		&templates.CRDViewerRole{},
		&templates.WatchesUpdater{ChartPath: chartPath},
		&crd.CRD{CRDVersion: s.opts.CRDVersion},
	); err != nil {
		return fmt.Errorf("error scaffolding APIs: %v", err)
	}

	if err := machinery.NewScaffold().Execute(
		s.newUniverse(res),
		&crd.Kustomization{},
	); err != nil {
		return fmt.Errorf("error scaffolding kustomization: %v", err)
	}

	if err := machinery.NewScaffold().Execute(
		s.newUniverse(res),
		&templates.Role{},
		&templates.RoleUpdater{Chart: chrt},
	); err != nil {
		return fmt.Errorf("error scaffolding role: %v", err)
	}

	return nil
}
