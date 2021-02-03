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

package v3

import (
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
)

type createAPISubcommand struct {
	plugin.CreateAPISubcommand

	config config.Config
}

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

func (p *createAPISubcommand) UpdateContext(ctx *plugin.Context) {
	p.CreateAPISubcommand.UpdateContext(ctx)
}
func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) { p.CreateAPISubcommand.BindFlags(fs) }

func (p *createAPISubcommand) InjectConfig(c config.Config) {
	p.CreateAPISubcommand.InjectConfig(c)
	p.config = c
}

func (p *createAPISubcommand) Run() error {
	// Run() may add a new resource to the config, so we can compare resources before/after to get the new resource.
	oldResources, err := p.config.GetResources()
	if err != nil {
		return err
	}

	if err := p.CreateAPISubcommand.Run(); err != nil {
		return err
	}

	// Find the new resource. Here we shouldn't worry about checking if one was found,
	// since downstream plugins will do so.
	newResources, err := p.config.GetResources()
	if err != nil {
		return err
	}
	var newResource resource.Resource
	for _, newR := range newResources {
		newResource = newR
		for _, oldR := range oldResources {
			if !oldR.GVK.IsEqualTo(newR.GVK) {
				newResource = newR
				break
			}
		}
	}

	// Run SDK phase 2 plugins.
	return p.runPhase2(newResource.GVK)
}

// SDK phase 2 plugins.
func (p *createAPISubcommand) runPhase2(gvk resource.GVK) error {
	return manifestsv2.RunCreateAPI(p.config, gvk)
}
