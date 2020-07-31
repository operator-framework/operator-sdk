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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"

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

var _ scaffold.Scaffolder = &apiScaffolder{}

type CreateOptions struct {
	GVK schema.GroupVersionKind
	// CRDVersion is the version of the `apiextensions.k8s.io` API which will be used to generate the CRD.
	CRDVersion       string
	GeneratePlaybook bool
	GenerateRole     bool
}

type apiScaffolder struct {
	config *config.Config
	opts   CreateOptions
}

// NewCreateAPIScaffolder returns a new Scaffolder for project initialization operations
func NewCreateAPIScaffolder(config *config.Config, opts CreateOptions) scaffold.Scaffolder {
	return &apiScaffolder{
		config: config,
		opts:   opts,
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

	resourceOptions := resource.Options{
		Group:   s.opts.GVK.Group,
		Version: s.opts.GVK.Version,
		Kind:    s.opts.GVK.Kind,
	}

	if s.config.HasResource(resourceOptions.GVK()) {
		return errors.New("the API resource already exists")
	}

	// Check that the provided group can be added to the project
	if !s.config.MultiGroup && len(s.config.Resources) != 0 && !s.config.HasGroup(resourceOptions.Group) {
		return fmt.Errorf("multiple groups are not allowed by default, to enable multi-group visit %s",
			"kubebuilder.io/migration/multi-group.html")
	}

	resource := resourceOptions.NewResource(s.config, true)
	s.config.AddResource(resource.GVK())

	var createAPITemplates []file.Builder
	createAPITemplates = append(createAPITemplates,
		&rbac.CRDViewerRole{},
		&rbac.CRDEditorRole{},
		&rbac.ManagerRoleUpdater{},

		&crd.CRD{CRDVersion: s.opts.CRDVersion},
		&crd.Kustomization{},
		&samples.CR{},
		&templates.WatchesUpdater{GeneratePlaybook: s.opts.GeneratePlaybook, GenerateRole: s.opts.GenerateRole, PlaybooksDir: constants.PlaybooksDir},
		&mdefault.ResourceTest{},
	)
	if s.opts.GenerateRole {
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

	if s.opts.GeneratePlaybook {
		createAPITemplates = append(createAPITemplates,
			&playbooks.Playbook{GenerateRole: s.opts.GenerateRole})
	}
	return machinery.NewScaffold().Execute(
		s.newUniverse(resource),
		createAPITemplates...,
	)
}
