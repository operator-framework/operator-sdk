// Copyright 2021 The Operator-SDK Authors
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

package v2

import (
	"errors"
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2/templates/config/samples"
	"github.com/operator-framework/operator-sdk/internal/plugins/util"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	config   config.Config
	resource *resource.Resource
}

func (s *createAPISubcommand) InjectConfig(c config.Config) error {
	s.config = c

	// Try to retrieve the plugin config
	if err := s.config.DecodePluginConfig(pluginKey, &Config{}); errors.As(err, &config.PluginKeyNotFoundError{}) {
		if util.UpdateIfLegacyKey(s.config) {
			return nil
		}
		// If we couldn't find it, it means we are not using this plugin, so we skip remaining hooks
		// This scenario could happen if the project was initialized with kubebuilder which doesn't have this plugin
		return plugin.ExitError{
			Plugin: pluginKey,
			Reason: "plugin not used in this project",
		}
	} else if err != nil && !errors.As(err, &config.UnsupportedFieldError{}) {
		return err
	}

	return nil
}

func (s *createAPISubcommand) InjectResource(res *resource.Resource) error {
	s.resource = res

	return nil
}

func (s *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		// NOTE: kubebuilder's default permissions are only for root users
		machinery.WithDirectoryPermissions(0755),
		machinery.WithFilePermissions(0644),
		machinery.WithConfig(s.config),
		machinery.WithResource(s.resource),
	)

	// If the gvk is non-empty
	if s.resource.Group != "" || s.resource.Version != "" || s.resource.Kind != "" {
		if err := scaffold.Execute(&samples.Kustomization{}); err != nil {
			return fmt.Errorf("error scaffolding manifests: %v", err)
		}
	}

	return nil
}
