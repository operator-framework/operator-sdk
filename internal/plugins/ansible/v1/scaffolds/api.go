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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"

	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/constants"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/crd"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/molecule/mdefault"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/playbooks"
	ansibleroles "github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/roles"
)

var _ plugins.Scaffolder = &apiScaffolder{}

type apiScaffolder struct {
	fs machinery.Filesystem

	config   config.Config
	resource resource.Resource

	doRole, doPlaybook bool
}

// NewCreateAPIScaffolder returns a new plugins.Scaffolder for project initialization operations
func NewCreateAPIScaffolder(cfg config.Config, res resource.Resource, doRole, doPlaybook bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:     cfg,
		resource:   res,
		doRole:     doRole,
		doPlaybook: doPlaybook,
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

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		// NOTE: kubebuilder's default permissions are only for root users
		machinery.WithDirectoryPermissions(0755),
		machinery.WithFilePermissions(0644),
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	createAPITemplates := []machinery.Builder{
		&rbac.ManagerRoleUpdater{},
		&crd.CRD{},
		&crd.Kustomization{},
		&templates.WatchesUpdater{
			GeneratePlaybook: s.doPlaybook,
			GenerateRole:     s.doRole,
			PlaybooksDir:     constants.PlaybooksDir,
		},
		&mdefault.ResourceTest{},
	}

	if s.doRole {
		createAPITemplates = append(createAPITemplates,
			&ansibleroles.TasksMain{},
			&ansibleroles.DefaultsMain{},
			&ansibleroles.RoleFiles{},
			&ansibleroles.HandlersMain{},
			&ansibleroles.MetaMain{},
			&ansibleroles.RoleTemplates{},
			&ansibleroles.VarsMain{},
			&ansibleroles.Readme{},
		)
	}

	if s.doPlaybook {
		createAPITemplates = append(createAPITemplates,
			&playbooks.Playbook{GenerateRole: s.doRole},
		)
	}

	return scaffold.Execute(createAPITemplates...)
}
