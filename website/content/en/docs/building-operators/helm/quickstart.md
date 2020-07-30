---
title: Quickstart for Helm-based Operators
linkTitle: Quickstart
weight: 100
description: A simple set of instructions that demonstrates the basics of setting up and running a Helm-based operator.
---

This guide walks through an example of building a simple nginx-operator powered by [Helm][helm-official] using tools and libraries provided by the Operator SDK.

## Prerequisites

- [Install `operator-sdk`][operator_install] and its prequisites.
- Access to a Kubernetes v1.16.0+ cluster.

## Quickstart Steps

### Create a project

Create and change into a directory for your project. Then call `operator-sdk init`
with the Helm plugin to initialize the [base project layout][project_layout]:

```sh
mkdir nginx-operator
cd nginx-operator
operator-sdk init --plugins=helm
```

### Create an API

Create a simple nginx API using Helm's built-in chart boilerplate (from
`helm create`):

```sh
operator-sdk create api --group demo --version v1 --kind Nginx
```

### Build and push the operator image

Use the built-in Makefile targets to build and push your operator. Make
sure to define `IMG` when you call `make`:

```sh
make docker-build docker-push IMG=<some-registry>/<project-name>:<tag>
```

**NOTE**: To allow the cluster pull the image the repository needs to be
          set as public or you must configure an image pull secret.


### Run the operator

Install the CRD and deploy the project to the cluster. Set `IMG` with
`make deploy` to use the image you just pushed:

```sh
make install
make deploy IMG=<some-registry>/<project-name>:<tag>
```

### Create a sample custom resource

Create a sample CR:
```sh
kubectl apply -f config/samples/demo_v1_nginx.yaml
```

Watch for the CR to trigger the operator to deploy the nginx deployment
and service:
```sh
kubectl get all -l "app.kubernetes.io/instance=nginx-sample"
```

### Clean up

Delete the CR to uninstall the release:
```sh
kubectl delete -f config/samples/demo_v1_nginx.yaml
```

Use `make undeploy` to uninstall the operator and its CRDs:
```sh
make undeploy
```

## Next Steps

Read the [tutorial][tutorial] for an in-depth walkthough of building a Helm operator.

[operator_install]: /docs/installation/install-operator-sdk
[project_layout]: /docs/building-operators/helm/reference/project_layout/
[tutorial]: /docs/building-operators/helm/tutorial/
[helm-official]: https://helm.sh/docs/
