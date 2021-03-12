---
link: Migrating Projects from pre-v1.0.0 to the latest release
linkTitle: Migrating from pre-v1.0.0 to latest
weight: 200
description: Instructions for migrating an Helm-based operator built prior to `v1.0.0` to use a Kubebuilder-style.
---

## Overview

The v1.0 release improves upon prior `operator-sdk` releases with a new project structure and CLI, each of which enhances project extensibility and customizability. These design changes are influenced by [`kubebuilder`](https://book.kubebuilder.io/).

**Note:** It is recommended that you have your project upgraded to the latest SDK release version (0.19.x+) before following the steps in this guide to migrate to the new layout. However, the steps might work from previous versions as well. In this case, if you find an issue which is not covered here then check the previous [Migration Guides][migration-doc] which might help out.

### What was changed

- The `deploy` directory was replaced with the `config` directory including a new layout of Kubernetes manifests files:
    * CRD manifests in `deploy/crds/` are now in `config/crd/bases`
    * CR manifests in `deploy/crds/` are now in `config/samples`
    * Controller manifest `deploy/operator.yaml` is now in `config/manager/manager.yaml`
    * RBAC manifests in `deploy` are now in `config/rbac/`

- `build/Dockerfile` is moved to `Dockerfile` in the project root directory

### What is new

Scaffolded projects now use:

- [kustomize][kustomize] to manage Kubernetes resources needed to deploy your operator
- A `Makefile` with helpful targets for build, test, and deployment, and to give you flexibility to tailor things to your project's needs
- Updated metrics configuration using [kube-auth-proxy][kube-auth-proxy], a `--metrics-addr` flag, and [kustomize][kustomize]-based deployment of a Kubernetes `Service` and prometheus operator `ServiceMonitor`
- Preliminary support for CLI plugins. For more info see the [plugins design document][plugins-phase1-design-doc]
- A `PROJECT` configuration file to store information about GVKs, plugins, and help the CLI make decisions.

Generated files with the default API versions:

- `apiextensions/v1` for generated CRDs (`apiextensions/v1beta1` was deprecated in Kubernetes `1.16` and will be removed in `1.22`)
- `admissionregistration.k8s.io/v1` for webhooks (`admissionregistration.k8s.io/v1beta1` was deprecated in Kubernetes `1.16` and will be removed in `1.22` )

## How to migrate

The easy migration path is to initialize a new project, re-recreate APIs, then copy pre-v1.0.0 configuration files into the new project.

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
mkdir nginx-operator
cd nginx-operator
operator-sdk init --plugins=helm --domain=example.com
```

Now that we have our new project initialized, we need to re-create each of our APIs.
Using our API example from earlier (`cache.example.com`), we'll use `cache` for the
`--group` flag.

For `--version` and `--kind`, we use `spec.versions[0].name` and `spec.names.kind`, respectively.

For each API in the existing project, run:
```sh
operator-sdk create api \
    --group=cache \
    --version=<version> \
    --kind=<Kind> \
    --helm-chart=<path_to_existing_project>/helm-charts/<chart>
```

### Migrating your Custom Resource samples

Update the CR manifests in `config/samples` with the values of the CRs in your existing project which are in `deploy/crds/<group>_<version>_<kind>_cr.yaml`

### Migrating `watches.yaml`

Check if you have custom options in the `watches.yaml` file of your existing project. If so, update the new `watches.yaml` file to match. In our example, it will look like:

```yaml
# Use the 'create api' subcommand to add watches to this file.
- group: example.com
  version: v1alpha1
  kind: Nginx
  chart: helm-charts/nginx
#+kubebuilder:scaffold:watch
```

**NOTE**: Do not remove the `+kubebuilder:scaffold:watch` [marker][marker]. It allows the tool to update the watches file when new APIs are created.

### Checking the Permissions (RBAC)

In your new project, roles are automatically generated in `config/rbac/role.yaml`.
If you modified these permissions manually in `deploy/role.yaml` in your existing
project, you need to re-apply them in `config/rbac/role.yaml`.

New projects are configured to watch all namespaces by default, so they need a `ClusterRole` to have the necessary permissions. Ensure that `config/rbac/role.yaml` remains a `ClusterRole` if you want to retain the default behavior of the new project conventions.

<!--
todo(camilamacedo86): Create an Ansible operator scope document.
https://github.com/operator-framework/operator-sdk/issues/3447
-->

The following rules were used in earlier versions of helm-operator to automatically create and manage services and servicemonitors for metrics collection. If your operator's charts don't require these rules, they can safely be left out of the new `config/rbac/role.yaml` file:

```yaml  
  - apiGroups:
    - monitoring.coreos.com
    resources:
    - servicemonitors
    verbs:
    - get
    - create
  - apiGroups:
    - apps
    resourceNames:
    - memcached-operator
    resources:
    - deployments/finalizers
    verbs:
    - update
```

### Configuring your Operator

If your existing project has customizations in `deploy/operator.yaml`, they need to be ported to
`config/manager/manager.yaml`. If you are passing custom arguments in your deployment, make sure to also update `config/default/auth_proxy_patch.yaml`.

Note that the following environment variables are no longer used.

- `OPERATOR_NAME` is deprecated. It is used to define the name for a leader election config map. Operator authors should begin using `--leader-election-id` instead.
- `POD_NAME` was used to enable a particular pod to hold the leader election lock when the Helm operator used the leader for life mechanism. Helm operator now uses controller-runtime's leader with lease mechanism, and `POD_NAME` is no longer necessary.

## Exporting metrics

If you are using metrics and would like to keep them exported you will need to configure
it in the `config/default/kustomization.yaml`. Please see the [metrics][metrics] doc to know how you can perform this setup.

The default port used by the metric endpoint binds to was changed from `:8383` to `:8080`. To continue using port `8383`, specify `--metrics-addr=:8383` when you start the operator.

## Checking the changes

Finally, follow the steps in the ["run the Operator"][run-the-operator] section to verify your project is running.

[quickstart]: /docs/building-operators/helm/quickstart
[integration-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
[run-the-operator]: /docs/building-operators/helm/tutorial#run-the-operator
[kustomize]: https://github.com/kubernetes-sigs/kustomize
[kube-auth-proxy]: https://github.com/brancz/kube-rbac-proxy
[metrics]: https://book.kubebuilder.io/reference/metrics.html?highlight=metr#metrics
[marker]: https://book.kubebuilder.io/reference/markers.html?highlight=markers#marker-syntax
[migration-doc]: /docs/upgrading-sdk-version/