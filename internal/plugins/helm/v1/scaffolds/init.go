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

	kustomizev2 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/chartutil"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/version"
)

const imageName = "controller:latest"

// helmOperatorVersion is set to the version of helm-operator at compile-time.
var helmOperatorVersion = version.ImageVersion

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	fs machinery.Filesystem

	config config.Config
}

// NewInitScaffolder returns a new plugins.Scaffolder for project initialization operations
func NewInitScaffolder(config config.Config) plugins.Scaffolder {
	return &initScaffolder{
		config: config,
	}
}

// InjectFS implements plugins.Scaffolder
func (s *initScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements Scaffolder
func (s *initScaffolder) Scaffold() error {
	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		// NOTE: kubebuilder's default permissions are only for root users
		machinery.WithDirectoryPermissions(0755),
		machinery.WithFilePermissions(0644),
		machinery.WithConfig(s.config),
	)

	if err := os.MkdirAll(chartutil.HelmChartsDir, 0755); err != nil {
		return err
	}
	return scaffold.Execute(
		&templates.Dockerfile{
			HelmOperatorVersion: helmOperatorVersion,
		},
		&templates.GitIgnore{},
		&templates.Makefile{
			Image:               imageName,
			KustomizeVersion:    kustomizev2.KustomizeVersion,
			HelmOperatorVersion: helmOperatorVersion,
		},
		&templates.Watches{},
		&rbac.ManagerRole{},
	)
}
