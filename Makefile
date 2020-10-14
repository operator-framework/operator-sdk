# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
  BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
  Q =
else
  Q = @
endif

SIMPLE_VERSION=$(shell (test "$(shell git describe)" = "$(shell git describe --abbrev=0)" && echo $(shell git describe)) || echo $(shell git describe --abbrev=0)+git)
GIT_VERSION = $(shell git describe --dirty --tags --always)
GIT_COMMIT = $(shell git rev-parse HEAD)
K8S_VERSION = v1.18.2
GOLANGCI_LINT_VER = "1.30.0"
OLM_VERSION = "0.15.1"
REPO = github.com/operator-framework/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
TEST_PKGS = $(shell go list ./... | grep -v -E 'github.com/operator-framework/operator-sdk/test/')
SOURCES = $(shell find . -name '*.go' -not -path "*/vendor/*")
# GO_BUILD_ARGS should be set when running 'go build' or 'go install'.
GO_BUILD_ARGS = \
  -gcflags "all=-trimpath=$(shell go env GOPATH)" \
  -asmflags "all=-trimpath=$(shell go env GOPATH)" \
  -ldflags " \
    -X '$(REPO)/internal/version.Version=$(SIMPLE_VERSION)' \
    -X '$(REPO)/internal/version.GitVersion=$(GIT_VERSION)' \
    -X '$(REPO)/internal/version.GitCommit=$(GIT_COMMIT)' \
    -X '$(REPO)/internal/version.KubernetesVersion=$(K8S_VERSION)' \
  " \

ANSIBLE_BASE_IMAGE = quay.io/operator-framework/ansible-operator
HELM_BASE_IMAGE = quay.io/operator-framework/helm-operator
OPERATOR_SDK_BASE_IMAGE = quay.io/operator-framework/operator-sdk
CUSTOM_SCORECARD_TESTS_BASE_IMAGE = quay.io/operator-framework/custom-scorecard-tests
SCORECARD_TEST_BASE_IMAGE = quay.io/operator-framework/scorecard-test
SCORECARD_TEST_KUTTL_BASE_IMAGE = quay.io/operator-framework/scorecard-test-kuttl

ANSIBLE_IMAGE ?= $(ANSIBLE_BASE_IMAGE)
HELM_IMAGE ?= $(HELM_BASE_IMAGE)
OPERATOR_SDK_IMAGE ?= $(OPERATOR_SDK_BASE_IMAGE)
CUSTOM_SCORECARD_TESTS_IMAGE ?= $(CUSTOM_SCORECARD_TESTS_BASE_IMAGE)
SCORECARD_TEST_IMAGE ?= $(SCORECARD_TEST_BASE_IMAGE)
SCORECARD_TEST_KUTTL_IMAGE ?= $(SCORECARD_TEST_KUTTL_BASE_IMAGE)

ANSIBLE_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
HELM_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
OPERATOR_SDK_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
SCORECARD_TEST_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
SCORECARD_TEST_KUTTL_ARCHES:="amd64" "ppc64le" "arm64"
# the custom scorecard test image is a scorecard example only
CUSTOM_SCORECARD_TESTS_ARCHES:="amd64" "ppc64le" "arm64"

export CGO_ENABLED:=0
.DEFAULT_GOAL:=help

.PHONY: help
help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##############################
# Development                #
##############################

##@ Development

.PHONY: all install

all: format test build/operator-sdk ## Test and Build the Operator SDK

install: ## Install the binaries
	$(Q)$(GOARGS) go install $(GO_BUILD_ARGS) ./cmd/operator-sdk ./cmd/ansible-operator ./cmd/helm-operator


# Code management.
.PHONY: format tidy clean cli-doc lint

format: ## Format the source code
	$(Q)go fmt $(PKGS)

tidy: ## Update dependencies
	$(Q)go mod tidy -v

clean: ## Clean up the build artifacts
	$(Q)rm -rf build

lint: ## Install and run golangci-lint checks
ifneq (${GOLANGCI_LINT_VER}, "$(shell ./bin/golangci-lint --version 2>/dev/null | cut -b 27-32)")
	@echo "golangci-lint missing or not version '${GOLANGCI_LINT_VER}', downloading..."
	curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/v${GOLANGCI_LINT_VER}/install.sh" | sh -s -- -b ./bin "v${GOLANGCI_LINT_VER}"
endif
	./bin/golangci-lint --timeout 5m run

setup-k8s:
	hack/ci/setup-k8s.sh ${K8S_VERSION}

