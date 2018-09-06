pkgs = $(shell go list ./... | grep -v /vendor/)

all: format test install

install:
	go install github.com/operator-framework/operator-sdk/commands/operator-sdk

format:
	go fmt $(pkgs)

dep:
	dep ensure -v

.PHONY: all install test format dep
