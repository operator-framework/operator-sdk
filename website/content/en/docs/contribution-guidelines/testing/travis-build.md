---
title: Travis CI
weight: 30
---

Travis is set to run one every push to a branch or PR.
The results of the builds can be found [here][branches] for branches and [here][pr-builds] for PRs.

## Test Workflow

In Travis CI, 4 jobs are run to test the sdk:

- [Go](#go-tests)
- [Ansible](#ansible-tests)
- [Helm](#helm-tests)

### Before Install for Go, Ansible, and Helm

For the Go, Ansible, and Helm tests, the `before_install` and `install` stages are the same:

1. Check if non documentation files have been updated.
    - If only documentation has been updated, skip these tests.
2. Run `make tidy` to ensure `go.mod` and `go.sum` are up-to-date.
3. Build and install the sdk using `make install`.
4. Install ansible using `sudo pip install ansible`.
5. Run the [`hack/ci/setup-k8s.sh`][k8s-script] script, which spins up a [kind][kind] Kubernetes cluster of a particular version by configuring docker, and downloads the `kubectl` of the same version.

The Go, Ansible, and Helm tests then differ in what tests they run.

### Go Tests

1. Run some basic [sanity checks][sanity].
    1. Run `go vet`.
    2. Check that all source files have a license.
    3. Check that all error messages start with a lower case alphabetical character and do not end with punctuation, and log messages start with an upper case alphabetical character.
    4. Make sure the repo is in a clean state (this is particularly useful for making sure `go.mod` and `go.sum` are up-to-date after running `make tidy`).
2. Run unit tests.
    1. Run `make test-unit`.
3. Run [go e2e tests][go-e2e].
    1. Run `make test-e2e-go`.

### Ansible tests

1. Run [ansible molecule tests][ansible-molecule]. (`make test-e2e-ansible-molecule`)
    1. Create and configure a new ansible type memcached-operator.
    2. Create cluster resources.
    4. Change directory to [`test/ansible`][ansible-test] and run `molecule test -s local`

**NOTE**: All created resources, including the namespace, are deleted using a bash trap when the test finishes

### Helm Tests

1. Run [helm e2e tests][helm-e2e].
    1. Create base helm operator project by running [`hack/image/helm/scaffold-helm-image.go`][helm-base].
    2. Build base helm operator image.
    3. Create and configure a new helm type nginx-operator.
    4. Create cluster resources.
    5. Wait for operator to be ready.
    6. Create nginx CR and wait for it to be ready.
    7. Scale up the dependent deployment and verify the operator reconciles it back down.
    8. Scale up the CR and verify the dependent deployment scales up accordingly.
    9. Delete nginx CR and verify that finalizer (which writes a message in the operator logs) ran.
    10. Run `operator-sdk migrate` to add go source to the operator (see this [note][deps_mgmt] on dependency management first).
    11. Run `operator-sdk build` to compile the new binary and build a new image.
    12. Re-run steps 4-9 to test the migrated operator.

**NOTE**: All created resources, including the namespace, are deleted using a bash trap when the test finishes

[branches]: https://travis-ci.org/operator-framework/operator-sdk/branches
[pr-builds]: https://travis-ci.org/operator-framework/operator-sdk/pull_requests
[k8s-script]: https://github.com/operator-framework/operator-sdk/blob/master/hack/ci/setup-k8s.sh
[kind]: https://kind.sigs.k8s.io/
[sanity]: https://github.com/operator-framework/operator-sdk/blob/master/hack/tests/sanity-check.sh
[go-e2e]: https://github.com/operator-framework/operator-sdk/blob/master/test/e2e/e2e_suite.go
[ansible-molecule]: https://github.com/operator-framework/operator-sdk/blob/master/hack/tests/e2e-ansible-molecule.sh
[ansible-test]: https://github.com/operator-framework/operator-sdk/tree/master/test/ansible
[helm-e2e]: https://github.com/operator-framework/operator-sdk/blob/master/hack/tests/e2e-helm.sh
[helm-base]: https://github.com/operator-framework/operator-sdk/blob/master/hack/image/helm/scaffold-helm-image.go
[deps_mgmt]: /docs/golang/legacy/quickstart#a-note-on-dependency-management
