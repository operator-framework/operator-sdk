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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/scaffold/kustomize"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

// sampleKustomizationFragment is a template for samples/kustomization.yaml.
const sampleKustomizationFragment = `## This file is auto-generated, do not modify ##
resources:
`

type createAPIPlugin struct {
	plugin.CreateAPI

	config *config.Config
}

var _ plugin.CreateAPI = &createAPIPlugin{}

func (p *createAPIPlugin) UpdateContext(ctx *plugin.Context) { p.CreateAPI.UpdateContext(ctx) }
func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet)       { p.CreateAPI.BindFlags(fs) }

func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.CreateAPI.InjectConfig(c)
	p.config = c
}

func (p *createAPIPlugin) Run() error {
	if err := p.CreateAPI.Run(); err != nil {
		return err
	}

	// Emulate plugins phase 2 behavior by checking the config for this plugin's
	// config object.
	if !hasPluginConfig(p.config) {
		return nil
	}

	return p.run()
}

// SDK plugin-specific scaffolds.
func (p *createAPIPlugin) run() error {

	// Write CR paths to the samples' kustomization file. This file has a
	// "do not modify" comment so it is safe to overwrite.
	samplesKustomization := sampleKustomizationFragment
	for _, gvk := range p.config.Resources {
		samplesKustomization += fmt.Sprintf("- %s\n", makeCRFileName(gvk))
	}
	kpath := filepath.Join("config", "samples")
	if err := kustomize.Write(kpath, samplesKustomization); err != nil {
		return err
	}

	return nil
}

// makeCRFileName returns a Custom Resource example file name in the same format
// as kubebuilder's CreateAPI plugin for a gvk.
func makeCRFileName(gvk config.GVK) string {
	return fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
}
