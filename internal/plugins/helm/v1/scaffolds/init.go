/*
Copyright 2019 The Kubernetes Authors.

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
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates/manager"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates/metricsauth"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates/prometheus"
	"github.com/operator-framework/operator-sdk/version"
)

const (
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

// HelmOperatorVersion is the version of the helm binary used in the Makefile
var HelmOperatorVersion = strings.TrimSuffix(version.Version, "+git")

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
		&templates.GitIgnore{},
		&templates.AuthProxyRole{},
		&templates.AuthProxyRoleBinding{},
		&metricsauth.AuthProxyPatch{},
		&metricsauth.AuthProxyService{},
		&metricsauth.ClientClusterRole{},
		&manager.Config{Image: imageName},
		&templates.Makefile{
			Image:               imageName,
			KustomizeVersion:    KustomizeVersion,
			HelmOperatorVersion: HelmOperatorVersion,
		},
		&templates.Dockerfile{
			HelmOperatorVersion: HelmOperatorVersion,
		},
		&templates.Kustomize{},
		&templates.ManagerRoleBinding{},
		&templates.LeaderElectionRole{},
		&templates.LeaderElectionRoleBinding{},
		&templates.KustomizeRBAC{},
		&templates.Watches{},
		&manager.Kustomization{},
		&prometheus.Kustomization{},
		&prometheus.ServiceMonitor{},
	)
}
