SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# IMAGE_VERSION represents the ansible-operator, helm-operator, and scorecard subproject versions.
# This value must be updated to the release tag of the most recent release, a change that must
# occur in the release commit. IMAGE_VERSION will be removed once each subproject that uses this
# version is moved to a separate repo and release process.
export IMAGE_VERSION = v1.13.0
# Build-time variables to inject into binaries
export SIMPLE_VERSION = $(shell (test "$(shell git describe)" = "$(shell git describe --abbrev=0)" && echo $(shell git describe)) || echo $(shell git describe --abbrev=0)+git)
export GIT_VERSION = $(shell git describe --dirty --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export K8S_VERSION = 1.21
# TODO: bump this to 1.21, after kubectl `--generator` flag is removed from e2e tests.
export ENVTEST_K8S_VERSION = 1.20.2

# Build settings
export TOOLS_DIR = tools/bin
export SCRIPTS_DIR = tools/scripts
REPO = $(shell go list -m)
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
	go run ./hack/generate/cncf-maintainers/main.go
	go run ./hack/generate/cli-doc/gen-cli-doc.go
	go run ./hack/generate/samples/generate_testdata.go
	go generate ./...

.PHONY: bindata
OLM_VERSIONS = 0.16.1 0.17.0 0.18.2
bindata: ## Update project bindata
	./hack/generate/olm_bindata.sh $(OLM_VERSIONS)
	$(MAKE) fix

.PHONY: fix
fix: ## Fixup files in the repo.
	go mod tidy
	go fmt ./...

.PHONY: clean
clean: ## Cleanup build artifacts and tool binaries.
	rm -rf $(BUILD_DIR) dist $(TOOLS_DIR)

##@ Build

.PHONY: install
install: ## Install operator-sdk, ansible-operator, and helm-operator.
	go install $(GO_BUILD_ARGS) ./cmd/{operator-sdk,ansible-operator,helm-operator}

.PHONY: build
build: ## Build operator-sdk, ansible-operator, and helm-operator.
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR) ./cmd/{operator-sdk,ansible-operator,helm-operator}

.PHONY: build/operator-sdk build/ansible-operator build/helm-operator
build/operator-sdk build/ansible-operator build/helm-operator:
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/$(@F) ./cmd/$(@F)

# Build scorecard binaries.
.PHONY: build/scorecard-test build/scorecard-test-kuttl build/custom-scorecard-tests
build/scorecard-test build/scorecard-test-kuttl build/custom-scorecard-tests:
	go build $(GO_GCFLAGS) $(GO_ASMFLAGS) -o $(BUILD_DIR)/$(@F) ./images/$(@F)

##@ Dev image build

# Convenience wrapper for building all remotely hosted images.
.PHONY: image-build
IMAGE_TARGET_LIST = operator-sdk helm-operator ansible-operator scorecard-test scorecard-test-kuttl
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

##@ Release

.PHONY: release
release: ## Release target. See 'make -f release/Makefile help' for more information.
	$(MAKE) -f release/Makefile $@

.PHONY: prerelease
prerelease: ## Write release commit changes. See 'make -f release/Makefile help' for more information.
ifneq ($(RELEASE_VERSION),$(IMAGE_VERSION))
	$(error "IMAGE_VERSION "$(IMAGE_VERSION)" must be updated to match RELEASE_VERSION "$(RELEASE_VERSION)" prior to creating a release commit")
endif
	$(MAKE) -f release/Makefile $@

.PHONY: tag
tag: ## Tag a release commit. See 'make -f release/Makefile help' for more information.
	$(MAKE) -f release/Makefile $@

##@ Test

TEST_ARTIFACTS := test/artifacts

.PHONY: test-all
test-all: test-static test-e2e test-integration ## Run all tests

.PHONY: test-static
test-static: test-sanity test-unit test-docs ## Run all non-cluster-based tests

.PHONY: test-sanity
test-sanity: generate fix ## Test repo formatting, linting, etc.
	git diff --exit-code # fast-fail if generate or fix produced changes
	./hack/check-license.sh
	./hack/check-error-log-msg-format.sh
	go vet ./...
	$(SCRIPTS_DIR)/fetch golangci-lint 1.31.0 && $(TOOLS_DIR)/golangci-lint run
	git diff --exit-code # diff again to ensure other checks don't change repo

