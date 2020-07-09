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
# TODO: change path to ansible repo
REPO = github.com/operator-framework/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
# TODO: change path to ansible repo
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


DEFAULT_IMAGE = quay.io/operator-framework/ansible-operator
IMAGE ?= $(DEFAULT_IMAGE)
ARCHES:="amd64" "ppc64le" "s390x" "arm64"

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

all: format test build/ansible-operator ## Test and Build the ansible-operator

install: ## Install the ansible-operator binary
	$(Q)$(GOARGS) go install $(GO_BUILD_ARGS) ./cmd/ansible-operator

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
	hack/ci/setup-k8s.sh $(K8S_VERSION)

##############################
# Generate Artifacts         #
##############################

##@ Generate

.PHONY: generate gen-cli-doc gen-test-framework gen-changelog

generate: gen-cli-doc  ## Run all non-release generate targets

gen-cli-doc: ## Generate CLI documentation
	./hack/generate/cli-doc/gen-cli-doc.sh

gen-changelog: ## Generate CHANGELOG.md and migration guide updates
	./hack/generate/changelog/gen-changelog.sh

##############################
# Build and Release                    #
##############################

##@ Build and Release

# Build and release ansible-operator.
.PHONY: release_builds release

release_builds := \
	build/ansible-operator-$(VERSION)-aarch64-linux-gnu \
	build/ansible-operator-$(VERSION)-x86_64-linux-gnu \
	build/ansible-operator-$(VERSION)-x86_64-apple-darwin \
	build/ansible-operator-$(VERSION)-ppc64le-linux-gnu \
	build/ansible-operator-$(VERSION)-s390x-linux-gnu

# TODO: add `clean` recipe as the first step
release: $(release_builds) $(release_builds:=.asc) ## Release ansible-operator

build/ansible-operator-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
build/ansible-operator-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/ansible-operator-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/ansible-operator-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
build/ansible-operator-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
build/ansible-operator-%-linux-gnu: GOARGS = GOOS=linux

build/%: $(SOURCES) ## Build the ansible-operator binary
	$(Q)$(GOARGS) go build $(GO_BUILD_ARGS) -o $@ ./cmd/ansible-operator

build/%.asc: ## Create release signatures for ansible-operator release binaries
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

##############################
# Tests                      #
##############################

##@ Tests

# Static tests.
.PHONY: test test-sanity test-unit test-links

test: test-sanity test-unit ## Run static tests

test-sanity: tidy build/ansible-operator lint
	./hack/tests/sanity-check.sh

test-unit: ## Run the unit tests
	$(Q)go test -coverprofile=coverage.out -covermode=count -count=1 -short $(TEST_PKGS)

test-links:
	./hack/check-links.sh

# E2E tests.
.PHONY: test-e2e
test-e2e: image-build ## Run e2e tests
	./hack/tests/e2e-ansible.sh
	# ./hack/tests/e2e-ansible-molecule.sh

# TODO: remove this and uncomment the above line
test-e2e-molecule: image-build ## Run molecule e2e tests
	./hack/tests/e2e-ansible-molecule.sh

##############################
# Images                     #
##############################

##@ Images

# Image scaffold/build/push.
.PHONY: image image-scaffold image-build image-push image-push-multiarch
image: image-build image-push ## Build and push all images

image-scaffold:
	go run ./hack/image/ansible/scaffold-ansible-image.go

image-build: build/ansible-operator-dev-linux-gnu image-scaffold
	./hack/image/build-ansible-image.sh $(DEFAULT_IMAGE):dev

image-push:
	./hack/image/push-image-tags.sh $(DEFAULT_IMAGE):dev $(IMAGE)-$(shell go env GOARCH)

image-push-multiarch:
	./hack/image/push-manifest-list.sh $(IMAGE) $(ARCHES)
