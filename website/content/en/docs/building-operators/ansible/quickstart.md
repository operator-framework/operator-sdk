---
title: Quickstart for Ansible-based Operators
linkTitle: Quickstart
weight: 2
description: A simple set of instructions to set up and run an Ansible-based operator.
---

This guide walks through an example of building a simple memcached-operator powered by [Ansible][ansible-link] using tools and libraries provided by the Operator SDK.

## Prerequisites

- Go through the [installation guide][install-guide].
- Make sure your user is authorized with `cluster-admin` permissions.
- Have an accessible image registry for various operator images (ex. [hub.docker.com](https://hub.docker.com/signup),
[quay.io](https://quay.io/)) and be logged in to your command line environment.
  - `example.com` is used as the registry Docker Hub namespace in these examples.
  Replace it with another value if using a different registry or namespace.
  - [Authentication and certificates][image-reg-config] if the registry is private or uses a custom CA.


## Steps

1. Create a project directory for your project and initialize the project:

  ```sh
  mkdir memcached-operator
  cd memcached-operator
  operator-sdk init --domain example.com --plugins ansible
  ```

1. Create a simple Memcached API:

  ```sh
  operator-sdk create api --group cache --version v1alpha1 --kind Memcached --generate-role
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
  make bundle-build bundle-push
  ```

1. Run your bundle. If your bundle image is hosted in a registry that is private and/or
has a custom CA, these [configuration steps][image-reg-config] must be complete.

  ```sh
  operator-sdk run bundle example.com/memcached-operator-bundle:v0.0.1
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


## Next Steps

Read the [full tutorial][tutorial] for an in-depth walkthrough of building an Ansible operator.


[ansible-link]:https://www.ansible.com/
[install-guide]:/docs/building-operators/ansible/installation
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
[doc-olm]:/docs/olm-integration/tutorial-bundle/#enabling-olm
[tutorial]:/docs/building-operators/ansible/tutorial/
