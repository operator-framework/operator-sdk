# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
  BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
  Q =
else
  Q = @
endif

VERSION = $(shell git describe --dirty --tags --always)
GIT_COMMIT = $(shell git rev-parse HEAD)
K8S_VERSION = v1.18.2
REPO = github.com/operator-framework/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
TEST_PKGS = $(shell go list ./... | grep -v -E 'github.com/operator-framework/operator-sdk/test/')
SOURCES = $(shell find . -name '*.go' -not -path "*/vendor/*")
# GO_BUILD_ARGS should be set when running 'go build' or 'go install'.
GO_BUILD_ARGS = \
  -gcflags "all=-trimpath=$(shell go env GOPATH)" \
  -asmflags "all=-trimpath=$(shell go env GOPATH)" \
  -ldflags " \
    -X '$(REPO)/version.GitVersion=$(VERSION)' \
    -X '$(REPO)/version.GitCommit=$(GIT_COMMIT)' \
    -X '$(REPO)/version.KubernetesVersion=$(K8S_VERSION)' \
  " \


SCORECARD_PROXY_BASE_IMAGE = quay.io/operator-framework/scorecard-proxy
SCORECARD_TEST_BASE_IMAGE = quay.io/operator-framework/scorecard-test
SCORECARD_TEST_KUTTL_BASE_IMAGE = quay.io/operator-framework/scorecard-test-kuttl

SCORECARD_PROXY_IMAGE ?= $(SCORECARD_PROXY_BASE_IMAGE)
SCORECARD_TEST_IMAGE ?= $(SCORECARD_TEST_BASE_IMAGE)
SCORECARD_TEST_KUTTL_IMAGE ?= $(SCORECARD_TEST_KUTTL_BASE_IMAGE)

SCORECARD_PROXY_ARCHES:="amd64" "ppc64le" "s390x" "arm64"
SCORECARD_TEST_ARCHES:="amd64" "ppc64le" "s390x" "arm64"
SCORECARD_TEST_KUTTL_ARCHES:="amd64" "ppc64le" "s390x" "arm64"

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

install: ## Install the operator-sdk binary
	$(Q)$(GOARGS) go install $(GO_BUILD_ARGS) ./cmd/operator-sdk
	$(MAKE) -f ansible.mk install
	$(MAKE) -f helm.mk install

# Code management.
.PHONY: format tidy clean cli-doc lint

format: ## Format the source code
	$(Q)go fmt $(PKGS)

tidy: ## Update dependencies
	$(Q)go mod tidy -v

clean: ## Clean up the build artifacts
	$(Q)rm -rf build

lint-dev:  ## Run golangci-lint with all checks enabled (development purpose only)
	./hack/tests/check-lint.sh dev

lint-fix: ## Run golangci-lint automatically fix (development purpose only)
	./hack/tests/check-lint.sh fix

lint: ## Run golangci-lint with all checks enabled in the ci
	./hack/tests/check-lint.sh ci

setup-k8s:
	hack/ci/setup-k8s.sh ${K8S_VERSION}

##############################
# Generate Artifacts         #
##############################

##@ Generate

.PHONY: generate gen-cli-doc gen-test-framework gen-changelog

generate: gen-cli-doc gen-test-framework  ## Run all non-release generate targets

gen-cli-doc: ## Generate CLI documentation
	./hack/generate/cli-doc/gen-cli-doc.sh

gen-test-framework: build/operator-sdk ## Run generate commands to update test/test-framework
	./hack/generate/test-framework/gen-test-framework.sh

gen-changelog: ## Generate CHANGELOG.md and migration guide updates
	./hack/generate/changelog/gen-changelog.sh

##############################
# Release                    #
##############################

##@ Release

# Build/install/release the SDK.
.PHONY: release_builds release

release_builds := \
	build/operator-sdk-$(VERSION)-aarch64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-apple-darwin \
	build/operator-sdk-$(VERSION)-ppc64le-linux-gnu \
	build/operator-sdk-$(VERSION)-s390x-linux-gnu \

release: clean $(release_builds) $(release_builds:=.asc) ## Release the Operator SDK and helm/ansible operators
	$(MAKE) -f ansible.mk release
	$(MAKE) -f helm.mk release

build/operator-sdk-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/operator-sdk-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
build/operator-sdk-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
build/operator-sdk-%-linux-gnu: GOARGS = GOOS=linux

# check-build ensures the calling recipe is actually running a build/operator-sdk.* recipe.
# This prevents attempts to build other binaries (ansible/helm operators) with name operator-sdk.
# Use a case statement to avoid shell feature issues.
define check-build =
case $@ in \
build/operator-sdk*) ;; \
*) \
	echo "this recipe is only intended for operator-sdk builds, see {ansible,helm}.Makefile for the correct recipe"; \
	exit 1; \
