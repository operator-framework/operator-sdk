---
title: Ansible Based Operator Scaffolding
linkTitle: Scaffolding
weight: 20
---

A new Ansible operator project can be created using a command that looks like
the following:

```
operator-sdk init --plugins ansible \
  --domain=my.domain \
  --group=apps --version=v1alpha1 --kind=AppService \
  --generate-playbook \
  --generate-role
```

The new project directory has many generated folders and files. The following
table describes a basic rundown of each generated file/directory.

| File/Folders   | Purpose |
| :---           | :---    |
| Dockerfile | The Dockerfile for building the container image for the operator. |
| Makefile | Contains make targets for building, publishing, deploying the container image that wraps the operator binary, and make targets for installing and uninstalling the CRD. |
| PROJECT | A YAML file containing meta information for the operator. |
| config/crd | The base CRD files and the kustomization settings. |
| config/default | Collects all operator manifests for deployment, used by `make deploy`. |
| config/manager | The controller manager deployment. |
| config/prometheus | The ServiceMonitor resource for monitoring the operator. |
| config/rbac | The role, role binding for leader election and authentication proxy. |
| config/samples | The sample resources created for the CRDs. |
| config/testing | Some sample configurations for testing. |
| playbooks/ | A subdirectory for the playbooks to run. |
| roles/ | A subdirectory for the roles tree to run. |
| watches.yaml | The Group, Version, and Kind of the resources to watch, and the Ansible invocation method. New entries are added via the 'create api' command. |
| requirements.yml | A YAML file containing the Ansible collections and role dependencies to install during build. |
| molecule/ | The [Molecule](https://molecule.readthedocs.io/) scenarios for end-to-end testing of your role and operator |


## The Deployment

The default Deployment manifest generated for the operator can be found in the
`config/manager/manager.yaml` file. By default, the Deployment is named as
'controller-manager'. It contains a single container named 'manager', and it
may pick up a sidecar patch from the `config/default` directory. The
Deployment will create a single Pod.

For the container in the Pod, there are a few things to note. The default
Deployment contains a placeholder for the container image to use, so you
cannot create a meaningful operator using the YAML file directly. To deploy
the operator, you will run `make deploy IMG=<IMG>`. The image name and tag
are then patched using kustomize.

### The Volume Mount Path

The default EmptyDir volume mounted at `/tmp/ansible-operator/runner` is used
to serve the [input directory][runner_input_dir] in ansible-runner's terms.
The mount path can *NOT* be changed to other paths, or else the Operator will
fail to communicate with ansible-runner.

### The Environment Variables

You can customize the behavior of the Ansible operator by specifying the
environment variables for the container. Please refer to the
[Ansible Documentation][ansible_env] for a list of environment variables that
can be used to tune the behavior of the Ansible engine.

In addition to the Ansible environment variables, Operator SDK also support
some special environment variables:

- `WATCH_NAMESPACE`: This is the namespace your operator will watch for resource
  changes, i.e. resource create, update or delete operations. In the scaffolded
  operator Deployment, this is set to the same namespace in which the
  operator is deployed. This variable can be set to one of the following types
  of values:

  - '': An empty string means that the operator will watch all namespaces.
    This is the default value if the `WATCH_NAMESPACE` environment variable is
    not set. It is especially useful for watching cluster-scoped resources.

  - `foo`: The operator will watch the namespace named `foo`.
    This is the setting you will use if you are operating a namespaced resource
    which is deployed into a specific namespace.

  - `foo,bar`: The operator checks the value of the environment variable and
    realizes that it is a comma-separated list. This means the operator will
    watch for resources in each of the listed namespaces.


- `ANSIBLE_DEBUG_LOGS`: A boolean value for toggling the Ansible output during
  reconciliation. When set to True, the operator dumps the Ansible result into
  its standard output. 

- `ANSIBLE_ROLES_PATH`: The parent path(s) for the Ansible roles. When there
  are more than one path to set, you can use ":" to separate them. Given a
  path `/opt/foo` and a role name `bar`, the Ansible operator will check if
  the Ansible role can be found in either `/opt/foo/bar` or
  `/opt/foo/roles/bar`.

  This value overrides the setting from the `ansible-roles-path` flag.

- `ANSIBLE_COLLECTIONS_PATH`: The base path for the Ansible collections which
  defaults to `~/.ansible/collections` or `/usr/share/ansible/collections`
  when `ANSIBLE_COLLECTIONS_PATH` is not explicitly specified. When a fully
  qualified collection name in the [watches][watches_doc] file, the Ansible
  operator checks if the specified collection can found under the base path
  that can be customized using this variable. Suppose you have
  `ANSIBLE_COLLECTIONS_PATH` set to `/foo` and the fully qualified collection
  name set to `example.com.bar`, the Ansible operator searches for the roles
  under `/foo/ansible_collections/example/com/roles/bar`.

  This value takes precedence over the `--ansible-collections-path` flag. 

- `MAX_CONCURRENT_RECONCILES_<kind>_<group>`: This specifies the maximum number
  of concurrent reconciliations for the operator. It defaults to the number of
  CPUs. You can adjust this based on the cluster resources.

- `WORKER_<kind>_<group>`: **Deprecated**. Use
  `MAX_CONCURRENT_RECONCILES_<kind>_<group>` instead.

- `ANSIBLE_VERBOSITY_<kind>_<group>`: This is used to customize the verbosity
  of the ansible-runner command. The default value is 2.
  The value must be no less than  0 and no greater than 7. 
  This value takes precedence over the global `--ansible-verbosity` flag,
  and it can be overridden by the per-resource annotation named
  `ansible.operator-sdk/verbosity`.

[ansible_env]: https://docs.ansible.com/ansible/latest/reference_appendices/config.html#environment-variables
[runner_input_dir]: https://ansible-runner.readthedocs.io/en/latest/intro.html#runner-input-directory-hierarchy
[watches_doc]: /docs/building-operators/ansible/reference/watches/
