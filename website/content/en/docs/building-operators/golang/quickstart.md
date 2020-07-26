---
title: Golang Based Operator QuickStart
linkTitle: QuickStart
weight: 20
---

## Prerequisites

- [go][go_tool] version v1.13+.
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.3+.
- [operator-sdk][operator_install] v.0.19+
- Access to a Kubernetes v1.11.3+ cluster.

## Creating a Project

Create a directory, and then run the init command inside of it to generate a new project.
 
```sh
$ mkdir $GOPATH/src/memcached-operator
$ cd $GOPATH/src/memcached-operator
$ operator-sdk init 
```

## Creating an API

Let's create a new API with `(group/version)` as `cache/v1` and new Kind(CRD) Memcached on it:

```sh
$ operator-sdk create api --group cache --version v1 --kind Memcached
```

**Note** If you press `y` for Create Resource `[y/n]` and for Create Controller `[y/n]` then this will create the files `api/v1/memcached_types.go` where the API is defined and the `controllers/memcached_controller.go` where the reconciliation business logic will be done for the `Memcached` Kind(CRD).

## Applying the CRDs into the cluster:

To apply the `Memcached` Kind(CRD): 

```sh
$ make install
```

## Running it locally

To run the project out of the cluster:

```sh
$ make run
```

## Configuring your test environment

Projects are scaffolded with tests that requires certain Kubernetes server binaries be present locally. Run:

```sh
$ curl -sSLo setup_envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/kubebuilder/master/scripts/setup_envtest_bins.sh 
$ chmod +x setup_envtest.sh
$ ./setup_envtest.sh v1.18.2 v3.4.3
```

**Note:** More info can be found [here][env-test-setup].

## Building and Pushing the Project Image

To build and push your image to your repository :

```sh
$ make docker-build docker-push IMG=<some-registry>/<project-name>:tag
```

**Note** To allow the cluster pull the image the repository needs to be set as public. 

## Running it on Cluster

Deploy the project to the cluster:

```sh
$ make deploy IMG=<some-registry>/<project-name>:tag
```

## Applying the CR's into the cluster:

To create instances (CR's) of the `Memcached` Kind (CRD) in the same namespaced of the operator 

```sh
$ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml -n memcached-operator-system
```

## Uninstall CRDs

To delete your CRDs from the cluster:

```sh
$ make uninstall
```

## Undeploy Project

To undeploy and remove the manifests from the cluster. 

- Add the following target to your Makefile :

```sh
# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -
```

- Run `make undeploy`

## Next Step

Now, follow up the [Tutorial][tutorial] to better understand how it works by developing a demo project.

[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[operator_install]: /docs/installation/install-operator-sdk
[env-test-setup]: /docs/building-operators/golang/references/env-test-setup
[tutorial]: /docs/building-operators/golang/tutorial/ 
