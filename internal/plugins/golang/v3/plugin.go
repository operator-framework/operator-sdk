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
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
	kbgov3 "sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3"
)

var (
	_ plugin.Plugin = Plugin{}
	_ plugin.Full   = Plugin{}
)

// Plugin defines an Operator SDK Go scaffold and CLI plugin. Its current purpose is to
// add operator-framework features to the base kubebuilder Go scaffold and CLI.
type Plugin struct{}

func (Plugin) Name() string            { return (kbgov3.Plugin{}).Name() }
func (Plugin) Version() plugin.Version { return (kbgov3.Plugin{}).Version() }
func (Plugin) SupportedProjectVersions() []string {
	return (kbgov3.Plugin{}).SupportedProjectVersions()
}

func (p Plugin) GetInitSubcommand() plugin.InitSubcommand {
	return &initSubcommand{
		InitSubcommand: (kbgov3.Plugin{}).GetInitSubcommand(),
	}
}

func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand {
	return &createAPISubcommand{
		CreateAPISubcommand: (kbgov3.Plugin{}).GetCreateAPISubcommand(),
	}
}

func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return (kbgov3.Plugin{}).GetCreateWebhookSubcommand()
}

func (p Plugin) GetEditSubcommand() plugin.EditSubcommand {
	return (kbgov3.Plugin{}).GetEditSubcommand()
}
