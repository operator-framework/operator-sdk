---
title: Quickstart for Helm-based Operators
linkTitle: Quickstart
weight: 100
description: A simple set of instructions to set up and run a Helm-based operator.
---

This guide walks through an example of building a simple nginx-operator powered by [Helm][helm-official] using tools and libraries provided by the Operator SDK.

## Prerequisites

- Go through the [installation guide][install-guide].
- User authorized with `cluster-admin` permissions.
- An accessible image registry for various operator images (ex. [hub.docker.com](https://hub.docker.com/signup),
[quay.io](https://quay.io/)) and be logged in in your command line environment.
  - `example.com` is used as the registry Docker Hub namespace in these examples.
  Replace it with another value if using a different registry or namespace.
  - The registry/namespace must be public, or the cluster must be provisioned with an
  [image pull secret][k8s-image-pull-sec] if the image namespace is private.


## Steps

1. Create a project directory for your project and initialize the project:

  ```sh
  mkdir nginx-operator
  cd nginx-operator
  operator-sdk init --domain example.com --plugins helm
  ```

1. Create a simple nginx API using Helm's built-in chart boilerplate (from `helm create`):

  ```sh
  operator-sdk create api --group demo --version v1alpha1 --kind Nginx
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

1. Run your bundle:

  ```sh
  operator-sdk run bundle example.com/memcached-operator-bundle:v0.0.1
  ```

1. Create a sample Nginx custom resource:

  ```console
  $ kubectl apply -f config/samples/demo_v1alpha1_nginx.yaml
  nginx.demo.example.com/nginx-sample created
  ```

1. Uninstall the operator:

  ```sh
  operator-sdk cleanup nginx-operator
  ```


### Direct deployment

1. Deploy your operator:

  ```sh
  make deploy IMG="example.com/memcached-operator:v0.0.1"
  ```

1. Create a sample Nginx custom resource:

  ```console
  $ kubectl apply -f config/samples/demo_v1alpha1_nginx.yaml
  nginx.demo.example.com/nginx-sample created
  ```

1. Uninstall the operator:

  ```sh
  make undeploy
  ```

## Next Steps

Read the [full tutorial][tutorial] for an in-depth walkthough of building a Helm operator.


[helm-official]:https://helm.sh/docs/
[install-guide]:/docs/building-operators/helm/installation
[doc-olm]:/docs/olm-integration/quickstart-bundle/#enabling-olm
[tutorial]:/docs/building-operators/helm/tutorial/
[k8s-image-pull-sec]:https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
