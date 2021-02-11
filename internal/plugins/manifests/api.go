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
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
)

// RunCreateAPI runs the manifests SDK phase 2 plugin.
func RunCreateAPI(cfg config.Config, gvk resource.GVK) error {
	// Only run these if project version is v3.
	isV3 := cfg.GetVersion().Compare(cfgv3.Version) == 0
	if !isV3 {
		return nil
	}

	if err := newAPIScaffolder(cfg, gvk).scaffold(); err != nil {
		return err
	}

	return nil
}

type apiScaffolder struct {
	config config.Config
	gvk    resource.GVK
}

func newAPIScaffolder(config config.Config, gvk resource.GVK) *apiScaffolder {
	return &apiScaffolder{
		config: config,
		gvk:    gvk,
	}
}

func (s *apiScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
	)
}

func (s *apiScaffolder) scaffold() error {
	var builders []file.Builder
	// If the gvk is non-empty, add relevant builders.
	if s.gvk.Group != "" || s.gvk.Version != "" || s.gvk.Kind != "" {
		builders = append(builders, &kustomization{GroupVersionKind: s.gvk})
	}

	err := machinery.NewScaffold().Execute(s.newUniverse(), builders...)
	if err != nil {
		return fmt.Errorf("error scaffolding manifests: %v", err)
	}

	return nil
}
