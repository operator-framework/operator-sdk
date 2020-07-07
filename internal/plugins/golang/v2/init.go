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
	"io/ioutil"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

type initPlugin struct {
	plugin.Init

	config *config.Config
}

var _ plugin.Init = &initPlugin{}

func (p *initPlugin) UpdateContext(ctx *plugin.Context) { p.Init.UpdateContext(ctx) }
func (p *initPlugin) BindFlags(fs *pflag.FlagSet)       { p.Init.BindFlags(fs) }

func (p *initPlugin) InjectConfig(c *config.Config) {
	p.Init.InjectConfig(c)
	p.config = c
}

func (p *initPlugin) Run() error {
	if err := p.Init.Run(); err != nil {
		return err
	}

	// Update the scaffolded Makefile with operator-sdk recipes.
	// TODO: rewrite this when plugins phase 2 is implemented.
	if err := initUpdateMakefile("Makefile"); err != nil {
		return fmt.Errorf("error updating Makefile: %v", err)
	}

	// Update plugin config section with this plugin's configuration.
	cfg := Config{}
	if err := p.config.EncodePluginConfig(pluginConfigKey, cfg); err != nil {
		return fmt.Errorf("error writing plugin config for %s: %v", pluginConfigKey, err)
	}

	return nil
}

// initUpdateMakefile updates a vanilla kubebuilder Makefile with operator-sdk recipes.
func initUpdateMakefile(filePath string) error {
	makefileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Prepend bundle variables.
	makefileBytes = append([]byte(makefileBundleVarFragment), makefileBytes...)
	// Append bundle recipes.
	makefileBytes = append(makefileBytes, []byte(makefileBundleFragment)...)
	makefileBytes = append(makefileBytes, []byte(makefileBundleBuildFragment)...)

	return ioutil.WriteFile(filePath, makefileBytes, 0644)
}

// Makefile fragments to add to the base Makefile.
const (
	makefileBundleVarFragment = `# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)
`

	makefileBundleFragment = `
# Generate bundle manifests and metadata, then validate generated files.
bundle: manifests
	operator-sdk generate kustomize manifests -q
	kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
`

	makefileBundleBuildFragment = `
# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
`
)
