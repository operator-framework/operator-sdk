## This Makefile should only be called from the project root ##

# Needed for glob expansion.
SHELL = /bin/bash
.SHELLFLAGS = -O extglob -c

ifneq ($(DRY_RUN),)
TAG ?= $(GIT_VERSION)
endif

# Ensure that this Makefile is run from the project root (always contains the 'cmd/' directory).
ifeq (,$(wildcard cmd))
$(error "This Makefile must be invoked from the operator-sdk project root")
endif

.PHONY: all
all: release

.PHONY: check_tag
check_tag:
ifeq ($(TAG),)
	$(error "TAG must be set to a release tag")
endif

.PHONY: changelog
changelog: check_tag ## Generate the changelog.
	go run ./release/changelog/gen-changelog.go -tag=$(TAG) -changelog=CHANGELOG.md
ifeq ($(DRY_RUN),)
	rm -f ./changelog/fragments/!(00-template.yaml)
endif

# Convenience wrappers for pushing all remotely hosted images.
.PHONY: image-push image-push-multiarch
IMAGE_TARGET_LIST = operator-sdk helm-operator ansible-operator scorecard-test scorecard-test-kuttl
image-push: $(foreach i,$(IMAGE_TARGET_LIST),image-push/$(i)) ## Push all images for the host architecture.
image-push-multiarch: $(foreach i,$(IMAGE_TARGET_LIST),image-push-multiarch/$(i)) ## Push the manifest list for all architectures.

# Push an image for an architecture.
BUILD_IMAGE_REPO = quay.io/operator-framework
IMAGE_REPO ?= $(BUILD_IMAGE_REPO)
GO_ARCH := $(shell go env GOARCH)
image-push/%: IMAGE_PUSH_TAG = $(IMAGE_REPO)/$*-$(GO_ARCH)
image-push/%:
	./hack/image/push-image-tags.sh $(BUILD_IMAGE_REPO)/$*:dev $(IMAGE_PUSH_TAG)

# Push multiarch images.
ARCHES ?= amd64 arm64 ppc64le s390x
KUTTL_ARCHES := amd64 arm64 ppc64le
image-push-multiarch/scorecard-test-kuttl: ARCHES = $(KUTTL_ARCHES)
image-push-multiarch/%: IMAGE_PUSH_TAG = $(IMAGE_REPO)/$*
image-push-multiarch/%:
	./hack/image/push-manifest-list.sh $(IMAGE_PUSH_TAG) $(ARCHES)

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
release: $(release_builds) $(release_builds:=.asc) ## Release the Operator SDK

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
	go build $(GO_BUILD_ARGS) -o $@ ./cmd/$$cmdpkg; \
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
