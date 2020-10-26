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

package v1

import (
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/plugins"
)

const pluginName = "helm" + plugins.DefaultNameQualifier

var (
	supportedProjectVersions = []string{config.Version3Alpha}
	pluginVersion            = plugin.Version{Number: 1}
	pluginKey                = plugin.KeyFor(Plugin{})
)

var (
	_ plugin.Base                  = Plugin{}
	_ plugin.InitPluginGetter      = Plugin{}
	_ plugin.CreateAPIPluginGetter = Plugin{}
)

type Plugin struct {
	initPlugin
	createAPIPlugin
}

func (Plugin) Name() string                           { return pluginName }
func (Plugin) Version() plugin.Version                { return pluginVersion }
func (Plugin) SupportedProjectVersions() []string     { return supportedProjectVersions }
func (p Plugin) GetInitPlugin() plugin.Init           { return &p.initPlugin }
func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI { return &p.createAPIPlugin }
