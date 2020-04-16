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

type initPlugin struct {
	plugin.Init
}

var _ plugin.Init = &initPlugin{}

func (p *initPlugin) UpdateContext(ctx *plugin.Context) { p.Init.UpdateContext(ctx) }
func (p *initPlugin) BindFlags(fs *pflag.FlagSet)       { p.Init.BindFlags(fs) }
func (p *initPlugin) Run() error                        { return p.Init.Run() }
func (p *initPlugin) InjectConfig(c *config.Config) {
	p.Init.InjectConfig(c)
	// Update layout key with this plugin's name.
	c.Layout = plugin.KeyFor(Plugin{})
}
