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
	"os"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"

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

	var mode os.FileMode = 0644
	if info, err := os.Stat(filePath); err != nil {
		mode = info.Mode()
	}
	return ioutil.WriteFile(filePath, makefileBytes, mode)
}

// Makefile fragments to add to the base Makefile.
const (
	makefileBundleVarFragment = `# VERSION defines the project version for the bundle. 
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle. 
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle. 
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_IMG defines the image:tag used for the bundle. 
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= controller-bundle:$(VERSION)
`

	makefileBundleFragmentGo = `
# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
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
