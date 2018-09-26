# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
       BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
       Q =
else
       Q = @
endif

VERSION = $(shell git describe --dirty)
REPO = github.com/operator-framework/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
LD_FLAGS = "-w -X $(REPO)/version.Version=$(VERSION)"

# export GOPATH=$(shell pwd)/gopath
export CGO_ENABLED:=0

all: format test build/operator-sdk

format:
	go fmt $(PKGS)

dep:
	dep ensure -v

# gopath:
# 	$(Q)mkdir -p gopath/src/github.com/operator-framework
# 	$(Q)ln -s ../../../.. gopath/src/$(REPO)

clean:
	$(Q)rm -rf build

install:
	go install github.com/operator-framework/operator-sdk/commands/operator-sdk

release_aarch64 := \
	build/operator-sdk-$(VERSION)-aarch64-linux-gnu

release_x86_64 := \
	build/operator-sdk-$(VERSION)-x86_64-linux-gnu \
	build/operator-sdk-$(VERSION)-x86_64-apple-darwin

release: $(release_aarch64) $(release_x86_64)

build/operator-sdk-%-aarch64-linux-gnu: GOARGS = GOOS=linux GOARCH=arm64

build/operator-sdk-%-x86_64-linux-gnu: GOARGS = GOOS=linux GOARCH=amd64
build/operator-sdk-%-x86_64-apple-darwin: GOARGS = GOOS=darwin GOARCH=amd64

build/%: clean
	$(Q)$(GOARGS) go build -o $@ -ldflags $(LD_FLAGS)

.PHONY: all test format dep clean install release_aarch64 release_x86_64 release
