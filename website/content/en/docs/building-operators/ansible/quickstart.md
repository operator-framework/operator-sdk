---
title: Quickstart for Ansible-based Operators
linkTitle: Quickstart
weight: 2
description: A simple set of instructions that demonstrates the basics of setting up and running a Ansible-based operator.
---

This guide walks through an example of building a simple memcached-operator powered by [Ansible][ansible-link] using tools and libraries provided by the Operator SDK.

## Prerequisites

- [Install `operator-sdk`][operator_install] and the [Ansible prequisites][ansible-operator-install] 
- Access to a Kubernetes v1.16.0+ cluster.
- User authorized with `cluster-admin` permissions.

## Quickstart Steps

### Create a project

Create and change into a directory for your project. Then call `operator-sdk init`
with the Ansible plugin to initialize the [base project layout][layout-doc]:

```sh
mkdir memcached-operator
cd memcached-operator
operator-sdk init --plugins=ansible --domain=example.com
```

### Create an API

Let's create a new API with a role for it:

```sh
operator-sdk create api --group cache --version v1 --kind Memcached --generate-role 
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
kubectl apply -f config/samples/cache_v1_memcached.yaml
```

Watch for the CR be reconciled by the operator:
```sh
kubectl logs deployment.apps/memcached-operator-controller-manager -n memcached-operator-system -c manager
```

### Clean up

Delete the CR to uninstall memcached:
```sh
kubectl delete -f config/samples/cache_v1_memcached.yaml 
```

Use `make undeploy` to uninstall the operator and its CRDs:
```sh
make undeploy
```

## Next Steps

Read the [tutorial][tutorial] for an in-depth walkthough of building a Ansible operator.

[operator_install]: /docs/installation/install-operator-sdk
[layout-doc]:../reference/scaffolding
[tutorial]: /docs/building-operators/ansible/tutorial/
[ansible-link]: https://www.ansible.com/ 
