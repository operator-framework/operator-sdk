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

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/cmdutil"
	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/constants"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/crd"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/rbac"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/config/samples"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/molecule/mdefault"
	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/playbooks"
	ansibleroles "github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/scaffolds/internal/templates/roles"
)

var _ cmdutil.Scaffolder = &apiScaffolder{}

type apiScaffolder struct {
	config   config.Config
	resource *resource.Resource

	doRole, doPlaybook bool
}

// NewCreateAPIScaffolder returns a new Scaffolder for project initialization operations
func NewCreateAPIScaffolder(config config.Config, res *resource.Resource, doRole, doPlaybook bool) cmdutil.Scaffolder {
	return &apiScaffolder{
		config:     config,
		resource:   res,
		doRole:     doRole,
		doPlaybook: doPlaybook,
	}
}

func (s *apiScaffolder) newUniverse(r *resource.Resource) *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithResource(r),
	)
}

// Scaffold implements Scaffolder
func (s *apiScaffolder) Scaffold() error {
	return s.scaffold()
}

func (s *apiScaffolder) scaffold() error {
	if s.resource == nil {
		return errors.New("resource must not be nil")
	}

	if err := s.config.UpdateResource(*s.resource); err != nil {
		return err
	}

	var createAPITemplates []file.Builder
	createAPITemplates = append(createAPITemplates,
		&rbac.CRDViewerRole{},
		&rbac.CRDEditorRole{},
		&rbac.ManagerRoleUpdater{},

		&crd.CRD{CRDVersion: s.resource.API.CRDVersion},
		&crd.Kustomization{},
		&samples.CR{},
		&templates.WatchesUpdater{
			GeneratePlaybook: s.doPlaybook,
			GenerateRole:     s.doRole,
			PlaybooksDir:     constants.PlaybooksDir,
		},
		&mdefault.ResourceTest{},
	)

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

	return machinery.NewScaffold().Execute(
		s.newUniverse(s.resource),
		createAPITemplates...,
	)
}
