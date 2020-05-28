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
	"github.com/operator-framework/operator-sdk/internal/plugins"

	"sigs.k8s.io/kubebuilder/pkg/plugin"
	kbgov3 "sigs.k8s.io/kubebuilder/pkg/plugin/v3"
)

const (
	// These will be used as plugin name/version in phase 2 plugins when we can
	// pipe kubebuilder's init/create api output into our scaffold modification
	// plugins. In phase 1 we wrap kubebuilder's Go plugin to have the same effect.
	pluginName    = "go" + plugins.DefaultNameQualifier
	pluginVersion = "v2.0.0"
)

var pluginConfigKey = plugin.Key(pluginName, pluginVersion)

var (
	_ plugin.Base                      = Plugin{}
	_ plugin.InitPluginGetter          = Plugin{}
	_ plugin.CreateAPIPluginGetter     = Plugin{}
	_ plugin.CreateWebhookPluginGetter = Plugin{}
)

type Plugin struct{}

func (Plugin) Name() string                       { return (kbgov3.Plugin{}).Name() }
func (Plugin) Version() string                    { return (kbgov3.Plugin{}).Version() }
func (Plugin) SupportedProjectVersions() []string { return (kbgov3.Plugin{}).SupportedProjectVersions() }

func (p Plugin) GetInitPlugin() plugin.Init {
	return &initPlugin{
		Init: (kbgov3.Plugin{}).GetInitPlugin(),
	}
}

func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI {
	return &createAPIPlugin{
		CreateAPI: (kbgov3.Plugin{}).GetCreateAPIPlugin(),
	}
}

func (p Plugin) GetCreateWebhookPlugin() plugin.CreateWebhook {
	return (kbgov3.Plugin{}).GetCreateWebhookPlugin()
}
