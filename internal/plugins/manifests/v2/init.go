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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2/templates/config/manifests"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/version"
)

// Version of `opm` to download and use for building index images.
// This version's release artifacts *must* contain a binary for multiple arches; certain releases do not.
const opmVersion = "v1.55.0"

const filePath = "Makefile"

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (s *initSubcommand) InjectConfig(c config.Config) error {
	s.config = c

	return nil
}

func (s *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	operatorType := projutil.PluginChainToOperatorType(s.config.GetPluginChain())
	if operatorType == projutil.OperatorTypeUnknown {
		return fmt.Errorf("none of the required plugins detected in %q", s.config.GetPluginChain())
	}

	makefileBytes, err := afero.ReadFile(fs.FS, filePath)
	if err != nil {
		return err
	}

	// Prepend bundle variables.
	projectName := s.config.GetProjectName()
	if projectName == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current directory: %w", err)
		}
		projectName = strings.ToLower(filepath.Base(dir))
	}
	makefileBytes = append([]byte(fmt.Sprintf(makefileBundleVarFragment, s.config.GetDomain(), projectName, strings.Trim(version.Version, "+git"))), makefileBytes...)

	// Append SDK recipes.
	switch operatorType {
	case projutil.OperatorTypeGo:
		makefileBytes = append(makefileBytes, []byte(makefileSDKFragmentGo)...)
	default:
		makefileBytes = append(makefileBytes, []byte(makefileSDKFragmentNonGo)...)
	}

	// Append bundle recipes.
	switch operatorType {
	case projutil.OperatorTypeGo:
		makefileBytes = append(makefileBytes, []byte(makefileBundleFragmentGo)...)
	default:
		makefileBytes = append(makefileBytes, []byte(makefileBundleFragmentNonGo)...)
	}
	makefileBytes = append(makefileBytes, []byte(makefileBundleBuildPushFragment)...)

	// Append catalog recipes.
	switch operatorType {
	case projutil.OperatorTypeGo:
		makefileBytes = append(makefileBytes, []byte(fmt.Sprintf(makefileOPMFragmentGo, opmVersion))...)
	default:
		makefileBytes = append(makefileBytes, []byte(fmt.Sprintf(makefileOPMFragmentNonGo, opmVersion))...)
	}
	makefileBytes = append(makefileBytes, []byte(makefileCatalogBuildFragment)...)

	var mode os.FileMode = 0644
	if info, err := fs.FS.Stat(filePath); err == nil {
		mode = info.Mode()
	}
	if err := afero.WriteFile(fs.FS, filePath, makefileBytes, mode); err != nil {
		return fmt.Errorf("error updating Makefile: %w", err)
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		// NOTE: kubebuilder's default permissions are only for root users
		machinery.WithDirectoryPermissions(0755),
		machinery.WithFilePermissions(0644),
		machinery.WithConfig(s.config),
	)

	if err := scaffold.Execute(
		&manifests.Kustomization{
			SupportsKustomizeV4: HasSupportForKustomizeV4(s.config),
			SupportsWebhooks:    operatorType == projutil.OperatorTypeGo},
	); err != nil {
		return fmt.Errorf("error scaffolding manifests: %w", err)
	}

	if err := s.config.EncodePluginConfig(pluginKey, Config{}); err != nil && !errors.As(err, &config.UnsupportedFieldError{}) {
		return err
	}

	// TODO: remove this when we bump kubebuilder to v5.x
	// Not adopt changes introduced by mistake in the default Makefile of kubebuilder v4.x.
	if operatorType == projutil.OperatorTypeGo {
		err = util.ReplaceInFile("Makefile", "$(KIND) create cluster --name $(KIND_CLUSTER)", makefileTestFix)
		if err != nil {
			return fmt.Errorf("error replacing Makefile: %w", err)
		}
	}

	return nil
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
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
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

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# %[1]s/%[2]s-bundle:$VERSION and %[1]s/%[2]s-catalog:$VERSION.
IMAGE_TAG_BASE ?= %[1]s/%[2]s

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Set the Operator SDK version to use. By default, what is installed on the system is used.
# This is useful for CI or a project to utilize a specific version of the operator-sdk toolkit.
OPERATOR_SDK_VERSION ?= %[3]s
`

	makefileSDKFragmentGo = `
.PHONY: operator-sdk
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (, $(shell which operator-sdk 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which operator-sdk)
endif
endif
`

	makefileSDKFragmentNonGo = `
.PHONY: operator-sdk
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (, $(shell which operator-sdk 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$(OS)_$(ARCH) ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which operator-sdk)
endif
endif
`

	makefileBundleFragmentGo = `
.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(OPERATOR_SDK) bundle validate ./bundle
`

	makefileBundleFragmentNonGo = `

.PHONY: bundle
bundle: kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(OPERATOR_SDK) bundle validate ./bundle
`

	makefileBundleBuildPushFragment = `
.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	$(CONTAINER_TOOL) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)
`

	makefileOPMFragmentGo = `
.PHONY: opm
OPM = $(LOCALBIN)/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/%[1]s/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif
`

	makefileOPMFragmentNonGo = `
.PHONY: opm
OPM = $(LOCALBIN)/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/%[1]s/$(OS)-$(ARCH)-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif
`

	makefileCatalogBuildFragment = `
# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool $(CONTAINER_TOOL) --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)
`

	// TODO: remove it when we bump kubebuilder to v4.x
	// We will not adopt this change since it did not work and was a bug introduced in the
	// default Makefile of kubebuilder v4.x.
	makefileTestFix = `@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac`
)
