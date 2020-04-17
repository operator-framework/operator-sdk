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

package golang

import (
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

type createAPIPlugin struct {
	plugin.CreateAPI

	config *config.Config
}

var _ plugin.CreateAPI = &createAPIPlugin{}

func (p *createAPIPlugin) UpdateContext(ctx *plugin.Context) { p.CreateAPI.UpdateContext(ctx) }
func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet)       { p.CreateAPI.BindFlags(fs) }

func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.CreateAPI.InjectConfig(c)
	p.config = c
}

func (p *createAPIPlugin) Run() error {
	if err := p.CreateAPI.Run(); err != nil {
		return err
	}

	// Emulate plugins phase 2 behavior by checking the config for this plugin's
	// config object.
	if !hasPluginConfig(p.config) {
		return nil
	}

	return p.run()
}

// SDK plugin-specific scaffolds.
func (p *createAPIPlugin) run() error {
	return nil
}
