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
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
	scorecardv2 "github.com/operator-framework/operator-sdk/internal/plugins/scorecard/v2"
)

type initSubcommand struct {
	plugin.InitSubcommand

	config config.Config
}

var _ plugin.InitSubcommand = &initSubcommand{}

func (p *initSubcommand) UpdateContext(ctx *plugin.Context) { p.InitSubcommand.UpdateContext(ctx) }
func (p *initSubcommand) BindFlags(fs *pflag.FlagSet)       { p.InitSubcommand.BindFlags(fs) }

func (p *initSubcommand) InjectConfig(c config.Config) {
	p.InitSubcommand.InjectConfig(c)
	p.config = c
}

func (p *initSubcommand) Run(fs machinery.Filesystem) error {
	if err := p.InitSubcommand.Run(fs); err != nil {
		return err
	}

	// Run SDK phase 2 plugins.
	if err := p.runPhase2(); err != nil {
		return err
	}

	return nil
}

// SDK phase 2 plugins.
func (p *initSubcommand) runPhase2() error {
	if err := manifestsv2.RunInit(p.config); err != nil {
		return err
	}
	if err := scorecardv2.RunInit(p.config); err != nil {
		return err
	}
	return nil
}
