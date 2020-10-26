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

package v2

import (
	"github.com/operator-framework/operator-sdk/internal/plugins"

	"sigs.k8s.io/kubebuilder/pkg/plugin"
	kbgov2 "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

// Plugin name/version used in this file will also be used in phase 2 plugins when we can
// pipe kubebuilder's init/create api output into our scaffold modification
// plugins. In phase 1 we wrap kubebuilder's Go plugin to have the same effect.

const (
	pluginName = "go" + plugins.DefaultNameQualifier
)

var (
	pluginVersion   = plugin.Version{Number: 2, Stage: plugin.AlphaStage}
	pluginConfigKey = plugin.Key(pluginName, pluginVersion.String())
)

var (
	_ plugin.Base                      = Plugin{}
	_ plugin.InitPluginGetter          = Plugin{}
	_ plugin.CreateAPIPluginGetter     = Plugin{}
	_ plugin.CreateWebhookPluginGetter = Plugin{}
)

// Plugin defines an Operator SDK Go scaffold and CLI plugin. Its current purpose is to
// add operator-framework features to the base kubebuilder Go scaffold and CLI.
type Plugin struct{}

func (Plugin) Name() string            { return (kbgov2.Plugin{}).Name() }
func (Plugin) Version() plugin.Version { return (kbgov2.Plugin{}).Version() }
func (Plugin) SupportedProjectVersions() []string {
	return (kbgov2.Plugin{}).SupportedProjectVersions()
}

func (p Plugin) GetInitPlugin() plugin.Init {
	return &initPlugin{
		Init: (kbgov2.Plugin{}).GetInitPlugin(),
	}
}

func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI {
	return &createAPIPlugin{
		CreateAPI: (kbgov2.Plugin{}).GetCreateAPIPlugin(),
	}
}

func (p Plugin) GetCreateWebhookPlugin() plugin.CreateWebhook {
	return (kbgov2.Plugin{}).GetCreateWebhookPlugin()
}
