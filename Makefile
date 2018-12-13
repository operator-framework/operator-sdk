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
REPO = github.com/operator-framework/operator-sdk
BUILD_PATH = $(REPO)/commands/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)

export CGO_ENABLED:=0

all: format test build/operator-sdk

format:
	$(Q)go fmt $(PKGS)

dep:
	$(Q)dep ensure -v

dep-update:
	$(Q)dep ensure -update -v

clean:
	$(Q)rm -rf build

.PHONY: all test format dep clean

install:
	$(Q)go install $(BUILD_PATH)

release_x86_64 := \
	build/operator-sdk-$(VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-apple-darwin

release: clean $(release_x86_64) $(release_x86_64:=.asc)

build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64

build/%:
	$(Q)$(GOARGS) go build -o $@ $(BUILD_PATH)
	
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

test: dep test/sanity test/unit install test/subcommand test/e2e

test/ci-go: test/sanity test/unit test/subcommand test/e2e/go

test/ci-ansible: test/e2e/ansible

test/ci-helm: test/e2e/helm

test/sanity:
	./hack/tests/sanity-check.sh

test/unit:
	./hack/tests/unit.sh

test/subcommand:
	./hack/tests/test-subcommand.sh

test/e2e: test/e2e/go test/e2e/ansible test/e2e/helm

test/e2e/go:
	./hack/tests/e2e-go.sh $(ARGS)

test/e2e/ansible:
	./hack/tests/e2e-ansible.sh

test/e2e/helm:
	./hack/tests/e2e-helm.sh

.PHONY: test test/sanity test/unit test/subcommand test/e2e test/e2e/go test/e2e/ansible test/e2e/helm test/ci-go test/ci-ansible
