# Build-time variables to inject into binaries
export SIMPLE_VERSION=$(shell (test "$(shell git describe)" = "$(shell git describe --abbrev=0)" && echo $(shell git describe)) || echo $(shell git describe --abbrev=0)+git)
export GIT_VERSION = $(shell git describe --dirty --tags --always)
export GIT_COMMIT = $(shell git rev-parse HEAD)
export K8S_VERSION = 1.18.8

# Build settings
SHELL=/bin/bash
REPO=$(shell go list -m)
BUILD_DIR=$(PWD)/build
GO_BUILD_ARGS = \
  -gcflags "all=-trimpath=$(shell pwd)" \
  -asmflags "all=-trimpath=$(shell pwd)" \
  -ldflags " \
    -X '$(REPO)/internal/version.Version=$(SIMPLE_VERSION)' \
    -X '$(REPO)/internal/version.GitVersion=$(GIT_VERSION)' \
    -X '$(REPO)/internal/version.GitCommit=$(GIT_COMMIT)' \
    -X '$(REPO)/internal/version.KubernetesVersion=v$(K8S_VERSION)' \
  " \

export GO111MODULE = on
export CGO_ENABLED = 0
export PATH := $(BUILD_DIR):$(PWD)/tools/bin:$(PATH)

##@ Development

.PHONY: generate
generate: build # Generate CLI docs and samples
	go run ./hack/generate/cli-doc/gen-cli-doc.go
	go run ./hack/generate/samples/generate_all.go

.PHONY: bindata
OLM_VERSION=0.15.1
bindata: ## Update project bindata
	./hack/generate/olm_bindata.sh $(OLM_VERSION)

.PHONY: fix
fix: ## Fixup files in the repo.
	go mod tidy
	go fmt ./...

.PHONY: clean
clean: ## Cleanup build artifacts and tool binaries.
	rm -rf $(BUILD_DIR) dist tools/bin

##@ Build

.PHONY: build
build: ## Build operator-sdk, ansible-operator, and helm-operator.
	mkdir -p $(BUILD_DIR) && go build $(GO_BUILD_ARGS) -o $(BUILD_DIR) ./cmd/operator-sdk ./cmd/ansible-operator ./cmd/helm-operator

.PHONY: install
GOBIN ?= $(shell go env GOPATH)/bin
install: build ## Install operator-sdk, ansible-operator, and helm-operator in $GOBIN
	install -t $(GOBIN) $(BUILD_DIR)/{operator-sdk,ansible-operator,helm-operator}

.PHONY: image-build-sdk image-push-sdk image-push-sdk-multiarch
OPERATOR_SDK_CI_IMAGE = quay.io/operator-framework/operator-sdk
OPERATOR_SDK_IMAGE ?= $(OPERATOR_SDK_CI_IMAGE)
OPERATOR_SDK_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
image-build-sdk: build
	./hack/image/build-sdk-image.sh $(OPERATOR_SDK_CI_IMAGE):dev
image-push-sdk:
	./hack/image/push-image-tags.sh $(OPERATOR_SDK_CI_IMAGE):dev $(OPERATOR_SDK_IMAGE)-$(shell go env GOARCH)
image-push-sdk-multiarch:
	./hack/image/push-manifest-list.sh $(OPERATOR_SDK_IMAGE) ${OPERATOR_SDK_ARCHES}

.PHONY: image-build-ansible image-push-ansible image-push-ansible-multiarch
ANSIBLE_CI_IMAGE = quay.io/operator-framework/ansible-operator
ANSIBLE_IMAGE ?= $(ANSIBLE_CI_IMAGE)
ANSIBLE_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
image-build-ansible: build
	./hack/image/build-ansible-image.sh $(ANSIBLE_CI_IMAGE):dev
image-push-ansible:
	./hack/image/push-image-tags.sh $(ANSIBLE_CI_IMAGE):dev $(ANSIBLE_IMAGE)-$(shell go env GOARCH)
image-push-ansible-multiarch:
	./hack/image/push-manifest-list.sh $(ANSIBLE_IMAGE) ${ANSIBLE_ARCHES}

.PHONY: image-build-helm image-push-helm image-push-helm-multiarch
HELM_CI_IMAGE = quay.io/operator-framework/helm-operator
HELM_IMAGE ?= $(HELM_CI_IMAGE)
HELM_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
image-build-helm: build
	./hack/image/build-helm-image.sh $(HELM_CI_IMAGE):dev
image-push-helm:
	./hack/image/push-image-tags.sh $(HELM_CI_IMAGE):dev $(HELM_IMAGE)-$(shell go env GOARCH)
image-push-helm-multiarch:
	./hack/image/push-manifest-list.sh $(HELM_IMAGE) ${HELM_ARCHES}

