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
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/kdefault"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/manager"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/prometheus"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/testing"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/testing/pullpolicy"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/molecule/mdefault"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/molecule/mkind"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/playbooks"
	ansibleroles "github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/roles"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
)

const (
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

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
	return machinery.NewScaffold().Execute(
		s.newUniverse(),
		&templates.Dockerfile{},
		&templates.RequirementsYml{},
		&templates.Watches{},

		&rbac.Kustomization{},
		&rbac.ClientClusterRole{},
		&rbac.AuthProxyRole{},
		&rbac.AuthProxyRoleBinding{},
		&rbac.AuthProxyService{},
		&rbac.LeaderElectionRole{},
		&rbac.LeaderElectionRoleBinding{},
		&rbac.ManagerRole{},
		&rbac.RoleBinding{},

		&prometheus.Kustomization{},
		&prometheus.ServiceMonitor{},

		&manager.Manager{Image: imageName},
		&manager.Kustomization{},

		&kdefault.Kustomize{},
		&kdefault.AuthProxyPatch{},

		&templates.Makefile{},
		&ansibleroles.Placeholder{},
		&playbooks.Placeholder{},

		&mdefault.Converge{},
		&mdefault.Create{},
		&mdefault.Destroy{},
		&mdefault.Kustomize{},
		&mdefault.Molecule{},
		&mdefault.Prepare{},
		&mdefault.Verify{},
		&mkind.Converge{},
		&mkind.Create{},
		&mkind.Destroy{},
		&mkind.Molecule{},
		&pullpolicy.AlwaysPullPatch{},
		&pullpolicy.IfNotPresentPullPatch{},
		&pullpolicy.NeverPullPatch{},
		&testing.DebugLogsPatch{},
		&testing.Kustomization{},
		&testing.ManagerImage{},
	)
}
