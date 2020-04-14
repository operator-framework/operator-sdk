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
K8S_VERSION = v1.17.2
REPO = github.com/operator-framework/operator-sdk
BUILD_PATH = $(REPO)/cmd/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
TEST_PKGS = $(shell go list ./... | grep -v -E 'github.com/operator-framework/operator-sdk/(hack/|test/)')
SOURCES = $(shell find . -name '*.go' -not -path "*/vendor/*")

ANSIBLE_BASE_IMAGE = quay.io/operator-framework/ansible-operator
HELM_BASE_IMAGE = quay.io/operator-framework/helm-operator
SCORECARD_PROXY_BASE_IMAGE = quay.io/operator-framework/scorecard-proxy
SCORECARD_TEST_BASE_IMAGE = quay.io/operator-framework/scorecard-test

ANSIBLE_IMAGE ?= $(ANSIBLE_BASE_IMAGE)
HELM_IMAGE ?= $(HELM_BASE_IMAGE)
SCORECARD_PROXY_IMAGE ?= $(SCORECARD_PROXY_BASE_IMAGE)
SCORECARD_TEST_IMAGE ?= $(SCORECARD_TEST_BASE_IMAGE)

ANSIBLE_ARCHES:="amd64" "ppc64le" "s390x" "arm64"
HELM_ARCHES:="amd64" "ppc64le" "s390x" "arm64"
SCORECARD_PROXY_ARCHES:="amd64" "ppc64le" "s390x" "arm64"

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

install: ## Build & install the Operator SDK CLI binary
	$(Q)go install \
		-gcflags "all=-trimpath=${GOPATH}" \
		-asmflags "all=-trimpath=${GOPATH}" \
		-ldflags " \
			-X '${REPO}/version.GitVersion=${VERSION}' \
			-X '${REPO}/version.GitCommit=${GIT_COMMIT}' \
			-X '${REPO}/version.KubernetesVersion=${K8S_VERSION}' \
		" \
		$(BUILD_PATH)

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

gen-cli-doc: ## Generate CLI documentation
	./hack/generate/gen-cli-doc.sh

gen-test-framework: build/operator-sdk ## Run generate commands to update test/test-framework
	./hack/generate/gen-test-framework.sh

generate: gen-cli-doc gen-test-framework  ## Run all generate targets
.PHONY: generate gen-cli-doc gen-test-framework

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
	build/operator-sdk-$(VERSION)-s390x-linux-gnu

release: clean $(release_builds) $(release_builds:=.asc) ## Release the Operator SDK

build/operator-sdk-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/operator-sdk-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
build/operator-sdk-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
build/operator-sdk-%-linux-gnu: GOARGS = GOOS=linux

build/%: $(SOURCES)
	$(Q)$(GOARGS) go build \
		-gcflags "all=-trimpath=${GOPATH}" \
		-asmflags "all=-trimpath=${GOPATH}" \
		-ldflags " \
			-X '${REPO}/version.GitVersion=${VERSION}' \
			-X '${REPO}/version.GitCommit=${GIT_COMMIT}' \
		" \
		-o $@ $(BUILD_PATH)

build/%.asc:
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
.PHONY: image image-scaffold-ansible image-scaffold-helm image-build image-build-ansible image-build-helm image-push image-push-ansible image-push-helm

image: image-build image-push ## Build and push all images

image-scaffold-ansible:
	go run ./hack/image/ansible/scaffold-ansible-image.go

image-scaffold-helm:
	go run ./hack/image/helm/scaffold-helm-image.go

image-build: image-build-ansible image-build-helm image-build-scorecard-proxy ## Build all images

image-build-ansible: build/operator-sdk-dev-linux-gnu
	./hack/image/build-ansible-image.sh $(ANSIBLE_BASE_IMAGE):dev

image-build-helm: build/operator-sdk-dev-linux-gnu
	./hack/image/build-helm-image.sh $(HELM_BASE_IMAGE):dev

image-build-scorecard-proxy:
	./hack/image/build-scorecard-proxy-image.sh $(SCORECARD_PROXY_BASE_IMAGE):dev

image-build-scorecard-test:
	./hack/image/build-scorecard-test-image.sh $(SCORECARD_TEST_BASE_IMAGE):dev

image-push: image-push-ansible image-push-helm image-push-scorecard-proxy ## Push all images

image-push-ansible:
	./hack/image/push-image-tags.sh $(ANSIBLE_BASE_IMAGE):dev $(ANSIBLE_IMAGE)-$(shell go env GOARCH)

image-push-ansible-multiarch:
	./hack/image/push-manifest-list.sh $(ANSIBLE_IMAGE) ${ANSIBLE_ARCHES}

image-push-helm:
	./hack/image/push-image-tags.sh $(HELM_BASE_IMAGE):dev $(HELM_IMAGE)-$(shell go env GOARCH)

image-push-helm-multiarch:
	./hack/image/push-manifest-list.sh $(HELM_IMAGE) ${HELM_ARCHES}

image-push-scorecard-proxy:
	./hack/image/push-image-tags.sh $(SCORECARD_PROXY_BASE_IMAGE):dev $(SCORECARD_PROXY_IMAGE)-$(shell go env GOARCH)

image-push-scorecard-proxy-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_PROXY_IMAGE) ${SCORECARD_PROXY_ARCHES}

image-push-scorecard-test:
	./hack/image/push-image-tags.sh $(SCORECARD_TEST_BASE_IMAGE):dev $(SCORECARD_TEST_IMAGE)-$(shell go env GOARCH)

##############################
# Tests                      #
##############################

##@ Tests

# Static tests.
.PHONY: test test-markdown test-sanity test-unit

test: test-unit ## Run the tests

test-markdown:
	./hack/check-markdown.sh

test-sanity: tidy build/operator-sdk lint
	./hack/tests/sanity-check.sh

test-unit: ## Run the unit tests
	$(Q)go test -coverprofile=coverage.out -covermode=count -count=1 -short $(TEST_PKGS)

# CI tests.
.PHONY: test-ci

test-ci: test-markdown test-sanity test-unit install test-subcommand test-e2e ## Run the CI test suite

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

# E2E and integration tests.
.PHONY: test-e2e test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm test-integration

test-e2e: test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm ## Run the e2e tests

test-e2e-go:
	./hack/tests/e2e-go.sh $(ARGS)

test-e2e-ansible: image-build-ansible
	./hack/tests/e2e-ansible.sh

test-e2e-ansible-molecule: image-build-ansible
	./hack/tests/e2e-ansible-molecule.sh

test-e2e-helm: image-build-helm
	./hack/tests/e2e-helm.sh

test-integration:
	./hack/tests/integration.sh
