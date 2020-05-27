IMAGE = quay.io/username/custom-scorecard-tests
SHELL = /bin/bash

all: build

clean: ## Clean up the build artifacts
	rm -rf build

build:
	go build internal/tests/tests.go
image-build: ## Running `make image-build` from the project root of this example test function will build docker test image.
	go build -o images/custom-scorecard-tests/custom-scorecard-tests images/custom-scorecard-tests/cmd/test/main.go
	cd images/custom-scorecard-tests && docker build -t $(IMAGE):dev .