##############################
# Generate Artifacts         #
##############################

##@ Generate

.PHONY: generate cli-doc changelog samples

generate: cli-doc samples bindata ## Run all non-release generate targets

cli-doc: ## Generate CLI documentation
	go run ./hack/generate/cli-doc/gen-cli-doc.go

samples: ## Generate samples
	go run ./hack/generate/samples/generate_all.go

changelog: ## Generate CHANGELOG.md and migration guide updates
	./hack/generate/changelog/gen-changelog.sh

bindata:
	./hack/generate/olm_bindata.sh $(OLM_VERSION)

##############################
# Release                    #
##############################

##@ Release

# Build/install/release the SDK.
.PHONY: release_builds release

release_builds := \
	build/operator-sdk-$(GIT_VERSION)-aarch64-linux-gnu \
	build/operator-sdk-$(GIT_VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(GIT_VERSION)-x86_64-apple-darwin \
	build/operator-sdk-$(GIT_VERSION)-ppc64le-linux-gnu \
	build/operator-sdk-$(GIT_VERSION)-s390x-linux-gnu \
	build/ansible-operator-$(GIT_VERSION)-aarch64-linux-gnu \
	build/ansible-operator-$(GIT_VERSION)-x86_64-linux-gnu \
	build/ansible-operator-$(GIT_VERSION)-x86_64-apple-darwin \
	build/ansible-operator-$(GIT_VERSION)-ppc64le-linux-gnu \
	build/ansible-operator-$(GIT_VERSION)-s390x-linux-gnu \
	build/helm-operator-$(GIT_VERSION)-aarch64-linux-gnu \
	build/helm-operator-$(GIT_VERSION)-x86_64-linux-gnu \
	build/helm-operator-$(GIT_VERSION)-x86_64-apple-darwin \
	build/helm-operator-$(GIT_VERSION)-ppc64le-linux-gnu \
	build/helm-operator-$(GIT_VERSION)-s390x-linux-gnu

release: clean $(release_builds) $(release_builds:=.asc) ## Release the Operator SDK

build/operator-sdk-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/operator-sdk-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
build/operator-sdk-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
build/operator-sdk-%-linux-gnu: GOARGS = GOOS=linux

build/ansible-operator-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
build/ansible-operator-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/ansible-operator-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/ansible-operator-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
build/ansible-operator-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
build/ansible-operator-%-linux-gnu: GOARGS = GOOS=linux

build/helm-operator-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
build/helm-operator-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/helm-operator-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/helm-operator-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
build/helm-operator-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
build/helm-operator-%-linux-gnu: GOARGS = GOOS=linux

build/%: $(SOURCES) ## Build the operator-sdk binary
	$(Q){ \
	cmdpkg=$$(echo $* | sed -E "s/(operator-sdk|ansible-operator|helm-operator).*/\1/"); \
	$(GOARGS) go build $(GO_BUILD_ARGS) -o $@ ./cmd/$$cmdpkg; \
	}

build/%.asc: ## Create release signatures for operator-sdk release binaries
	$(Q){ \
	default_key=$$(gpgconf --list-options gpg | awk -F: '$$1 == "default-key" { gsub(/"/,""); print toupper($$10)}'); \
	git_key=$$(git config --get user.signingkey | awk '{ print toupper($$0) }'); \
	if [ "$${default_key}" = "$${git_key}" ]; then \
		gpg --output $@ --detach-sig build/$*; \
		gpg --verify $@ build/$*; \
	else \
		echo "git and/or gpg are not configured to have default signing key $${default_key}"; \
		exit 1; \
	fi; \
	}

# Image build/push.
.PHONY: image image-build image-push

image: image-build image-push ## Build and push all images

image-build: image-build-ansible image-build-helm image-build-scorecard-test image-build-scorecard-test-kuttl image-build-sdk ## Build all images

image-push: image-push-ansible image-push-helm image-push-scorecard-test image-push-scorecard-test-kuttl image-push-sdk ## Push all images

# Ansible operator image build/push.
.PHONY: image-build-ansible image-push-ansible image-push-ansible-multiarch

image-build-ansible: build/ansible-operator-dev-linux-gnu
	./hack/image/build-ansible-image.sh $(ANSIBLE_BASE_IMAGE):dev

image-push-ansible:
	./hack/image/push-image-tags.sh $(ANSIBLE_BASE_IMAGE):dev $(ANSIBLE_IMAGE)-$(shell go env GOARCH)

image-push-ansible-multiarch:
	./hack/image/push-manifest-list.sh $(ANSIBLE_IMAGE) ${ANSIBLE_ARCHES}

# Helm operator image build/push.
.PHONY: image-build-helm image-push-helm image-push-helm-multiarch

image-build-helm: build/helm-operator-dev-linux-gnu
	./hack/image/build-helm-image.sh $(HELM_BASE_IMAGE):dev

image-push-helm:
	./hack/image/push-image-tags.sh $(HELM_BASE_IMAGE):dev $(HELM_IMAGE)-$(shell go env GOARCH)

image-push-helm-multiarch:
	./hack/image/push-manifest-list.sh $(HELM_IMAGE) ${HELM_ARCHES}

.PHONY: image-build-sdk image-push-sdk image-push-sdk-multiarch

image-build-sdk: build/operator-sdk-dev-linux-gnu
	./hack/image/build-sdk-image.sh $(OPERATOR_SDK_BASE_IMAGE):dev

image-push-sdk:
	./hack/image/push-image-tags.sh $(OPERATOR_SDK_BASE_IMAGE):dev $(OPERATOR_SDK_IMAGE)-$(shell go env GOARCH)

image-push-sdk-multiarch:
	./hack/image/push-manifest-list.sh $(OPERATOR_SDK_IMAGE) ${OPERATOR_SDK_ARCHES}


# Scorecard custom test image build/push.
.PHONY: image-build-custom-scorecard-tests

image-build-custom-scorecard-tests:
	./hack/image/build-custom-scorecard-tests-image.sh $(CUSTOM_SCORECARD_TESTS_BASE_IMAGE):dev

# Scorecard test image build/push.
.PHONY: image-build-scorecard-test image-push-scorecard-test image-push-scorecard-test-multiarch

image-build-scorecard-test:
	./hack/image/build-scorecard-test-image.sh $(SCORECARD_TEST_BASE_IMAGE):dev

image-push-scorecard-test:
	./hack/image/push-image-tags.sh $(SCORECARD_TEST_BASE_IMAGE):dev $(SCORECARD_TEST_IMAGE)-$(shell go env GOARCH)

image-push-scorecard-test-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_TEST_IMAGE) ${SCORECARD_TEST_ARCHES}

