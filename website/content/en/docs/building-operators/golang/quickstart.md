---
title: Quickstart for Go-based Operators
linkTitle: Quickstart
weight: 20
description: A simple set of instructions to set up and run a Go-based operator.
---

This guide walks through an example of building a simple memcached-operator using tools and libraries provided by the Operator SDK.

## Prerequisites

- Go through the [installation guide][install-guide].
- Make sure your user is authorized with `cluster-admin` permissions.
- An accessible image registry for various operator images (ex. [hub.docker.com](https://hub.docker.com/signup),
[quay.io](https://quay.io/)) and be logged in to your command line environment.
  - `example.com` is used as the registry Docker Hub namespace in these examples.
  Replace it with another value if using a different registry or namespace.
  - [Authentication and certificates][image-reg-config] if the registry is private or uses a custom CA.


## Steps

1. Create a project directory for your project and initialize the project:

  ```sh
  mkdir memcached-operator
  cd memcached-operator
  operator-sdk init --domain example.com --repo github.com/example/memcached-operator
  ```

**Note** If your local environment is Apple Silicon (`darwin/arm64`) use the `go/v4-alpha`
plugin which provides support for this platform by adding to the init subCommand the flag `--plugins=go/v4-alpha`

1. Create a simple Memcached API:

  ```sh
  operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource --controller
  ```

1. Build and push your operator's image:

  ```sh
  make docker-build docker-push IMG="example.com/memcached-operator:v0.0.1"
  ```

### OLM deployment

1. Install [OLM][doc-olm]:

  ```sh
  operator-sdk olm install
  ```

1. Bundle your operator, then build and push the bundle image (defaults to `example.com/memcached-operator-bundle:v0.0.1`):

  ```sh
  make bundle IMG="example.com/memcached-operator:v0.0.1"
  make bundle-build bundle-push BUNDLE_IMG="example.com/memcached-operator-bundle:v0.0.1"
  ```

1. Run your bundle. If your bundle image is hosted in a registry that is private and/or
has a custom CA, these [configuration steps][image-reg-config] must be complete.

  ```sh
  operator-sdk run bundle <some-registry>/memcached-operator-bundle:v0.0.1
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
  make deploy IMG="example.com/memcached-operator:v0.0.1"
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

### Run locally (outside the cluster)

This is recommended ONLY for development purposes

1. Run the operator:

  ```sh
  make install run
  ```

1. In a new terminal tab/window, create a sample Memcached custom resource:
  
  ```console
  $ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
  memcached.cache.example.com/memcached-sample created
  ```

1. Stop the operator by pressing `ctrl+c` in the terminal tab or window the operator is running in

## Next Steps

Read the [full tutorial][tutorial] for an in-depth walkthrough of building a Go operator.


[install-guide]:/docs/building-operators/golang/installation
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
[doc-olm]:/docs/olm-integration/tutorial-bundle/#enabling-olm
[tutorial]:/docs/building-operators/golang/tutorial/
