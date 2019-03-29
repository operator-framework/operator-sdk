# TravisCI Build Information

Travis is set to run one every push to a branch or PR.
The results of the builds can be found [here][branches] for branches and [here][pr-builds] for PRs.

## Test Workflow

In Travis CI, 4 jobs are run to test the sdk:

- [Go](#go-tests)
- [Ansible](#ansible-tests)
- [Helm](#helm-tests)
- [Markdown](#markdown)

### Before Install for Go, Ansible, and Helm

For the Go, Ansible, and Helm tests, the `before_install` and `install` stages are the same:

1. Check if non documentation files have been updated.
    - If only documentation has been updated, skip these tests.
2. Download dep and run `dep ensure`.
3. Build and install the sdk using `make install`.
4. Install ansible using `sudo pip install ansible`.
5. Run the [`hack/ci/setup-openshift`][script] script, which spins up an openshift cluster by configuring docker and then downloading the `oc` v3.11 binary and running `oc cluster up`.

The Go, Ansible, and Helm tests then differ in what tests they run.

### Go Tests

1. Run some basic [sanity checks][sanity].
    1. Run `go vet`.
    2. Check that all source files have a license.
    3. Check that all error messages start with a lower case alphabetical character and do not end with punctuation, and log messages start with an upper case alphabetical character.
    4. Make sure the repo is in a clean state (this is particularly useful for making sure the `Gopkg.lock` file up to date after `dep ensure`).
2. Run unit tests.
    1. Run `make test`.
3. Run [subcommand tests][subcommand].
    1. Run `test local` with no flags enabled.
    2. Run `test local` with most configuration flags enabled.
    3. Run `test local` in single namespace mode.
    4. Run `test local` with `--up-local` flag.
    5. Run `test local` with both `--up-local` and `--kubeconfig` flags.
    6. Create all test resources with kubectl and run `test local` with `--no-setup` flag.
    7. Run `scorecard` subcommand and check that expected score matches actual score.
4. Run [go e2e tests][go-e2e].
    1. Use `operator-sdk` to create and configure a new `memcached-operator` project and install the memcached CRD in the cluster.
    2. Run cluster test (namespace is auto-generated and deleted by test framework).
        1. Build `memcached-operator` image with `--enable-tests` flag enabled (used in the in-cluster test later).
        2. Deploy operator and resources to the cluster.
        3. Run the leader election test.
            1. Verify that operator deployment is ready.
            2. Verify that leader configmap specifies 1 leader and that the memcached operator has 2 pods (configuration for this is done in step 4.1).
            3. Delete current leader and wait for memcached-operator deployment to become ready again.
            4. Verify that leader configmap specifies 1 leader and that the memcached-operator has 2 pods.
            5. Verify that the name of the new leader is different from the name of the old leader.
        4. Run the memcached scale test.
            1. Create memcached CR specifying a desired cluster size of 3 and wait until memcached cluster is of size 3.
            2. Increase desired cluster size to 4 and wait until memcached cluster is of size 4.
    3. Run in-cluster test.
        1. Create new namespace for the test.
        2. Run the tests using the `test cluster` subcommand with the image generated in the cluster test (which contains the scale test described in step 4.2.4).
        3. Delete the test namespace.
    4. Run local test.
        1. Create new namespace for the test.
        2. Start operator using `up local` subcommand.
        3. Run memcached scale test (described in step 4.2.4)
        4. Delete the test namespace.
    5. Run [TLS library tests][tls-tests].
        1. This test runs multiple simple tests of the operator-sdk's TLS library. The tests run in parallel and each tests runs in its own namespace.

### Ansible tests

1. Run [ansible e2e tests][ansible-e2e].
    1. Create base ansible operator project by running [`hack/image/ansible/scaffold-ansible-image.go`][ansible-base].
    2. Build base ansible operator image.
    3. Create and configure a new ansible type memcached-operator.
    4. Create cluster resources.
    5. Wait for operator to be ready.
    6. Create a memcached CR and wait for it to be ready.
    7. Create a configmap that the memcached-operator is configured to delete using a finalizer.
    8. Delete memcached CR and verify that the finalizer deleted the configmap.
    9. Run `operator-sdk migrate` to add go source to the operator.
    10. Run `operator-sdk build` to compile the new binary and build a new image.
    11. Re-run steps 4-8 to test the migrated operator.

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
    10. Run `operator-sdk migrate` to add go source to the operator.
    11. Run `operator-sdk build` to compile the new binary and build a new image.
    12. Re-run steps 4-9 to test the migrated operator.

**NOTE**: All created resources, including the namespace, are deleted using a bash trap when the test finishes

### Markdown

The markdown test does not create a new cluster and runs in a barebones Travis VM configured only for `bash`. This allows documentation PRs to pass quickly, as they don't require code tests. The markdown checker uses a precompiled version of [`marker`][marker-github] stored in [`hack/ci/marker`][marker-local] to check the validity and correctness of the links in all markdown files in the `doc` directory.

**NOTE**: There is currently a bug in marker that causes link with underscores (`_`) to not be checked correctly.

[branches]: https://travis-ci.org/operator-framework/operator-sdk/branches
[pr-builds]: https://travis-ci.org/operator-framework/operator-sdk/pull_requests
[script]: ../../../hack/ci/setup-openshift.sh
[sanity]: ../../../hack/tests/sanity-check.sh
[subcommand]: ../../../hack/tests/test-subcommand.sh
[go-e2e]: ../../../hack/tests/e2e-go.sh
[tls-tests]: ../../../test/e2e/tls_util_test.go
[ansible-e2e]: ../../../hack/tests/e2e-ansible.sh
[ansible-base]: ../../../hack/image/ansible/scaffold-ansible-image.go
[helm-e2e]: ../../../hack/tests/e2e-helm.sh
[helm-base]: ../../../hack/image/helm/scaffold-helm-image.go
[marker-github]: https://github.com/crawford/marker
[marker-local]: ../../../hack/ci/marker