;; \
esac
endef

build/%: $(SOURCES) ## Build the operator-sdk binary
	@$(check-build)
	$(Q)$(GOARGS) go build $(GO_BUILD_ARGS) -o $@ ./cmd/operator-sdk

build/%.asc: ## Create release signatures for operator-sdk release binaries
	@$(check-build)
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

# Image scaffold/build/push.
.PHONY: image image-build image-push

image: image-build image-push ## Build and push all images

image-build: image-build-ansible image-build-helm image-build-scorecard-proxy image-build-scorecard-test image-build-scorecard-test-kuttl## Build all images

image-push: image-push-ansible image-push-helm image-push-scorecard-proxy image-push-scorecard-test ## Push all images

# Ansible operator image scaffold/build/push.
.PHONY: image-scaffold-ansible image-build-ansible image-push-ansible image-push-ansible-multiarch

image-scaffold-ansible:
	$(MAKE) -f ansible.mk image-scaffold

image-build-ansible:
	$(MAKE) -f ansible.mk image-build

image-push-ansible:
	$(MAKE) -f ansible.mk image-push

image-push-ansible-multiarch:
	$(MAKE) -f ansible.mk image-push-multiarch

# Helm operator image scaffold/build/push.
.PHONY: image-scaffold-helm image-build-helm image-push-helm image-push-helm-multiarch

image-scaffold-helm:
	$(MAKE) -f helm.mk image-scaffold

image-build-helm:
	$(MAKE) -f helm.mk image-build

image-push-helm:
	$(MAKE) -f helm.mk image-push

image-push-helm-multiarch:
	$(MAKE) -f helm.mk image-push-multiarch

# Scorecard proxy image scaffold/build/push.
.PHONY: image-build-scorecard-proxy image-push-scorecard-proxy image-push-scorecard-proxy-multiarch

image-build-scorecard-proxy:
	./hack/image/build-scorecard-proxy-image.sh $(SCORECARD_PROXY_BASE_IMAGE):dev

image-push-scorecard-proxy:
	./hack/image/push-image-tags.sh $(SCORECARD_PROXY_BASE_IMAGE):dev $(SCORECARD_PROXY_IMAGE)-$(shell go env GOARCH)

image-push-scorecard-proxy-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_PROXY_IMAGE) ${SCORECARD_PROXY_ARCHES}

# Scorecard test image scaffold/build/push.
.PHONY: image-build-scorecard-test image-push-scorecard-test image-push-scorecard-test-multiarch

# Scorecard test kuttl image scaffold/build/push.
.PHONY: image-build-scorecard-test-kuttl image-push-scorecard-test-kuttl image-push-scorecard-test-kuttl-multiarch

image-build-scorecard-test:
	./hack/image/build-scorecard-test-image.sh $(SCORECARD_TEST_BASE_IMAGE):dev

image-push-scorecard-test:
	./hack/image/push-image-tags.sh $(SCORECARD_TEST_BASE_IMAGE):dev $(SCORECARD_TEST_IMAGE)-$(shell go env GOARCH)

image-push-scorecard-test-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_TEST_IMAGE) ${SCORECARD_TEST_ARCHES}

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
.PHONY: test-subcommand test-subcommand-local test-subcommand-scorecard test-subcommand-olm-install

test-subcommand: test-subcommand-local test-subcommand-scorecard test-subcommand-olm-install
	./hack/tests/subcommand-bundle.sh
	./hack/tests/subcommand-generate-csv.sh

test-subcommand-local:
	./hack/tests/subcommand.sh

test-subcommand-scorecard:
	./hack/tests/subcommand-scorecard.sh

test-subcommand-olm-install:
	./hack/tests/subcommand-olm-install.sh

# E2E tests.
.PHONY: test-e2e test-e2e-go test-e2e-go-new test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm

test-e2e: test-e2e-go test-e2e-go-new test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm ## Run the e2e tests

test-e2e-go:
	./hack/tests/e2e-go.sh $(ARGS)

test-e2e-go-new:
	K8S_VERSION=$(K8S_VERSION) ./hack/tests/e2e-go-new.sh

test-e2e-ansible:
	$(MAKE) -f ansible.mk test-e2e

test-e2e-ansible-molecule:
	$(MAKE) -f ansible.mk test-e2e-molecule

test-e2e-helm:
	$(MAKE) -f helm.mk test-e2e

# Integration tests.
.PHONY: test-integration

test-integration: ## Run integration tests
	./hack/tests/integration.sh
