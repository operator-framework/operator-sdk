# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
       BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
       Q =
else
       Q = @
endif

VERSION = $(shell git describe --dirty --tags)
REPO = github.com/operator-framework/operator-sdk
BUILD_PATH = $(REPO)/commands/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)

export CGO_ENABLED:=0

all: format test build/operator-sdk

format:
	$(Q)go fmt $(PKGS)

dep:
	$(Q)dep ensure -v

clean:
	$(Q)rm -rf build

.PHONY: all test format dep clean

install:
	$(Q)go install $(BUILD_PATH)

release_x86_64 := \
	build/operator-sdk-$(VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-apple-darwin

release: clean $(release_x86_64)

build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64

build/%:
	$(Q)$(GOARGS) go build -o $@ $(BUILD_PATH)
	
DEFAULT_KEY = $(shell gpgconf --list-options gpg \
								| awk -F: '$$1 == "default-key" { gsub(/"/,""); print $$10}')
build/%.asc:
ifeq ("$(DEFAULT_KEY)","$(shell git config --get user.signingkey)")
	$(Q)gpg --output $@ --detach-sig build/$*
	$(Q)gpg --verify $@ build/$*
else
	@echo "git and/or gpg are not configured to have default signing key ${DEFAULT_KEY}"
	@exit 1
endif

.PHONY: install release_x86_64 release
