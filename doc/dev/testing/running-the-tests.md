# Running the SDK Tests

The operator-sdk is automatically tested with a variety of tests anytime
a pull request is made. The E2E tests ensure that the operator-sdk acts as intended by
simulating how a typical user might use the SDK. The automated tests test each PR and run in
Travis CI, and Travis CI has a couple of features to simplify the E2E tests that we run. For
a more in depth description of the tests that Travis runs, please read the [Travis Build][travis] doc.
This doc will talk about how to run the tests locally as well.

## Running the Tests Locally

To run the tests locally, the tests either need access to a remote Kubernetes cluster or a
local Kubernetes instance running on the machine.

### Remote Kubernetes Instance

To run the tests on a remote cluster, the tests need access to a remote Kubernetes cluster
running Kubernetes 1.11.3 or higher as well as a docker image repo to push the operator image to,
such as [`quay.io`][quay]. Your kubeconfig must be located at `$HOME/.kube/config` and certain
tests will not run on remote clusters. See [Running the Tests](#running-the-tests) for more details.

### Local OpenShift Cluster

One way to run the tests is with an OpenShift 3.11 cluster and `oc cluster up` on a Linux system.

For the first run configuration, either you can either run the `hack/ci/setup-openshift.sh` script, or download
and install the [oc binary][oc-binary] and run these commands:

```sh
$ sudo service docker stop
$ sudo sed -i 's/DOCKER_OPTS=\"/DOCKER_OPTS=\"--insecure-registry 172.30.0.0\/16 /' /etc/default/docker
$ sudo service docker start
```

Depending on the system you are on, you may also need to configure your firewall before running the cluster.
Refer to the [official docs][oc-docs] for more information.

After the initial configuration, you can use this command to start the cluster and login as admin:

```sh
$ oc cluster up --base-dir=$HOME/oscluster
$ oc login -u system:admin
```

You can use this command to stop the cluster:

```sh
$ oc cluster down
```

To fully delete the cluster, you must run these commands:

```sh
$ oc cluster down
$ sudo umount $(grep $HOME/oscluster /proc/mounts | cut -f2 -d" " | sort -r)
$ sudo rm -rf $HOME/oscluster
```

**NOTE**: Starting with openshift 4.0, the `oc cluster up` command will not be available. This document
and the tests will be updated in the future to support openshift 4.0.


### Local Minikube

Another option for testing is using minikube. This is not advised as it uses vanilla Kubernetes, which has less
strict security and may allow some tests to pass when they would not under openshift. Minikube is faster than
openshift and uses less RAM though. To start the minikube cluster, download and install the proper [binary][minikube-binary]
for your system and run these commands.

```sh
# The latest version of minikube at the time of writing (v0.31.0) defaults to k8s v1.10.0, so we must explicitly specify the latest k8s v1.11
$ minikube start --kubernetes-version v1.11.6
$ eval $(minikube docker-env)
```

## Running the tests

All the tests are run through the [`Makefile`][makefile]. This is a brief description of all makefile test instructions:

- `test` - Runs the unit tests (`test/unit`).
- `test-ci` - Runs markdown, sanity, and unit tests, installs the SDK binary, and runs the SDK subcommand and all E2E tests.
- `test/ci-go` - Runs all the tests that the Go job runs in CI (`subcommand` and `e2e/go`).
- `test/ci-ansible` - Runs all the tests that the Ansible job runs in Travis CI (`e2e/ansible` and `test/e2e/ansible-molecule`).
- `test/ci-helm` - Runs all the tests that the Helm job runs in Travis CI (`test/e2e/helm`).
- `test/sanity` - Runs sanity checks.
- `test/unit` - Runs unit tests.
- `test/subcommand` - Runs subcommand tests.
- `test/e2e` - Runs all E2E tests (`test/e2e/go`, `test/e2e/ansible`, `test/e2e/ansible-molecule`, and `e2e/helm`).
- `test/e2e/go` - Runs the go E2E test.
- `test/e2e/ansible` - Runs the ansible E2E test.
- `test/e2e/ansible-molecule` - Runs the ansible molecule E2E test.
- `test/e2e/helm` - Runs the helm E2E test.
- `test/markdown` - Runs the markdown checks

For more info on what these tests actually do, please see the [Travis Build][travis] doc.

Some of the tests will run using the kube config in `$HOME/.kube/config` (others may check the `KUBECONFIG` env var first)
and the operator images will be built and stored in you local docker registry.

### Go E2E test flags

The `make test/e2e/go` command accepts an `ARGS` variable containing flags that will be passed to `go test`:

- `-image-name` string - Sets the operator test image tag to be built and used in testing. Defaults to "quay.io/example/memcached-operator:v0.0.1"
- `-local-repo` string - Sets the path to the local SDK repo being tested. Defaults to the path of the SDK repo containing e2e tests. This is useful for testing customized e2e code.

An example of using `ARGS` is in the note below.

**NOTE**: Some of these tests, specifically the ansible (`test/e2e/ansible` and `test/ci-ansible`), helm
(`test/e2e/helm` and `test/ci-helm`), and Go (`test/e2e/go` and `test/e2e/ci-go`) tests, only work when the cluster shares the local docker
registry, as is the case with `oc cluster up` and `minikube` after running `eval $(minikube docker-env)`.

All other tests will run correctly on a remote cluster if `$HOME/.kube/config` points to the remote cluster and your
`KUBECONFIG` env var is either empty or is set to the path of a kubeconfig for the remote cluster.

## Cleanup of the Go E2E Tests

The E2E tests create a new project using the operator-sdk to run in the provided
cluster. The tests are designed to cleanup everything that gets created, but some errors
during the go tests can cause these cleanups to fail (the ansible and helm E2E tests should
always clean up correctly). For example, if a segfault occurs or a user kills the
testing process, the cleanup functions for the go tests will not run. To manually clean up a test:

1. Delete the CRD (`kubectl delete crd memcacheds.cache.example.com`).
2. Delete the namespaces that the tests run in, which also deletes any resources created within the namespaces. The namespaces start with `memcached-memcached-group` or `main` and are appended with a unix timestamp (seconds since Jan 1 1970). The kubectl command can be used to delete namespaces: `kubectl delete namespace $NAMESPACE`.

[travis]: ./travis-build.md
[quay]: https://quay.io
[oc-docs]: https://github.com/openshift/origin/blob/v3.11.0/docs/cluster_up_down.md
[oc-binary]: https://github.com/openshift/origin/releases/v3.11.0
[minikube-binary]: https://github.com/kubernetes/minikube/releases
[makefile]: ../../../Makefile