.PHONY: image-build-scorecard-test image-push-scorecard-test image-push-scorecard-test-multiarch
SCORECARD_TEST_CI_IMAGE = quay.io/operator-framework/scorecard-test
SCORECARD_TEST_IMAGE ?= $(SCORECARD_TEST_CI_IMAGE)
SCORECARD_TEST_ARCHES:="amd64" "ppc64le" "arm64" "s390x"
image-build-scorecard-test:
	./hack/image/build-scorecard-test-image.sh $(SCORECARD_TEST_CI_IMAGE):dev
image-push-scorecard-test:
	./hack/image/push-image-tags.sh $(SCORECARD_TEST_CI_IMAGE):dev $(SCORECARD_TEST_IMAGE)-$(shell go env GOARCH)
image-push-scorecard-test-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_TEST_IMAGE) ${SCORECARD_TEST_ARCHES}

.PHONY: image-build-scorecard-test-kuttl image-push-scorecard-test-kuttl image-push-scorecard-test-kuttl-multiarch
SCORECARD_TEST_KUTTL_CI_IMAGE = quay.io/operator-framework/scorecard-test-kuttl
SCORECARD_TEST_KUTTL_IMAGE ?= $(SCORECARD_TEST_KUTTL_CI_IMAGE)
SCORECARD_TEST_KUTTL_ARCHES:="amd64" "ppc64le" "arm64"
image-build-scorecard-test-kuttl:
	./hack/image/build-scorecard-test-kuttl-image.sh $(SCORECARD_TEST_KUTTL_CI_IMAGE):dev
image-push-scorecard-test-kuttl:
	./hack/image/push-image-tags.sh $(SCORECARD_TEST_KUTTL_CI_IMAGE):dev $(SCORECARD_TEST_KUTTL_IMAGE)-$(shell go env GOARCH)
image-push-scorecard-test-kuttl-multiarch:
	./hack/image/push-manifest-list.sh $(SCORECARD_TEST_KUTTL_IMAGE) ${SCORECARD_TEST_KUTTL_ARCHES}

.PHONY: image-build-custom-scorecard-tests
CUSTOM_SCORECARD_TESTS_CI_IMAGE = quay.io/operator-framework/custom-scorecard-tests
CUSTOM_SCORECARD_TESTS_IMAGE ?= $(CUSTOM_SCORECARD_TESTS_CI_IMAGE)
image-build-custom-scorecard-tests:
	./hack/image/build-custom-scorecard-tests-image.sh $(CUSTOM_SCORECARD_TESTS_CI_IMAGE):dev

##@ Test

.PHONY: test-all
test-all: test-static test-e2e ## Run all tests

.PHONY: test-static
test-static: test-sanity test-unit test-links ## Run all non-cluster-based tests

.PHONY: test-sanity
test-sanity: generate fix ## Test repo formatting, linting, etc.
	git diff --exit-code # fast-fail if generate or fix produced changes
	./hack/check-license.sh
	./hack/check-error-log-msg-format.sh
	go run ./hack/generate/changelog/gen-changelog.go -validate-only
	go vet ./...
	./tools/scripts/fetch golangci-lint 1.31.0 && ./tools/bin/golangci-lint run
	git diff --exit-code # diff again to ensure other checks don't change repo

.PHONY: test-links
test-links: ## Test doc links
	./hack/check-links.sh

.PHONY: test-unit
TEST_PKGS = $(shell go list ./... | grep -v -E 'github.com/operator-framework/operator-sdk/test/')
test-unit: ## Run unit tests
	go test -coverprofile=coverage.out -covermode=count -short $(TEST_PKGS)

e2e_tests := test-e2e-go test-e2e-ansible test-e2e-ansible-molecule test-e2e-helm test-e2e-integration
e2e_targets := test-e2e $(e2e_tests)
.PHONY: $(e2e_targets)

.PHONY: test-e2e-setup
export KIND_CLUSTER := operator-sdk-e2e
export KUBECONFIG := $(HOME)/.kube/kind-$(KIND_CLUSTER).config
export KUBEBUILDER_ASSETS := $(PWD)/tools/bin
test-e2e-setup: build
	./tools/scripts/fetch kind 0.9.0
	./tools/scripts/fetch kubectl ${K8S_VERSION}
	./tools/scripts/fetch envtest 0.6.3
	[[ "`./tools/bin/kind get clusters`" =~ "$(KIND_CLUSTER)" ]] || ./tools/bin/kind create cluster --image="kindest/node:v$(K8S_VERSION)" --name $(KIND_CLUSTER)