# Scorecard test kuttl image build/push.
.PHONY: image-build-scorecard-test-kuttl image-push-scorecard-test-kuttl image-push-scorecard-test-kuttl-multiarch

image-build-scorecard-test-kuttl:
	./hack/image/build-scorecard-test-kuttl-image.sh $(SCORECARD_TEST_KUTTL_BASE_IMAGE):dev

image-push-scorecard-test-kuttl:
	./hack/image/push-image-tags.sh $(SCORECARD_TEST_KUTTL_BASE_IMAGE):dev $(SCORECARD_TEST_KUTTL_IMAGE)-$(shell go env GOARCH)

image-push-scorecard-test-kuttl-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_TEST_KUTTL_IMAGE) ${SCORECARD_TEST_KUTTL_ARCHES}

##############################
# Tests                      #
##############################

##@ Tests

# Static tests.
.PHONY: test test-sanity test-unit

test: test-unit ## Run the tests

test-sanity: tidy build/operator-sdk lint
	./hack/tests/sanity-check.sh

test-unit: ## Run the unit tests
	$(Q)go test -coverprofile=coverage.out -covermode=count -count=1 -short $(TEST_PKGS)

test-links:
	./hack/check-links.sh

# CI tests.
.PHONY: test-ci

test-ci: test-sanity test-unit install test-subcommand test-e2e ## Run the CI test suite

# Subcommand tests.
.PHONY: test-subcommand test-subcommand-olm-install

test-subcommand: test-subcommand-olm-install

test-subcommand-olm-install:
	./hack/tests/subcommand-olm-install.sh

# E2E tests.
.PHONY: test-e2e test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm

test-e2e: test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm ## Run the e2e tests

test-e2e-go: image-build-scorecard-test image-build-custom-scorecard-tests
	./hack/tests/e2e-go.sh

test-e2e-ansible: image-build-ansible image-build-scorecard-test
	./hack/tests/e2e-ansible.sh

test-e2e-ansible-molecule: image-build-ansible
	./hack/tests/e2e-ansible-molecule.sh

test-e2e-helm: image-build-helm image-build-scorecard-test
	./hack/tests/e2e-helm.sh

# Integration tests.
.PHONY: test-integration

test-integration: ## Run integration tests
	./hack/tests/integration.sh
