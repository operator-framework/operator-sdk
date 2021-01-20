---
title: Quickstart for Go-based Operators
linkTitle: Quickstart
weight: 20
description: A simple set of instructions to set up and run a Go-based operator.
---

This guide walks through an example of building a simple memcached-operator using tools and libraries provided by the Operator SDK.

## Prerequisites

- Go through the [installation guide][install-guide].
- User authorized with `cluster-admin` permissions.

## Steps

1. Create a project directory for your project and initialize the project:

  ```sh
  mkdir memcached-operator
  cd memcached-operator
  operator-sdk init --domain example.com --repo github.com/example/memcached-operator
  ```

1. Create a simple Memcached API:

  ```sh
  operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource --controller
  ```

1. Use the built-in Makefile targets to build and push your operator.
Make sure to define `IMG` when you call `make`:

  ```sh
  export USERNAME=<quay-namespace>
  export OPERATOR_IMG="quay.io/$USERNAME/memcached-operator:v0.0.1"
  make docker-build docker-push IMG=$OPERATOR_IMG
  ```

**Note**: If using an OS which does not point `sh` to the `bash` shell (Ubuntu for example) then you should add the following line to the `Makefile`:

`SHELL := /bin/bash`

This will fix potential issues when the `docker-build` target runs the controller test suite. Issues maybe similar to following error:
`failed to start the controlplane. retried 5 times: fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory occurred`

### OLM deployment

1. Install [OLM][doc-olm]:

  ```sh
  operator-sdk olm install
  ```

1. Bundle your operator and push the bundle image:

  ```sh
  make bundle IMG=$OPERATOR_IMG
  # Note the "-bundle" component in the image name below.
  export BUNDLE_IMG="quay.io/$USERNAME/memcached-operator-bundle:v0.0.1"
  make bundle-build BUNDLE_IMG=$BUNDLE_IMG
  make docker-push IMG=$BUNDLE_IMG
  ```

1. Run your bundle:

  ```sh
  operator-sdk run bundle $BUNDLE_IMG
  ```

1. Create a sample Memcached custom resource:

  ```console
  $ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
  memcached.cache.example.com/memcached-sample created
  ```

1. Uninstall the operator:

  ```sh
  operator-sdk cleanup memcached-operator
  ```


### Direct deployment

1. Deploy your operator:

  ```sh
  make deploy IMG=$OPERATOR_IMG
  ```

1. Create a sample Memcached custom resource:

  ```console
  $ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
  memcached.cache.example.com/memcached-sample created
  ```

1. Uninstall the operator:

  ```sh
  make undeploy
  ```


## Next Steps

Read the [full tutorial][tutorial] for an in-depth walkthough of building a Go operator.


[install-guide]:/docs/building-operators/golang/installation
[doc-olm]:/docs/olm-integration/quickstart-bundle/#enabling-olm
[tutorial]:/docs/building-operators/golang/tutorial/