.PHONY: test-e2e-teardown
test-e2e-teardown:
	./tools/scripts/fetch kind 0.9.0 && ./tools/bin/kind delete cluster --name $(KIND_CLUSTER)
	rm -f $(KUBECONFIG)

# Double colon rules allow repeated rule declarations.
# Repeated rules are exectured in the order they appear.
$(e2e_targets):: test-e2e-setup image-build-scorecard-test

test-e2e:: $(e2e_tests) ## Run e2e tests
test-e2e-go:: image-build-custom-scorecard-tests ## Run Go e2e tests
	go test ./test/e2e-go -v -ginkgo.v
test-e2e-ansible:: image-build-ansible ## Run Ansible e2e tests
	go test -count=1 ./internal/ansible/proxy/...
	go test ./test/e2e-ansible -v -ginkgo.v
test-e2e-ansible-molecule:: image-build-ansible ## Run molecule-based Ansible e2e tests
	./hack/tests/e2e-ansible-molecule.sh
test-e2e-helm:: image-build-helm ## Run Helm e2e tests
	go test ./test/e2e-helm -v -ginkgo.v
test-e2e-integration:: ## Run integration tests
	./hack/tests/integration.sh
	./hack/tests/subcommand-olm-install.sh

##@ Release

.PHONY: changelog
changelog: ## Generate CHANGELOG.md and migration guide updates
	./hack/generate/changelog/gen-changelog.sh

# Build/install/release the SDK.
release_builds := \
	dist/operator-sdk-$(GIT_VERSION)-aarch64-linux-gnu \
	dist/operator-sdk-$(GIT_VERSION)-x86_64-linux-gnu \
	dist/operator-sdk-$(GIT_VERSION)-x86_64-apple-darwin \
	dist/operator-sdk-$(GIT_VERSION)-ppc64le-linux-gnu \
	dist/operator-sdk-$(GIT_VERSION)-s390x-linux-gnu \
	dist/ansible-operator-$(GIT_VERSION)-aarch64-linux-gnu \
	dist/ansible-operator-$(GIT_VERSION)-x86_64-linux-gnu \
	dist/ansible-operator-$(GIT_VERSION)-x86_64-apple-darwin \
	dist/ansible-operator-$(GIT_VERSION)-ppc64le-linux-gnu \
	dist/ansible-operator-$(GIT_VERSION)-s390x-linux-gnu \
	dist/helm-operator-$(GIT_VERSION)-aarch64-linux-gnu \
	dist/helm-operator-$(GIT_VERSION)-x86_64-linux-gnu \
	dist/helm-operator-$(GIT_VERSION)-x86_64-apple-darwin \
	dist/helm-operator-$(GIT_VERSION)-ppc64le-linux-gnu \
	dist/helm-operator-$(GIT_VERSION)-s390x-linux-gnu

.PHONY: release
release: clean $(release_builds) $(release_builds:=.asc) ## Release the Operator SDK

dist/operator-sdk-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
dist/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
dist/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
dist/operator-sdk-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
dist/operator-sdk-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
dist/operator-sdk-%-linux-gnu: GOARGS = GOOS=linux

dist/ansible-operator-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
dist/ansible-operator-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
dist/ansible-operator-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
dist/ansible-operator-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
dist/ansible-operator-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
dist/ansible-operator-%-linux-gnu: GOARGS = GOOS=linux

dist/helm-operator-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64
dist/helm-operator-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
dist/helm-operator-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64
dist/helm-operator-%-ppc64le-linux-gnu: GOARGS = GOOS=linux GOARCH=ppc64le
dist/helm-operator-%-s390x-linux-gnu: GOARGS = GOOS=linux GOARCH=s390x
dist/helm-operator-%-linux-gnu: GOARGS = GOOS=linux

dist/%: ## Build the operator-sdk release binaries
	{ \
	cmdpkg=$$(echo $* | sed -E "s/(operator-sdk|ansible-operator|helm-operator).*/\1/"); \
	$(GOARGS) go build $(GO_BUILD_ARGS) -o $@ ./cmd/$$cmdpkg; \
	}

dist/%.asc: ## Create release signatures for operator-sdk release binaries
	{ \
	default_key=$$(gpgconf --list-options gpg | awk -F: '$$1 == "default-key" { gsub(/"/,""); print toupper($$10)}'); \
	git_key=$$(git config --get user.signingkey | awk '{ print toupper($$0) }'); \
	if [ "$${default_key}" = "$${git_key}" ]; then \
		gpg --output $@ --detach-sig dist/$*; \
		gpg --verify $@ dist/$*; \
	else \
		echo "git and/or gpg are not configured to have default signing key $${default_key}"; \
		exit 1; \
	fi; \
	}


.DEFAULT_GOAL:=help
.PHONY: help
help: ## Show this help screen.
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


