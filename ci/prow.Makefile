# Makefile specifically intended for use in prow/api-ci only.

export CGO_ENABLED := 0

build:
	$(MAKE) -f Makefile build/operator-sdk

test-e2e-go test/e2e/go:
	./ci/tests/e2e-go.sh $(ARGS)

test-e2e-ansible test/e2e/ansible:
	./ci/tests/e2e-ansible.sh

test-e2e-helm test/e2e/helm:
	./ci/tests/e2e-helm.sh

test-subcommand test/subcommand:
	./ci/tests/subcommand.sh
