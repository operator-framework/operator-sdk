// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO: rewrite this when plugins phase 2 is implemented.
package manifests

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

// RunCreateAPI runs the manifests SDK phase 2 plugin.
func RunCreateAPI(cfg config.Config, fs machinery.Filesystem, res resource.Resource) error {
	// Only run these if project version is v3.
	if cfg.GetVersion().Compare(cfgv3.Version) != 0 {
		return nil
	}

	if err := newAPIScaffolder(cfg, res).Scaffold(fs); err != nil {
		return err
	}

	return nil
}

type apiScaffolder struct {
	config   config.Config
	resource resource.Resource
}

func newAPIScaffolder(config config.Config, res resource.Resource) *apiScaffolder {
	return &apiScaffolder{
		config:   config,
		resource: res,
	}
}

func (s *apiScaffolder) Scaffold(fs machinery.Filesystem) error {
	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		// NOTE: kubebuilder's default permissions are only for root users
		machinery.WithDirectoryPermissions(0755),
		machinery.WithFilePermissions(0644),
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	// If the gvk is non-empty, add relevant builders.
	if s.resource.Group != "" || s.resource.Version != "" || s.resource.Kind != "" {
		if err := scaffold.Execute(&kustomization{}); err != nil {
			return fmt.Errorf("error scaffolding manifests: %v", err)
		}
	}

	return nil
}
