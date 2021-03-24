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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/plugins"
)

const pluginName = "scorecard" + plugins.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 2}
	supportedProjectVersions = []config.Version{cfgv2.Version, cfgv3.Version}
	pluginKey                = plugin.KeyFor(Plugin{})
)

var (
	_ plugin.Plugin = Plugin{}
	_ plugin.Init   = Plugin{}
)

type Plugin struct {
	initSubcommand
}

func (Plugin) Name() string                               { return pluginName }
func (Plugin) Version() plugin.Version                    { return pluginVersion }
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand { return &p.initSubcommand }

type Config struct{}
