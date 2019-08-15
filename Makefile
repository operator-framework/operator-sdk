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

export CGO_ENABLED:=0
export GO111MODULE:=on
export GOPROXY?=https://proxy.golang.org/

all: format test build/operator-sdk

format:
	$(Q)go fmt $(PKGS)

tidy:
	$(Q)go mod tidy -v

clean:
	$(Q)rm -rf build

.PHONY: all test format tidy clean

install:
	$(Q)go install \
		-gcflags "all=-trimpath=${GOPATH}" \
		-asmflags "all=-trimpath=${GOPATH}" \
		-ldflags " \
			-X '${REPO}/version.GitVersion=${VERSION}' \
			-X '${REPO}/version.GitCommit=${GIT_COMMIT}' \
		" \
		$(BUILD_PATH)

ci-build: build/operator-sdk-$(VERSION)-x86_64-linux-gnu ci-install

ci-install:
	mv build/operator-sdk-$(VERSION)-x86_64-linux-gnu build/operator-sdk

release_x86_64 := \
	build/operator-sdk-$(VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-apple-darwin

release: clean $(release_x86_64) $(release_x86_64:=.asc)

build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64

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

.PHONY: install release_x86_64 release

test: test/unit

test-ci: test/markdown test/sanity test/unit install test/subcommand test/e2e

test/ci-go: test/subcommand test/e2e/go

test/ci-ansible: test/e2e/ansible-molecule

test/ci-helm: test/e2e/helm

test/sanity: tidy
	./hack/tests/sanity-check.sh

test/unit:
	$(Q)go test -count=1 -short ./cmd/...
	$(Q)go test -count=1 -short ./pkg/...
	$(Q)go test -count=1 -short ./internal/...

test/subcommand: test/subcommand/test-local test/subcommand/scorecard test/subcommand/alpha-olm

test/subcommand2:
	./ci/tests/subcommand.sh

test/subcommand/test-local:
	./hack/tests/test-subcommand.sh

test/subcommand/scorecard:
	./hack/tests/scorecard-subcommand.sh

test/subcommand/scorecard2:
	./ci/tests/scorecard-subcommand.sh

test/subcommand/alpha-olm:
	./hack/tests/alpha-olm-subcommands.sh

test/e2e: test/e2e/go test/e2e/ansible test/e2e/ansible-molecule test/e2e/helm

test/e2e/go:
	./hack/tests/e2e-go.sh $(ARGS)

test/e2e/go2:
	./ci/tests/e2e-go.sh $(ARGS)

test/e2e/ansible: image/build/ansible
	./hack/tests/e2e-ansible.sh

test/e2e/ansible2:
	./ci/tests/e2e-ansible.sh

test/e2e/ansible-molecule: image/build/ansible
	./hack/tests/e2e-ansible-molecule.sh

test/e2e/helm: image/build/helm
	./hack/tests/e2e-helm.sh

test/e2e/helm2:
	./ci/tests/e2e-helm.sh

test/markdown:
	./hack/ci/marker --root=doc

.PHONY: test test-ci test/sanity test/unit test/subcommand test/subcommand/test-local test/subcommand/scorecard test/subcommand/alpha-olm test/e2e test/e2e/go test/e2e/ansible test/e2e/ansible-molecule test/e2e/helm test/ci-go test/ci-ansible test/ci-helm test/markdown

image: image/build image/push

image/scaffold/ansible:
	go run ./hack/image/ansible/scaffold-ansible-image.go

image/scaffold/helm:
	go run ./hack/image/helm/scaffold-helm-image.go

image/build: image/build/ansible image/build/helm image/build/scorecard-proxy

image/build/ansible: build/operator-sdk-dev-x86_64-linux-gnu
	./hack/image/build-ansible-image.sh $(ANSIBLE_BASE_IMAGE):dev

image/build/helm: build/operator-sdk-dev-x86_64-linux-gnu
	./hack/image/build-helm-image.sh $(HELM_BASE_IMAGE):dev

image/build/scorecard-proxy:
	./hack/image/build-scorecard-proxy-image.sh $(SCORECARD_PROXY_BASE_IMAGE):dev

image/push: image/push/ansible image/push/helm image/push/scorecard-proxy

image/push/ansible:
	./hack/image/push-image-tags.sh $(ANSIBLE_BASE_IMAGE):dev $(ANSIBLE_IMAGE)

image/push/helm:
	./hack/image/push-image-tags.sh $(HELM_BASE_IMAGE):dev $(HELM_IMAGE)

image/push/scorecard-proxy:
	./hack/image/push-image-tags.sh $(SCORECARD_PROXY_BASE_IMAGE):dev $(SCORECARD_PROXY_IMAGE)

.PHONY: image image/scaffold/ansible image/scaffold/helm image/build image/build/ansible image/build/helm image/push image/push/ansible image/push/helm
