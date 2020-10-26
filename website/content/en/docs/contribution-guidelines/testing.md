---
title: Testing
linkTitle: Testing
weight: 10
---

On all PRs, a suite of static and cluster tests is run against your changes in a CI environment.
These tests can also be run locally, which is discussed [below](#local-test-environment).

Static tests consist of [unit][unit-tests], formatting, and doc link tests.

Cluster tests consist of several test types:
- End-to-end (e2e): simulate the "happy path" usage of the `operator-sdk` binary and resulting operator project.
- Integration: test components of the `operator-sdk` binary and features of scaffolded projects that are
bound to external projects, such as [OLM][olm].
- Subcommand: ensure individual subcommands function as intended with a variety of input options.

## Local Test Environment

If running tests locally, access to a Kubernetes cluster of server version v1.11.3 or higher is required.
These tests require `KUBECONFIG` be set or kubeconfig file be present in a default location like `$HOME/.kube/config`.

You will also need to set up an `envtest` environment for cluster tests. Follow [this doc][envtest-setup]
for setup instructions.

### Local clusters

Two options for testing with a local cluster are [minikube][minikube] and [kind][kind].
Ensure `KUBECONFIG` is set correctly for the chosen cluster type.

## Running Tests

On any PR, the entire test suite is run against your changes in a CI environment.
Therefore it is advantageous to run all tests before pushing changes to the remote repo:

```sh
make test-sanity test-links test-unit test-subcommand test-integration test-e2e
```

All the tests are run through the [`Makefile`][makefile]. Run `make help` for a full list of available tests.

[unit-tests]: https://onsi.github.io/gomega/
[olm]: https://olm.operatorframework.io/
[minikube]: https://kubernetes.io/docs/setup/learning-environment/minikube/
[kind]: https://kind.sigs.k8s.io/
[envtest-setup]: /docs/building-operators/golang/references/envtest-setup
[makefile]: https://github.com/operator-framework/operator-sdk/blob/v1.1.x/Makefile
