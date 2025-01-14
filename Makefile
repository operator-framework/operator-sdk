SHELL = /bin/bash

# IMAGE_VERSION represents the helm-operator, and scorecard subproject versions.
# This value must be updated to the release tag of the most recent release, a change that must
# occur in the release commit. IMAGE_VERSION will be removed once each subproject that uses this
# version is moved to a separate repo and release process.
export IMAGE_VERSION = v1.39.1
# Build-time variables to inject into binaries
export SIMPLE_VERSION = $(shell (test "$(shell git describe --tags)" = "$(shell git describe --tags --abbrev=0)" && echo $(shell git describe --tags)) || echo $(shell git describe --tags --abbrev=0)+git)
export GIT_VERSION = $(shell git describe --dirty --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export K8S_VERSION = 1.31.0

# Build settings
export TOOLS_DIR = tools/bin
export SCRIPTS_DIR = tools/scripts
GO := $(shell type -P go)
REPO = $(shell $(GO) list -m)
BUILD_DIR = build
GO_ASMFLAGS = -asmflags "all=-trimpath=$(shell dirname $(PWD))"
GO_GCFLAGS = -gcflags "all=-trimpath=$(shell dirname $(PWD))"
GO_BUILD_ARGS = \
  $(GO_GCFLAGS) $(GO_ASMFLAGS) \
  -ldflags " \
    -X '$(REPO)/internal/version.Version=$(SIMPLE_VERSION)' \
    -X '$(REPO)/internal/version.GitVersion=$(GIT_VERSION)' \
    -X '$(REPO)/internal/version.GitCommit=$(GIT_COMMIT)' \
    -X '$(REPO)/internal/version.KubernetesVersion=v$(K8S_VERSION)' \
    -X '$(REPO)/internal/version.ImageVersion=$(IMAGE_VERSION)' \
  " \

export GO111MODULE = on
export CGO_ENABLED = 0
export PATH := $(PWD)/$(BUILD_DIR):$(PWD)/$(TOOLS_DIR):$(PATH)

##@ Development

.PHONY: generate
generate: build # Generate CLI docs and samples
	rm -rf testdata
	$(GO) run ./hack/generate/cncf-maintainers/main.go
	$(GO) run ./hack/generate/cli-doc/gen-cli-doc.go
	$(GO) run ./hack/generate/samples/generate_testdata.go
	$(GO) generate ./...

.PHONY: bindata
OLM_VERSIONS = 0.26.0 0.27.0 0.28.0
bindata: ## Update project bindata
	./hack/generate/olm_bindata.sh $(OLM_VERSIONS)
	$(MAKE) fix

.PHONY: fix
fix: ## Fixup files in the repo.
	$(GO) mod tidy
	$(GO) fmt ./...
	make setup-lint
	$(TOOLS_DIR)/golangci-lint run --fix

.PHONY: setup-lint
setup-lint: ## Setup the lint
	$(SCRIPTS_DIR)/fetch golangci-lint 1.62.2

.PHONY: lint
lint: setup-lint ## Run the lint check
	$(TOOLS_DIR)/golangci-lint run


.PHONY: clean
clean: ## Cleanup build artifacts and tool binaries.
	rm -rf $(BUILD_DIR) dist $(TOOLS_DIR)

##@ Build

.PHONY: install
install: ## Install operator-sdk and helm-operator.
	@if [ -z "$(GOBIN)" ]; then \
		echo "Error: GOBIN is not set"; \
		exit 1; \
	fi
	$(GO) install $(GO_BUILD_ARGS) ./cmd/{operator-sdk,helm-operator}

.PHONY: build
build: ## Build operator-sdk and helm-operator.
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GO_BUILD_ARGS) -o $(BUILD_DIR) ./cmd/{operator-sdk,helm-operator}

.PHONY: build/operator-sdk build/helm-operator
build/operator-sdk build/helm-operator:
	$(GO) build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/$(@F) ./cmd/$(@F)

