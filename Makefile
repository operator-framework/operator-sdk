pkgs = $(shell go list ./... | grep -v /vendor/)
BUILD_DIR := ./build

all: format test install

install:
	go install github.com/operator-framework/operator-sdk/commands/operator-sdk

format:
	go fmt $(pkgs)

dep:
	dep ensure -v

clean:
	rm -rf $(BUILD_DIR)	
	
build: clean
	mkdir $(BUILD_DIR)
	go build \
		-o $(BUILD_DIR)/operator-sdk \
		github.com/operator-framework/operator-sdk/commands/operator-sdk
		
.PHONY: all install test format dep clean build
