---
title: Migrating Legacy Projects
linkTitle: Migrating Legacy Projects
weight: 3
---

This guide walks through an example of migrating a simple nginx-operator which was built by following the [legacy quick-start][quickstart-legacy] to the new layout.

## Overview

The motivations for the new layout are related to bringing more flexibility to users and 
part of the process to [Integrating Kubebuilder and Operator SDK][integration-doc].

### What was changed
 
The `deploy` directory was replaced with the `config` directory including a new layout of Kubernetes manifests files
- CRD's manifests in `deploy/crds/` are now in `config/crd/bases`
- CR's manifests in `deploy/crds/` are now in `config/samples`
- Controller manifest `deploy/operator.yaml` was replaced for `config/manager/manager.yaml` 
- RBCA's manifests in `deploy` are in `config/rbac/`

The `build/Dockerfile` directory was replaced by the `Dockerfile` in the root directory

### What is new

Projects are now scaffold using:

- [kustomize][kustomize] to manage Kubernetes resources needed to deploy your operator
- A `Makefile` with helpful targets for build, test, and deployment, and to give you flexibility to tailor things to your project's needs
- Updated metrics configuration using [kube-auth-proxy][kube-auth-proxy], a `--metrics-addr` flag, and [kustomize][kustomize]-based deployment of a Kubernetes `Service` and prometheus operator `ServiceMonitor`

## How to migrate

The easy migration path is to create a new project from the scratch and let the tool scaffold the files properly and then, 
just replace with your customizations and implementations. Following an example. 
 
### Creating a new project

In Kubebuilder-style projects, CRD groups are defined using two different flags
(`--group` and `--domain`).

When we initialize a new project, we need to specify the domain that _all_ APIs in
our project will share, so before creating the new project, we need to determine which
domain we're using for the APIs in our existing project.

To determine the domain, look at the `spec.group` field in your CRDs in the
`deploy/crds` directory.

The domain is everything after the first DNS segment. Using `cache.example.com` as an
example, the `--domain` would be `example.com`.

So let's create a new project with the same domain (`example.com`):

```sh
$ mkdir nginx-operator
$ cd nginx-operator
$ operator-sdk init --plugins=helm --domain=example.com
```

Now that we have our new project initialized, we need to re-create each of our APIs. 
Using our API example from earlier (`cache.example.com`), we'll use `cache` for the
`--group` flag.

For each API in the existing project, run:
```sh
$ operator-sdk create api \
    --group=cache \
    --api-version=<apiVersion> \
    --kind=<Kind> \
    --helm-chart=<path_to_existing_project>/helm-charts/<chart>
```

**Note** Ensure that you use the same values for the flags to recreate the same Helm Chart and API's. If you have
more than one chart or API's you can add them via `operator-sdk create api` command. For further information check the [quick-start][quickstart]. 
 
### Replacing the content

- Update the CR manifests in `config/samples` with the values of the CR's in your old project which are in `deploy/crds/`
- Check if you have customizations options in the `watch.yaml` file of your previous project and then, update the new `watch.yaml` file with the same ones
- Ensure that all roles configured in the `/deploy/roles.yaml` will be applied in the new project in the file `config/rbac/role.yaml`
- If you have customizations in your `helm-charts` then, apply them in the new `helm-charts`. Note that this directory was not changed at all.

## Exporting metrics 

If you are using metrics and would like to keep them exported you will need to configure 
it in the `config/default/kustomization.yaml`. Please see the [metrics][metrics] doc to know how you can perform this setup. 

The default port used by the metric endpoint binds to was changed from `:8383` to `:8080`. To continue using port `8383`, specify `--metrics-addr=:8383` when you start the operator. 

### Checking the changes

Now, follow the steps in the section [Build and run the operator][build-run-quick] to verify your project is running. 

[quickstart-legacy]: https://v0-19-x.sdk.operatorframework.io/docs/helm/quickstart/
[quickstart]: /docs/building-operators/helm/quickstart
[integration-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
[build-run-quick]: /docs/building-operators/helm/quickstart#build-and-run-the-operator
[kustomize]: https://github.com/kubernetes-sigs/kustomize 
[kube-auth-proxy]: https://github.com/brancz/kube-rbac-proxy 
[metrics]: https://book.kubebuilder.io/reference/metrics.html?highlight=metr#metrics
