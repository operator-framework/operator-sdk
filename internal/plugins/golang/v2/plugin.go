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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	kbgov2 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2"

	"github.com/operator-framework/operator-sdk/internal/plugins"
)

// Plugin name/version used in this file will also be used in phase 2 plugins when we can
// pipe kubebuilder's init/create api output into our scaffold modification
// plugins. In phase 1 we wrap kubebuilder's Go plugin to have the same effect.

const (
	pluginName = "go" + plugins.DefaultNameQualifier
)

var (
	pluginVersion   = plugin.Version{Number: 2, Stage: stage.Alpha}
	pluginConfigKey = plugin.Key(pluginName, pluginVersion.String())
)

var (
	_ plugin.Plugin = Plugin{}
	_ plugin.Full   = Plugin{}
)

// Plugin defines an Operator SDK Go scaffold and CLI plugin. Its current purpose is to
// add operator-framework features to the base kubebuilder Go scaffold and CLI.
type Plugin struct{}

func (Plugin) Name() string            { return (kbgov2.Plugin{}).Name() }
func (Plugin) Version() plugin.Version { return (kbgov2.Plugin{}).Version() }
func (Plugin) SupportedProjectVersions() []config.Version {
	return (kbgov2.Plugin{}).SupportedProjectVersions()
}

func (p Plugin) GetInitSubcommand() plugin.InitSubcommand {
	return &initSubcommand{
		InitSubcommand: (kbgov2.Plugin{}).GetInitSubcommand(),
	}
}

func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand {
	return &createAPISubcommand{
		CreateAPISubcommand: (kbgov2.Plugin{}).GetCreateAPISubcommand(),
	}
}

func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return (kbgov2.Plugin{}).GetCreateWebhookSubcommand()
}

func (p Plugin) GetEditSubcommand() plugin.EditSubcommand {
	return (kbgov2.Plugin{}).GetEditSubcommand()
}
