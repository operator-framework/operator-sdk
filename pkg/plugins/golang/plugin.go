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
	"github.com/operator-framework/operator-sdk/pkg/plugins"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	kbgov2 "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

const (
	pluginName    = "go" + plugins.DefaultNameQualifier
	pluginVersion = "v2.0.0"
)

var supportedProjectVersions = []string{config.Version3}

var (
	_ plugin.Base                      = Plugin{}
	_ plugin.InitPluginGetter          = Plugin{}
	_ plugin.CreateAPIPluginGetter     = Plugin{}
	_ plugin.CreateWebhookPluginGetter = Plugin{}
)

type Plugin struct{}

func (Plugin) Name() string                       { return pluginName }
func (Plugin) Version() string                    { return pluginVersion }
func (Plugin) SupportedProjectVersions() []string { return supportedProjectVersions }
func (p Plugin) GetInitPlugin() plugin.Init {
	return &initPlugin{(kbgov2.Plugin{}).GetInitPlugin()}
}
func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI {
	return (kbgov2.Plugin{}).GetCreateAPIPlugin()
}
func (p Plugin) GetCreateWebhookPlugin() plugin.CreateWebhook {
	return (kbgov2.Plugin{}).GetCreateWebhookPlugin()
}
