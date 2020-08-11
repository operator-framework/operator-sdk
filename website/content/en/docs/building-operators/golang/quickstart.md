---
title: Quickstart for Go-based Operators
linkTitle: Quickstart
weight: 20
description: A simple set of instructions that demonstrates the basics of setting up and running Go-based operator.
---

This guide walks through an example of building a simple memcached-operator using tools and libraries provided by the Operator SDK.

## Prerequisites

- [Install operator-sdk][operator_install] and its prequisites.
- Access to a Kubernetes v1.11.3+ cluster (v1.16.0+ if using `apiextensions.k8s.io/v1` CRDs).
- User authorized with `cluster-admin` permissions

## Quickstart Steps

### Create a project

Create and change into a directory for your project. Then call `operator-sdk init`
with the Go plugin to initialize the project. 
 
```sh
mkdir memcached-operator
cd memcached-operator
operator-sdk init --domain=example.com --repo=github.com/example-inc/memcached-operator
```

### Create an API

Create a simple Memcached API:

```sh
operator-sdk create api --group cache --version v1 --kind Memcached --resource=true --controller=true
```

### Configuring your test environment

[Setup the `envtest` binaries and environment][envtest-setup] for your project.
Update your `test` Makefile target to the following:

```sh
# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out
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
```
kubectl logs deployment.apps/memcached-operator-controller-manager -n memcached-operator-system -c manager
```

## Clean up

Delete the CR to uninstall memcached:
```sh 
kubectl delete -f config/samples/cache_v1_memcached.yaml
```

Uninstall the operator and its CRDs:
```sh
kustomize build config/default | kubectl delete -f -
```

## Next Steps
Read the [tutorial][tutorial] for an in-depth walkthough of building a Go operator.

[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[operator_install]: /docs/installation/install-operator-sdk
[envtest-setup]: /docs/building-operators/golang/references/envtest-setup
[tutorial]: /docs/building-operators/golang/tutorial/ 

