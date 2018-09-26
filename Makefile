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
BUILD_PATH = $(REPO)/commands/operator-sdk
PKGS = $(shell go list ./... | grep -v /vendor/)
LD_FLAGS = "-w -X $(REPO)/version.Version=$(VERSION)"

# export GOPATH=$(shell pwd)/gopath
export CGO_ENABLED:=0

all: format test build/operator-sdk

format:
	$(Q)go fmt $(PKGS)

dep:
	$(Q)dep ensure -v

# gopath:
# 	$(Q)mkdir -p gopath/src/github.com/operator-framework
# 	$(Q)ln -s ../../../.. gopath/src/$(REPO)

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
	$(Q)$(GOARGS) go build -o $@ -ldflags $(LD_FLAGS) $(BUILD_PATH)
	
build/%.asc:
	$(Q)gpg --output $@ --detach-sig build/$*
	$(Q)gpg --verify $@ build/$*

.PHONY: install release_x86_64 release