.PHONY: test-docs
test-docs: ## Test doc links
	go run ./release/changelog/gen-changelog.go -validate-only
	git submodule update --init --recursive website/
	./hack/check-links.sh

.PHONY: test-unit
TEST_PKGS = $(shell go list ./... | grep -v -E 'github.com/operator-framework/operator-sdk/test/')
test-unit: ## Run unit tests
	go test -coverprofile=coverage.out -covermode=count -short $(TEST_PKGS)

testbins: build
	$(SCRIPTS_DIR)/fetch kind 0.11.0
	$(SCRIPTS_DIR)/fetch envtest 0.8.3
	$(SCRIPTS_DIR)/fetch kubectl $(K8S_VERSION) # Install kubectl AFTER envtest because envtest includes its own kubectl binary

# e2e tests
e2e_tests := test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm
e2e_targets := test-e2e $(e2e_tests)
.PHONY: $(e2e_targets)

.PHONY: test-e2e-setup
export KIND_CLUSTER := operator-sdk-e2e

export KUBEBUILDER_ASSETS = $(PWD)/$(shell go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && $(shell go env GOPATH)/bin/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir tools/bin/ -p path)
test-e2e-setup: testbins
	[[ "`$(TOOLS_DIR)/kind get clusters`" =~ "$(KIND_CLUSTER)" ]] || $(TOOLS_DIR)/kind create cluster --image="kindest/node:v$(ENVTEST_K8S_VERSION)" --name $(KIND_CLUSTER)

.PHONY: teardown-test-cluster
teardown-test-cluster:
	$(SCRIPTS_DIR)/fetch kind 0.11.0
	$(TOOLS_DIR)/kind delete cluster --name $(KIND_CLUSTER)
	(docker stop kind-registry 2>/dev/null; docker rm --volumes kind-registry 2>/dev/null) || true
	rm -rf $(TEST_ARTIFACTS)

# Double colon rules allow repeated rule declarations.
# Repeated rules are executed in the order they appear.
$(e2e_targets):: test-e2e-setup image/scorecard-test
test-e2e:: $(e2e_tests) ## Run e2e tests

test-e2e-go:: image/custom-scorecard-tests ## Run Go e2e tests
	go test ./test/e2e/go -v -ginkgo.v
test-e2e-ansible:: image/ansible-operator ## Run Ansible e2e tests
	go test -count=1 ./internal/ansible/proxy/...
	go test ./test/e2e/ansible -v -ginkgo.v
test-e2e-ansible-molecule:: image/ansible-operator ## Run molecule-based Ansible e2e tests
	go run ./hack/generate/samples/molecule/generate.go
	./hack/tests/e2e-ansible-molecule.sh
test-e2e-helm:: image/helm-operator ## Run Helm e2e tests
	go test ./test/e2e/helm -v -ginkgo.v

# Integration tests
DOCKER_REGISTRY_NAME := kind-registry
CLUSTER ?= kind

integration_tests := test-integration-olm test-integration-bundle test-integration-pkgman
integration_targets := test-integration $(integration_tests)
.PHONY: $(integration_targets)

$(integration_targets):: testbins
test-integration:: $(integration_tests) ## Run integration tests
test-integration-olm:: test-e2e-setup ## Run OLM integration tests
	./hack/tests/subcommand-olm-install.sh
test-integration-bundle:: FOCUS = bundle
test-integration-pkgman:: FOCUS = packagemanifests
test-integration-bundle test-integration-pkgman:: test-integration-cluster ## Run packagemanifests integration tests.
test-integration-cluster:: ## Set up cluster for integration tests.
ifeq ($(CLUSTER),kind)
	$(SCRIPTS_DIR)/create_kind_cluster -c $(TEST_ARTIFACTS)/registry-certs -k $(K8S_VERSION)
endif
	go test ./test/integration -v -ginkgo.v -ginkgo.focus="$(FOCUS)" $(if ($(CLUSTER),kind),-cert-dir $(shell pwd)/$(TEST_ARTIFACTS)/registry-certs)

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show this help screen.
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
