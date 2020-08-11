---
title: Migrating Legacy Projects
linkTitle: Migrating Projects to 1.0.0+
weight: 6
description: Instructions for migrating a Ansible-based project built prior to 1.0.0 to use the new Kubebuilder-style layout.
---

## Overview

The motivations for the new layout are related to bringing more flexibility to users and 
part of the process to [Integrating Kubebuilder and Operator SDK][integration-doc].

### What was changed
 
- The `deploy` directory was replaced with the `config` directory including a new layout of Kubernetes manifests files:
    * CRD manifests in `deploy/crds/` are now in `config/crd/bases`
    * CR manifests in `deploy/crds/` are now in `config/samples`
    * Controller manifest `deploy/operator.yaml` is now in `config/manager/manager.yaml` 
    * RBAC manifests in `deploy` are now in `config/rbac/`
    
- `build/Dockerfile` is moved to `Dockerfile` in the project root directory
- The `molecule/` directory is now more aligned to Ansible and the new Layout

### What is new

Scaffolded projects now use:

- [kustomize][kustomize] to manage Kubernetes resources needed to deploy your operator
- A `Makefile` with helpful targets for build, test, and deployment, and to give you flexibility to tailor things to your project's needs
- Updated metrics configuration using [kube-auth-proxy][kube-auth-proxy], a `--metrics-addr` flag, and [kustomize][kustomize]-based deployment of a Kubernetes `Service` and prometheus operator `ServiceMonitor`

## How to migrate

The easy migration path is to a project from the scratch and let the tool scaffold the new layout. Then, add your customizations and implementations. See below for an example.
 
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
mkdir memcached-operator
cd memcached-operator
operator-sdk init --plugins=ansible --domain=example.com
```

Now that we have our new project initialized, we need to recreate each of our APIs. 
Using our API example from earlier (`cache.example.com`), we'll use `cache` for the
`--group` flag.

For `--version` and `--kind`, we use `spec.versions[0].name` and `spec.names.kind`, respectively.

For each API in the existing project, run:

```sh
operator-sdk create api \
    --group=cache \
    --version=v1 \
    --kind=Memcached 
```

Running the above command creates an empty `roles/<kind>`. We can copy over the content of our old `roles/<kind>` to the new one.   
 
### Migrating your Custom Resource samples

Update the CR manifests in `config/samples` with the values of the CRs in your existing project which are in `deploy/crds/<group>_<version>_<kind>_cr.yaml` In our example 
the `config/samples/cache_v1alpha1_memcached.yaml` will look like:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: memcached-sample
spec:
  # Add fields here
  size: 3
```

### Migrating `watches.yaml

Update the `watches.yaml` file with your `roles/playbooks` and check if you have custom options in the `watches.yaml` file of your existing project. If so, update the new `watches.yaml file to match. 

In our example, we will replace `# FIXME: Specify the role or playbook for this resource.` with our previous role and it will look like:

```yaml
---
# Use the 'create api' subcommand to add watches to this file.
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: memcached 
# +kubebuilder:scaffold:watch
```

**NOTE**: Do not remove the `+kubebuilder:scaffold:watch` [marker][marker]. It allows the tool to update the watches file when new APIs are created. 

### Migrating your Molecule tests

If you are using [Molecule][molecule] in your project will be required to port the tests for the new layout.  

See that default structure changed from:

```sh
├── cluster
│   ├── converge.yml
│   ├── create.yml
│   ├── destroy.yml
│   ├── molecule.yml
│   ├── prepare.yml
│   └── verify.yml
├── default
│   ├── converge.yml
│   ├── molecule.yml
│   ├── prepare.yml
│   └── verify.yml
├── templates
│   └── operator.yaml.j2
└── test-local
    ├── converge.yml
    ├── molecule.yml
    ├── prepare.yml
    └── verify.yml

```

To: 

```
├── default
│   ├── converge.yml
│   ├── create.yml
│   ├── destroy.yml
│   ├── kustomize.yml
│   ├── molecule.yml
│   ├── prepare.yml
│   ├── tasks
│   │   └── foo_test.yml
│   └── verify.yml
└── kind
    ├── converge.yml
    ├── create.yml
    ├── destroy.yml
    └── molecule.yml
```

Ensure that the `provisioner.host_vars.localhost` has the following `host_vars`: 

```
....
    host_vars:
      localhost:
        ansible_python_interpreter: '{{ ansible_playbook_python }}'
        config_dir: ${MOLECULE_PROJECT_DIRECTORY}/config
        samples_dir: ${MOLECULE_PROJECT_DIRECTORY}/config/samples
        operator_image: ${OPERATOR_IMAGE:-""}
        operator_pull_policy: ${OPERATOR_PULL_POLICY:-"Always"}
        kustomize: ${KUSTOMIZE_PATH:-kustomize}
...
```

For more information read the [Testing with Molecule][testing-guide].

### Checking the Permissions (RBAC)

In your new project, roles are automatically generated in `config/rbac/role.yaml`.
If you modified these permissions manually in `deploy/role.yaml` in your existing
project, you need to re-apply them in `config/rbac/role.yaml`.

New projects are configured to watch all namespaces by default, so they need a `ClusterRole` to have the necessary permissions. Ensure that `config/rbac/role.yaml` remains a `ClusterRole` if you want to retain the default behavior of the new project conventions. For further information refer to the [operator scope][operator-scope] documentation.  

The following rules were used in earlier versions of anisible-operator to automatically create and manage services and `servicemonitors` for metrics collection. If your operator's don't require these rules, they can safely be left out of the new `config/rbac/role.yaml` file:

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
- `POD_NAME` has been removed. It was used to enable a particular pod to hold the leader election lock when the Ansible operator used the leader for life mechanism. Ansible operator now uses controller-runtime's leader with lease mechanism.

## Exporting metrics 

If you are using metrics and would like to keep them exported you will need to configure 
it in the `config/default/kustomization.yaml`. Please see the [metrics][metrics] doc to know how you can perform this setup. 

The default port used by the metric endpoint binds to was changed from `:8383` to `:8080`. To continue using port `8383`, specify `--metrics-addr=:8383` when you start the operator. 

## Checking the changes

Finally, follow the steps in the section [Build and run the Operator][build-and-run-the-operator] to verify your project is running. 

Note that, you also can troubleshooting by checking the container logs. 
E.g `$ kubectl logs deployment.apps/memcached-operator-controller-manager -n memcached-operator-system -c manager`  

[quickstart-legacy]: https://v0-19-x.sdk.operatorframework.io/docs/ansible/quickstart/
[quickstart]: /docs/building-operators/ansible/quickstart
[integration-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
[build-and-run-the-operator]: /docs/building-operators/ansible/tutorial/#deploy-the-operator
[kustomize]: https://github.com/kubernetes-sigs/kustomize 
[kube-auth-proxy]: https://github.com/brancz/kube-rbac-proxy 
[metrics]: https://book.kubebuilder.io/reference/metrics.html?highlight=metr#metrics
[marker]: https://book.kubebuilder.io/reference/markers.html?highlight=markers#marker-syntax
[operator-scope]: /docs/building-operators/golang/operator-scope
[molecule]: https://molecule.readthedocs.io/en/latest/# 
[testing-guide]: /docs/building-operators/ansible/testing-guide
