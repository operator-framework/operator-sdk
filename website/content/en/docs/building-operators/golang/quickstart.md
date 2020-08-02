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
- User logged with admin permission. See [how to grant yourself cluster-admin privileges or be logged in as admin][role-based-access-control]

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

Projects are scaffolded with tests that utilize the [`envtest`][env-test]
library, which requires certain Kubernetes server binaries to be present locally:

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
curl -fsL "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.16.4-${OS}-${ARCH}.tar.gz" -o kubebuilder-tools
tar -zvxf kubebuilder-tools
sudo mv kubebuilder/ /usr/local/kubebuilder
```

**Note:** More info can be found [here][env-test-setup].

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

Delete the CR to uninstall the release:
```sh 
kubectl delete -f config/samples/cache_v1_memcached.yaml
```

To uninstall the operator and its CRDs:
```sh
kustomize build config/default | kubectl delete -f -
```

## Next Steps
Read the [tutorial][tutorial] for an in-depth walkthough of building a Go operator.

[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[operator_install]: /docs/installation/install-operator-sdk
[env-test-setup]: /docs/building-operators/golang/references/env-test-setup
[tutorial]: /docs/building-operators/golang/tutorial/ 
[env-test]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest
[role-based-access-control]: https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#iam-rolebinding-bootstrap
