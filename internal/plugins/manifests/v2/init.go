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
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// runInit runs the manifests SDK phase 2 plugin.
func runInit(cfg config.Config, fs machinery.Filesystem) error {

	if err := newInitScaffolder(cfg).scaffold(fs); err != nil {
		return err
	}

	return nil
}

type initScaffolder struct {
	config config.Config
}

func newInitScaffolder(config config.Config) *initScaffolder {
	return &initScaffolder{
		config: config,
	}
}

func (s *initScaffolder) scaffold(fs machinery.Filesystem) error {

	// Only Go operator types support webhooks right now.
	operatorType := projutil.PluginKeyToOperatorType(s.config.GetPluginChain())

	err := machinery.NewScaffold(fs, machinery.WithConfig(s.config)).Execute(
		&Kustomization{SupportsWebhooks: operatorType == projutil.OperatorTypeGo},
	)
	if err != nil {
		return fmt.Errorf("error scaffolding manifests: %v", err)
	}

	return nil
}
