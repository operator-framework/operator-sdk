---
title: Testing
linkTitle: Testing
weight: 6
---

On all PRs, a suite of static and cluster tests is run against your changes in a CI environment.
These tests can also be run locally, which is discussed [below](#local-test-environment).

Static tests consist of [unit][unit-tests], formatting, and doc link tests.

Cluster tests consist of several test types:
- End-to-end (e2e): simulate the "happy path" usage of the `operator-sdk` binary and resulting operator project.
- Integration: test components of the `operator-sdk` binary and features of scaffolded projects that are
bound to external projects, such as [OLM][olm].
- Subcommand: ensure individual subcommands function as intended with a variety of input options.

## Before submitting a PR

Always run tests before submitting a PR to reduce the number of needless CI errors.

##### Docs only

```sh
make test-static
```

##### Code

```sh
make test-all
```


## Local Test Environment

If running tests locally, access to a Kubernetes cluster of a [compatible version][k8s-version-compat] is required.
These tests require `KUBECONFIG` be set or kubeconfig file be present in a default location like `$HOME/.kube/config`.

You will also need to set up an `envtest` environment for cluster tests. Follow [this doc][envtest-setup]
for setup instructions.

### Local clusters

A local [kind][kind] cluster is used for running tests.

## Running Tests

All the tests are run through the [`Makefile`][makefile]. Run `make help` for a full list of available tests.

[unit-tests]: https://onsi.github.io/gomega/
[olm]: https://olm.operatorframework.io/
[minikube]: https://kubernetes.io/docs/setup/learning-environment/minikube/
[kind]: https://kind.sigs.k8s.io/
[envtest-setup]:https://book.kubebuilder.io/reference/envtest.html
[makefile]: https://github.com/operator-framework/operator-sdk/blob/master/Makefile
[k8s-version-compat]:/docs/overview#kubernetes-version-compatibility