# Build scorecard binaries.
.PHONY: build/scorecard-test build/scorecard-test-kuttl build/custom-scorecard-tests
build/scorecard-test build/scorecard-test-kuttl build/custom-scorecard-tests:
	$(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -o $(BUILD_DIR)/$(@F) ./images/$(@F)

##@ Dev image build

# Convenience wrapper for building all remotely hosted images.
.PHONY: image-build
IMAGE_TARGET_LIST = operator-sdk helm-operator scorecard-test scorecard-test-kuttl
image-build: $(foreach i,$(IMAGE_TARGET_LIST),image/$(i)) ## Build all images.


# Build an image.
BUILD_IMAGE_REPO = quay.io/operator-framework
# When running in a terminal, this will be false. If true (ex. CI), print plain progress.
ifneq ($(shell test -t 0; echo $$?),0)
DOCKER_PROGRESS = --progress plain
endif
image/%: export DOCKER_CLI_EXPERIMENTAL = enabled
image/%:
	docker buildx build $(DOCKER_PROGRESS) -t $(BUILD_IMAGE_REPO)/$*:dev -f ./images/$*/Dockerfile --load .

image-base/%: export DOCKER_CLI_EXPERIMENTAL = enabled
image-base/%:
	docker buildx build $(DOCKER_PROGRESS) -t $(BUILD_IMAGE_REPO)/$*-base:dev -f ./images/$*/base.Dockerfile --load images/$*
##@ Release

.PHONY: release
release: ## Release target. See 'make -f release/Makefile help' for more information.
	$(MAKE) -f release/Makefile $@

.PHONY: prerelease
prerelease: generate ## Write release commit changes. See 'make -f release/Makefile help' for more information.
ifneq ($(RELEASE_VERSION),$(IMAGE_VERSION))
	$(error "IMAGE_VERSION "$(IMAGE_VERSION)" must be updated to match RELEASE_VERSION "$(RELEASE_VERSION)" prior to creating a release commit")
endif
	$(MAKE) -f release/Makefile $@

.PHONY: tag
tag: ## Tag a release commit. See 'make -f release/Makefile help' for more information.
	$(MAKE) -f release/Makefile $@

##@ Test

.PHONY: test-all
test-all: test-static test-e2e ## Run all tests

.PHONY: test-static
test-static: test-sanity test-unit test-docs ## Run all non-cluster-based tests

.PHONY: test-sanity
test-sanity: generate fix ## Test repo formatting, linting, etc.
	git diff --exit-code # fast-fail if generate or fix produced changes
	./hack/check-license.sh
	./hack/check-error-log-msg-format.sh
	$(GO) vet ./...
	make setup-lint
	make lint
	git diff --exit-code # diff again to ensure other checks don't change repo

.PHONY: test-docs
test-docs: ## Test doc links
	$(GO) run ./release/changelog/gen-changelog.go -validate-only
	git submodule update --init --recursive website/
	./hack/check-links.sh

.PHONY: test-unit
TEST_PKGS = $(shell $(GO) list ./... | grep -v -E 'github.com/operator-framework/operator-sdk/test/')
test-unit: ## Run unit tests
	$(GO) test -coverprofile=coverage.out -covermode=count -short $(TEST_PKGS)

e2e_tests := test-e2e-go test-e2e-helm test-e2e-integration
e2e_targets := test-e2e $(e2e_tests)
.PHONY: $(e2e_targets)

.PHONY: test-e2e-setup
export KIND_CLUSTER := osdk-test

KUBEBUILDER_ASSETS = $(PWD)/$(shell $(GO) install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && $(shell $(GO) env GOPATH)/bin/setup-envtest use $(K8S_VERSION) --bin-dir tools/bin/ -p path)
test-e2e-setup:: build dev-install cluster-create

.PHONY: cluster-create
cluster-create::
	[[ "`$(TOOLS_DIR)/kind get clusters`" =~ "$(KIND_CLUSTER)" ]] || $(TOOLS_DIR)/kind create cluster --image="kindest/node:v$(K8S_VERSION)" --name $(KIND_CLUSTER)

.PHONY: dev-install
dev-install::
	$(SCRIPTS_DIR)/fetch kind 0.24.0
	$(SCRIPTS_DIR)/fetch kubectl $(K8S_VERSION) # Install kubectl AFTER envtest because envtest includes its own kubectl binary

.PHONY: test-e2e-teardown
test-e2e-teardown:
	$(SCRIPTS_DIR)/fetch kind 0.24.0
	$(TOOLS_DIR)/kind delete cluster --name $(KIND_CLUSTER)
	rm -f $(KUBECONFIG)

# Double colon rules allow repeated rule declarations.
# Repeated rules are executed in the order they appear.
$(e2e_targets):: test-e2e-setup image/scorecard-test
test-e2e:: $(e2e_tests) ## Run e2e tests

test-e2e-sample-go:: dev-install cluster-create ## Run Memcached Operator Sample e2e tests
	make test-e2e -C ./testdata/go/v4/memcached-operator/
test-e2e-go:: image/custom-scorecard-tests ## Run Go e2e tests
	$(GO) test ./test/e2e/go -v -ginkgo.v
test-e2e-helm:: image/helm-operator ## Run Helm e2e tests
	$(GO) test ./test/e2e/helm -v -ginkgo.v
test-e2e-integration:: ## Run integration tests
	$(GO) test ./test/integration -v -ginkgo.v
	./hack/tests/subcommand-olm-install.sh

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show this help screen.
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
