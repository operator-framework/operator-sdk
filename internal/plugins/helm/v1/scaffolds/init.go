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
	"os"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/kdefault"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/manager"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/prometheus"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/version"
)

const (
	// kustomizeVersion is the sigs.k8s.io/kustomize version to be used in the project
	kustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

// helmOperatorVersion is set to the version of helm-operator at compile-time.
var helmOperatorVersion = version.ImageVersion

var _ scaffold.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config        *config.Config
	apiScaffolder scaffold.Scaffolder
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config *config.Config, apiScaffolder scaffold.Scaffolder) scaffold.Scaffolder {
	return &initScaffolder{
		config:        config,
		apiScaffolder: apiScaffolder,
	}
}

func (s *initScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
	)
}

// Scaffold implements Scaffolder
func (s *initScaffolder) Scaffold() error {
	if err := s.scaffold(); err != nil {
		return err
	}
	if s.apiScaffolder != nil {
		return s.apiScaffolder.Scaffold()
	}
	return nil
}

func (s *initScaffolder) scaffold() error {
	if err := os.MkdirAll(chartutil.HelmChartsDir, 0755); err != nil {
		return err
	}
	return machinery.NewScaffold().Execute(
		s.newUniverse(),
		&templates.Dockerfile{
			HelmOperatorVersion: helmOperatorVersion,
		},
		&templates.GitIgnore{},
		&templates.Makefile{
			Image:               imageName,
			KustomizeVersion:    kustomizeVersion,
			HelmOperatorVersion: helmOperatorVersion,
		},
		&templates.Watches{},
		&rbac.AuthProxyRole{},
		&rbac.AuthProxyRoleBinding{},
		&rbac.AuthProxyService{},
		&rbac.ClientClusterRole{},
		&rbac.Kustomization{},
		&rbac.LeaderElectionRole{},
		&rbac.LeaderElectionRoleBinding{},
		&rbac.ManagerRole{},
		&rbac.ManagerRoleBinding{},
		&manager.Kustomization{},
		&manager.Manager{Image: imageName},
		&prometheus.Kustomization{},
		&prometheus.ServiceMonitor{},
		&kdefault.AuthProxyPatch{},
		&kdefault.Kustomization{},
	)
}
