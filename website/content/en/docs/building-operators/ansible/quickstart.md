---
title: Ansible Operator QuickStart
linkTitle: QuickStart
weight: 2
---
## Prerequisites

- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.3+.
- [Ansible Operator SDK Installation][ansible-operator-install] v1.0.0+
- Access to a Kubernetes v1.11.3+ cluster.

## Creating a Project

Create a directory, and then run the init command inside of it to generate a new project.
 
```sh
$ mkdir $GOPATH/src/memcached-operator
$ cd $GOPATH/src/memcached-operator
$ operator-sdk init --plugins=ansible
```

## Creating an API

Let's create a new API with a default role for it:

```sh
$ operator-sdk create api --group cache --version v1 --kind Memcached --generate-role 
```

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

```sh
$ make undeploy
```

## Next Step

Now, follow up the [Tutorial][tutorial] to better understand how it works by developing a demo project.

[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[ansible-operator-install]: /docs/building-operators/ansible/installation
[helm-repo-add]: https://helm.sh/docs/helm/helm_repo_add
[helm-chart-memcached]: https://github.com/helm/charts/tree/master/stable/memcached
[tutorial]: /docs/building-operators/ansible/tutorial/ 