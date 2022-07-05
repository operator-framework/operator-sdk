---
title: Quickstart for Helm-based Operators
linkTitle: Quickstart
weight: 100
description: A simple set of instructions to set up and run a Helm-based operator.
---

This guide walks through an example of building a simple nginx-operator powered by [Helm][helm-official] using tools and libraries provided by the Operator SDK.

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
  make docker-build docker-push IMG="example.com/nginx-operator:v0.0.1"
  ```


### OLM deployment

1. Install [OLM][doc-olm]:

  ```sh
  operator-sdk olm install
  ```

1. Bundle your operator, then build and push the bundle image (defaults to `example.com/nginx-operator-bundle:v0.0.1`):

  ```sh
  make bundle IMG="example.com/nginx-operator:v0.0.1"
  make bundle-build bundle-push IMG="example.com/nginx-operator:v0.0.1"
  ```

1. Run your bundle. If your bundle image is hosted in a registry that is private and/or
has a custom CA, these [configuration steps][image-reg-config] must be complete.

  ```sh
  operator-sdk run bundle example.com/nginx-operator-bundle:v0.0.1
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
  make deploy IMG="example.com/nginx-operator:v0.0.1"
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

Read the [full tutorial][tutorial] for an in-depth walkthrough of building a Helm operator.


[helm-official]:https://helm.sh/docs/
[install-guide]:/docs/building-operators/helm/installation
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
[doc-olm]:/docs/olm-integration/tutorial-bundle/#enabling-olm
[tutorial]:/docs/building-operators/helm/tutorial/
