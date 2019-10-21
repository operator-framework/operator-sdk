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
REPO = github.com/operator-framework/operator-sdk
BUILD_PATH = $(REPO)/cmd/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
SOURCES = $(shell find . -name '*.go' -not -path "*/vendor/*")

ANSIBLE_BASE_IMAGE = quay.io/operator-framework/ansible-operator
HELM_BASE_IMAGE = quay.io/operator-framework/helm-operator
SCORECARD_PROXY_BASE_IMAGE = quay.io/operator-framework/scorecard-proxy

ANSIBLE_IMAGE ?= $(ANSIBLE_BASE_IMAGE)
HELM_IMAGE ?= $(HELM_BASE_IMAGE)
SCORECARD_PROXY_IMAGE ?= $(SCORECARD_PROXY_BASE_IMAGE)

HELM_ARCHES:="amd64" "ppc64le"

export CGO_ENABLED:=0
export GO111MODULE:=on
export GOPROXY?=https://proxy.golang.org/
.DEFAULT_GOAL:=help

.PHONY: help
help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

.PHONY: all
all: format test build/operator-sdk ## Test and Build the Operator SDK

# Code management.
.PHONY: test format tidy clean

format: ## Format the source code
	$(Q)go fmt $(PKGS)

tidy: ## Update dependencies
	$(Q)go mod tidy -v

clean: ## Clean up the build artifacts
	$(Q)rm -rf build

# Build/install/release the SDK.
.PHONY: install release_builds release

install: ## Build & install the Operator SDK CLI binary
	$(Q)go install \
		-gcflags "all=-trimpath=${GOPATH}" \
		-asmflags "all=-trimpath=${GOPATH}" \
		-ldflags " \
			-X '${REPO}/version.GitVersion=${VERSION}' \
			-X '${REPO}/version.GitCommit=${GIT_COMMIT}' \
		" \
		$(BUILD_PATH)

release_builds := \
	build/operator-sdk-$(VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-apple-darwin \
	build/operator-sdk-$(VERSION)-ppc64le-linux-gnu

release: clean $(release_builds) $(release_builds:=.asc) ## Release the Operator SDK

build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
build/operator-sdk-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le

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

# Static tests.
.PHONY: test test-markdown test-sanity test-unit

test: test-unit ## Run the tests

test-markdown test/markdown:
	./hack/ci/marker --root=doc

test-sanity test/sanity: tidy
	./hack/tests/sanity-check.sh

test-unit test/unit: ## Run the unit tests
	$(Q)go test -count=1 -short ./cmd/...
	$(Q)go test -count=1 -short ./pkg/...
	$(Q)go test -count=1 -short ./internal/...

# CI tests.
.PHONY: test-ci

test-ci: test-markdown test-sanity test-unit install test-subcommand test-e2e ## Run the CI test suite

# Subcommand tests.
.PHONY: test-subcommand test-subcommand-local test-subcommand-scorecard test-subcommand-olm-install

test-subcommand: test-subcommand-local test-subcommand-scorecard test-subcommand-olm-install

test-subcommand-local:
	./hack/tests/subcommand.sh

test-subcommand-scorecard:
	./hack/tests/subcommand-scorecard.sh

test-subcommand-olm-install:
	./hack/tests/subcommand-olm-install.sh

# E2E tests.
.PHONY: test-e2e test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm

test-e2e: test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm ## Run the e2e tests

test-e2e-go:
	./hack/tests/e2e-go.sh $(ARGS)

test-e2e-ansible: image-build-ansible
	./hack/tests/e2e-ansible.sh

test-e2e-ansible-molecule: image-build-ansible
	./hack/tests/e2e-ansible-molecule.sh

test-e2e-helm: image-build-helm
	./hack/tests/e2e-helm.sh

# Image scaffold/build/push.
.PHONY: image image-scaffold-ansible image-scaffold-helm image-build image-build-ansible image-build-helm image-push image-push-ansible image-push-helm

image: image-build image-push ## Build and push all images

image-scaffold-ansible:
	go run ./hack/image/ansible/scaffold-ansible-image.go

image-scaffold-helm:
	go run ./hack/image/helm/scaffold-helm-image.go

image-build: image-build-ansible image-build-helm image-build-scorecard-proxy ## Build all images

image-build-ansible: build/operator-sdk-dev-x86_64-linux-gnu
	./hack/image/build-ansible-image.sh $(ANSIBLE_BASE_IMAGE):dev

image-build-helm: build/operator-sdk-dev
	./hack/image/build-helm-image.sh $(HELM_BASE_IMAGE):dev

image-build-scorecard-proxy:
	./hack/image/build-scorecard-proxy-image.sh $(SCORECARD_PROXY_BASE_IMAGE):dev

image-push: image-push-ansible image-push-helm image-push-scorecard-proxy ## Push all images

image-push-ansible:
	./hack/image/push-image-tags.sh $(ANSIBLE_BASE_IMAGE):dev $(ANSIBLE_IMAGE)

image-push-helm:
	./hack/image/push-image-tags.sh $(HELM_BASE_IMAGE):dev $(HELM_IMAGE)-$(shell go env GOARCH)

image-push-helm-multiarch:
	./hack/image/push-manifest-list.sh $(HELM_IMAGE) ${HELM_ARCHES}

image-push-scorecard-proxy:
	./hack/image/push-image-tags.sh $(SCORECARD_PROXY_BASE_IMAGE):dev $(SCORECARD_PROXY_IMAGE)
