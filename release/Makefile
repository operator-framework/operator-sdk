## This Makefile should only be called from the project root Makefile ##

# Needed for glob expansion.
SHELL = /bin/bash
.SHELLFLAGS = -O extglob -c

# Dry run flags.
ifneq ($(DRY_RUN),)
SNAPSHOT_FLAGS = --snapshot --skip-publish --skip-sign --rm-dist
CHANGELOG = changelog/generated/tmp.md
$(shell touch $(CHANGELOG))
endif

# Ensure that this Makefile is run from the project root (always contains the 'cmd/' directory).
ifeq (,$(wildcard cmd))
	$(error "This Makefile must be invoked from the operator-sdk project root")
endif

##@ Release

.PHONY: release
CHANGELOG ?= changelog/generated/$(GIT_VERSION).md
release: ## Publish an operator-sdk release, with option for a dry run with DRY_RUN.
ifeq (,$(GIT_VERSION))
	$(error "GIT_VERSION must be set to a git tag")
endif
	$(SCRIPTS_DIR)/fetch goreleaser 0.177.0
	GORELEASER_CURRENT_TAG=$(GIT_VERSION) $(TOOLS_DIR)/goreleaser $(SNAPSHOT_FLAGS) --release-notes=$(CHANGELOG) --parallelism 5
ifneq ($(DRY_RUN),)
	rm $(CHANGELOG)
endif

##@ Pre-Release

.PHONY: check_release_version
check_release_version:
ifeq (,$(RELEASE_VERSION))
	$(error "RELEASE_VERSION must be set to a release tag")
endif

.PHONY: prerelease
prerelease: check_release_version changelog ## Create release commit changes to commit.
	./website/scripts/update_branch_mappings.sh $(RELEASE_VERSION)
	./website/scripts/update_download_url.sh $(RELEASE_VERSION)
	./website/scripts/update_latest_release_compat.sh

.PHONY: changelog
changelog: check_release_version ## Generate the changelog.
	@mkdir -p changelog/generated && rm -f changelog/generated/$(RELEASE_VERSION).md
	go run ./release/changelog/gen-changelog.go -tag=$(RELEASE_VERSION) -changelog=changelog/generated/$(RELEASE_VERSION).md
	rm -f ./changelog/fragments/!(00-template.yaml)

.PHONY: tag
VERSION_REGEXP := ^(scorecard-kuttl/)?v[0-9]+\.[0-9]+\.[0-9]+(\-(alpha|beta|rc)\.[0-9]+)?$
tag: ## Create a release tag.
ifeq (,$(RELEASE_VERSION))
	$(error "RELEASE_VERSION must be set to tag HEAD")
endif
ifeq (,$(shell [[ "$(RELEASE_VERSION)" =~ $(VERSION_REGEXP) ]] && echo 1))
	$(error "Version $(RELEASE_VERSION) must match regexp $(VERSION_REGEXP)")
endif
	git tag --sign --message "Operator SDK $(RELEASE_VERSION)" $(RELEASE_VERSION)
	git verify-tag --verbose $(RELEASE_VERSION)

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show this help screen.
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
