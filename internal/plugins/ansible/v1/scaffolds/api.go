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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"

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

type CreateOptions struct {
	GVK schema.GroupVersionKind
	// CRDVersion is the version of the `apiextensions.k8s.io` API which will be used to generate the CRD.
	CRDVersion       string
	GeneratePlaybook bool
	GenerateRole     bool
}

type apiScaffolder struct {
	config config.Config
	opts   CreateOptions
}

// NewCreateAPIScaffolder returns a new Scaffolder for project initialization operations
func NewCreateAPIScaffolder(config config.Config, opts CreateOptions) cmdutil.Scaffolder {
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
	resourceOptions := &golang.Options{}
	resourceOptions.DoAPI = true
	resourceOptions.Group = s.opts.GVK.Group
	resourceOptions.Version = s.opts.GVK.Version
	resourceOptions.Kind = s.opts.GVK.Kind

	//todo(camilamacedo86): replace the options by kubernetes-sigs/kubebuilder#1974
	if err := resourceOptions.Validate(); err != nil {
		return err
	}

	// Check that resource doesn't exist
	if s.config.HasResource(resourceOptions.GVK()) {
		return errors.New("the API resource already exists")
	}

	// Check that the provided group can be added to the project
	if !s.config.IsMultiGroup() && s.config.ResourcesLength() != 0 && !s.config.HasGroup(resourceOptions.Group) {
		return errors.New("multiple groups are not allowed by default, to enable multi-group set 'multigroup: true' in your PROJECT file")
	}

	resource := resourceOptions.NewResource(s.config)

	resource.Domain = s.config.GetDomain()

	// remove the path since is not a Golang project
	resource.Path = ""

	// add the resource API info to complain with project-version=3
	// todo: ensure that this information is properly returned from
	// resource.newResource in upstream ( see kubernetes-sigs/kubebuilder#1974)
	// and then, remove it.
	resource.API.Namespaced = true
	resource.API.CRDVersion = s.opts.CRDVersion

	if err := s.config.UpdateResource(resource); err != nil {
		return err
	}

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
		s.newUniverse(&resource),
		createAPITemplates...,
	)
}
