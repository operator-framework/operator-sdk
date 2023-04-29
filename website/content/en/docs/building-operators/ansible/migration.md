---
link: Migrating Projects from pre-v1.0.0 to the latest release
linkTitle: Migrating from pre-v1.0.0 to latest
weight: 200
description: Instructions for migrating an Ansible-based operator built prior to `v1.0.0` to use a Kubebuilder-style.
---

## Overview

The motivations for the new layout are related to bringing more flexibility to users and part of the process to Integrating Kubebuilder and Operator SDK. Because of this integration you may be referred to the Kubebuilder documentation [https://book.kubebuilder.io/](https://book.kubebuilder.io/) for more information about certain topics. When using this document just remember to replace `$ kubebuilder <command>` with `$ operator-sdk <command>`.

**Note:** It is recommended that you have your project upgraded to the latest SDK v1.y release version before following the steps in this guide to migrate to the new layout. However, the steps might work from previous versions as well. In this case, if you find an issue which is not covered here then check the previous [Migration Guides][migration-doc] which might help out.

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
- Updated metrics configuration using [kube-auth-proxy][kube-auth-proxy], a `--metrics-bind-address` flag, and [kustomize][kustomize]-based deployment of a Kubernetes `Service` and prometheus operator `ServiceMonitor`
- Preliminary support for CLI plugins. For more info see the [plugins design document][plugins-phase1-design-doc]
- A `PROJECT` configuration file to store information about GVKs, plugins, and help the CLI make decisions.

Generated files with the default API versions:

- `apiextensions/v1` for generated CRDs (`apiextensions/v1beta1` was deprecated in Kubernetes `1.16` and will be removed in `1.22`)
- `admissionregistration.k8s.io/v1` for webhooks (`admissionregistration.k8s.io/v1beta1` was deprecated in Kubernetes `1.16` and will be removed in `1.22` )

## How to migrate

The easy migration path is to initialize a new project, re-recreate APIs, then copy pre-v1.0.0 configuration files into the new project.

### Prerequisites

- Go through the [installation guide][install-guide].
- Make sure your user is authorized with `cluster-admin` permissions.
- An accessible image registry for various operator images (ex. [hub.docker.com](https://hub.docker.com/signup),
[quay.io](https://quay.io/)) and be logged in to your command line environment.
  - `example.com` is used as the registry Docker Hub namespace in these examples.
  Replace it with another value if using a different registry or namespace.
  - [Authentication and certificates][image-reg-config] if the registry is private or uses a custom CA.

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
  # TODO(user): Add fields here
  size: 3
```

### Migrating `watches.yaml`

Update the `watches.yaml` file with your `roles/playbooks` and check if you have custom options in the `watches.yaml` file of your existing project. If so, update the new `watches.yaml` file to match.

In our example, we will replace `# FIXME: Specify the role or playbook for this resource.` with our previous role and it will look like:

```yaml
---
# Use the 'create api' subcommand to add watches to this file.
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: memcached
#+kubebuilder:scaffold:watch
```

**NOTE**: Do not remove the `+kubebuilder:scaffold:watch` [marker][marker]. It allows the tool to update the watches file when new APIs are created.

Additionally pre-1.0 the `reconcilePeriod` parameter was an integer representing the maximum time in seconds before a reconcile would be triggered.
With 1.0, it was changed to a string representing the maximum duration before a reconcile will be triggered. Appending an `s` to your `reconcilePeriod`
will set the duration unit to seconds and match the old behavior.

so for example a resource set to requeue every hour:

```yaml
---
# Use the 'create api' subcommand to add watches to this file.
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: memcached
  reconcilePeriod: 3600
#+kubebuilder:scaffold:watch
```

would become

```yaml
---
# Use the 'create api' subcommand to add watches to this file.
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: memcached
  reconcilePeriod: 3600s
#+kubebuilder:scaffold:watch
```

and the values `60m` and `1h` would be equivalent to the `3600s` that is used.

### Migrating your Molecule tests

If you are using [Molecule][molecule] in your project will be required to port the tests for the new layout.  

See that default structure changed from:

```
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

```yaml
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

### Checking RBAC Permissions

In your new project, roles are automatically generated in `config/rbac/role.yaml`.
If you modified these permissions manually in `deploy/role.yaml` in your existing
project, you need to re-apply them in `config/rbac/role.yaml`.

<!--
todo(camilamacedo86): Create an Ansible operator scope document.
https://github.com/operator-framework/operator-sdk/issues/3447
-->

New projects are configured to watch all namespaces by default, so they need a `ClusterRole` to have the necessary permissions. Ensure that `config/rbac/role.yaml` remains a `ClusterRole` if you want to retain the default behavior of the new project conventions.

The following rules were used in earlier versions of ansible-operator to automatically create and manage services and `servicemonitors` for metrics collection. If your operator's don't require these rules, they can safely be left out of the new `config/rbac/role.yaml` file:

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

##### Updating your ServiceAccount

New Ansible projects come with a ServiceAccount `controller-manager` in `config/rbac/service_account.yaml`.
Your project's RoleBinding and ClusterRoleBinding subjects, and Deployment's `spec.template.spec.serviceAccountName`
that reference a ServiceAccount already refer to this new name. When you run `make deploy`,
your project's name will be prepended to `controller-manager`, making it unique within a namespace,
much like your old `deploy/service_account.yaml`. If you wish to use the old ServiceAccount,
make sure to update all RBAC bindings and your manager Deployment.

### Configuring your Operator

If your existing project has customizations in `deploy/operator.yaml`, they need to be ported to
`config/manager/manager.yaml`. If you are passing custom arguments in your deployment, make sure to also update `config/default/auth_proxy_patch.yaml`.

Note that the following environment variables are no longer used.

- `OPERATOR_NAME` is deprecated. It is used to define the name for a leader election config map. Operator authors should begin using `--leader-election-id` instead.
- `POD_NAME` has been removed. It was used to enable a particular pod to hold the leader election lock when the Ansible operator used the leader for life mechanism. Ansible operator now uses controller-runtime's leader with lease mechanism.

### Exporting metrics

If you are using metrics and would like to keep them exported you will need to configure
it in the `config/default/kustomization.yaml`. Please see the [metrics][metrics] doc to know how you can perform this setup.

The default port used by the metric endpoint binds to was changed from `:8383` to `:8080`. To continue using port `8383`, specify `--metrics-bind-address=:8383` when you start the operator.

### Verify the migration

The project can now be deployed on the cluster by running the command:

```sh
make deploy IMG=example.com/memcached-operator:v0.0.1
```

You can troubleshoot your deployment by checking the container logs:
```sh
kubectl logs deployment.apps/memcached-operator-controller-manager -n memcached-operator-system -c manager
```

For further steps regarding the deployment of the operator, creation of custom resources, and cleaning up of resources, see the [tutorial][tutorial-deploy].

[install-guide]: /docs/building-operators/ansible/installation
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
[kustomize]: https://github.com/kubernetes-sigs/kustomize
[kube-auth-proxy]: https://github.com/brancz/kube-rbac-proxy
[metrics]: https://book.kubebuilder.io/reference/metrics.html?highlight=metr#metrics
[marker]: https://book.kubebuilder.io/reference/markers.html?highlight=markers#marker-syntax
[molecule]: https://molecule.readthedocs.io/
[testing-guide]: /docs/building-operators/ansible/testing-guide
[migration-doc]: /docs/upgrading-sdk-version/
[tutorial-deploy]: /docs/building-operators/ansible/tutorial/#run-the-operator
