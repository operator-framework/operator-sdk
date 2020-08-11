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

// TODO: rewrite this when plugins phase 2 is implemented.
package manifests

import (
	"fmt"
	"io/ioutil"

	"sigs.k8s.io/kubebuilder/pkg/model/config"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// RunInit modifies the project scaffolded by kubebuilder's Init plugin.
func RunInit(cfg *config.Config) error {
	// Only run these if project version is v3.
	if !cfg.IsV3() {
		return nil
	}

	// Update the scaffolded Makefile with operator-sdk recipes.
	if err := initUpdateMakefile(cfg, "Makefile"); err != nil {
		return fmt.Errorf("error updating Makefile: %v", err)
	}
	return nil
}

// initUpdateMakefile updates a vanilla kubebuilder Makefile with operator-sdk recipes.
func initUpdateMakefile(cfg *config.Config, filePath string) error {
	makefileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Prepend bundle variables.
	makefileBytes = append([]byte(makefileBundleVarFragment), makefileBytes...)

	// Append bundle recipes.
	operatorType := projutil.PluginKeyToOperatorType(cfg.Layout)
	switch operatorType {
	case projutil.OperatorTypeUnknown:
		return fmt.Errorf("unsupported plugin key %q", cfg.Layout)
	case projutil.OperatorTypeGo:
		makefileBytes = append(makefileBytes, []byte(makefileBundleFragmentGo)...)
	default:
		makefileBytes = append(makefileBytes, []byte(makefileBundleFragmentNonGo)...)
	}

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

	makefileBundleFragmentGo = `
# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
`

	makefileBundleFragmentNonGo = `
# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
`

	makefileBundleBuildFragment = `
# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
`
)
