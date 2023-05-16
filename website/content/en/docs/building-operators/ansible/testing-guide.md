---
title: Ansible Based Operator Testing with Molecule
linkTitle: Testing with Molecule
weight: 4
---

## Getting started

### Requirements
To begin, you should have:

- The latest version of the [operator-sdk](https://github.com/operator-framework/operator-sdk) installed.
- Docker installed and running
- [Molecule](https://github.com/ansible/molecule) >= v3.0
- [Ansible](https://github.com/ansible/ansible) >= v2.9
- [The OpenShift Python client](https://github.com/openshift/openshift-restclient-python) >= v0.8
- An initialized Ansible Operator project, with the molecule directory present.

**NOTE**  If you initialized a project with a previous version of operator-sdk, you can generate a new dummy project and copy in the `molecule` directory. Just be sure to generate the dummy project with the same `api-version` and `kind`, or some of the generated files will not work without modification. Your top-level project structure should look like this:

    ```
    .
    ├── config
    ├── Dockerfile
    ├── Makefile
    ├── molecule
    ├── playbooks
    ├── PROJECT
    ├── requirements.yml
    ├── roles
    └── watches.yaml

    ```

- The Ansible content specified in `requirements.yml` will also need to be installed. You can install them with `ansible-galaxy collection install -r requirements.yml`

<!-- TODO(fabianvf, asmacdo): update this section based on the new molecule scaffolding -->
### Molecule scenarios
If you look into the `molecule` directory, you will see four directories (`default`, `test-local`,`cluster`, `templates`). The `default`, `test-local`, and `cluster` directories contain a set of files that together make up what is known as a molecule *scenario*. The `templates` directory contains Jinja templates that are used by multiple scenarios to configure the Kubernetes cluster.

Our molecule scenarios have the following basic structure:

```
.
├── molecule.yml
├── prepare.yml
├── converge.yml
└── verify.yml
```

- `molecule.yml` is a configuration file for molecule. It defines what driver to use to stand up an environment and the associated configuration, linting rules, and a variety of other configuration options. For full documentation on the options available here, see the [molecule configuration documentation](https://molecule.readthedocs.io/configuration/)

- `prepare.yml` is an Ansible playbook that is run once during the set up of a scenario. You can put any arbitrary Ansible in this playbook. It is used for one-time configuration of your test environment, for example, creating the cluster-wide `CustomResourceDefinition` that your Operator will watch.

- `converge.yml` is an Ansible playbook that contains your core logic for the scenario. In a normal molecule scenario, this would import and run the associated role. For Ansible Operators, we mostly use this to create the Kubernetes resources necessary to deploy your operator into Kubernetes.

Below we will walk through the structure and function of each file for each scenario.

#### default
The default scenario is intended for use during the development of your Ansible role or playbook, and will run it outside of the context of an operator. You can run this scenario with
`molecule test` or `molecule converge`. There is no corresponding `operator-sdk` command for this scenario.

The scenario has the following structure:

```
molecule/default
├── molecule.yml
├── prepare.yml
├── converge.yml
└── verify.yml
```

- `molecule.yml` for this scenario tells molecule to use the docker driver to bring up a Kubernetes-in-Docker container,
and by default exposes the API on the host's port 9443. It also specifies a few inventory and environment
variables which are used in `prepare.yml` and `converge.yml`.

- `prepare.yml` ensures that a kubeconfig properly configured to connect to the Kubernetes-in-Docker cluster exists and
is mapped to the proper port, and also waits for the Kubernetes API to become available before allowing testing to begin.

- `converge.yml` imports and runs your role or playbook.

- `verify.yml` is an Ansible playbook where you can put tasks to verify that the state of your cluster matches what you expect.

##### Configuration

There are a few parameters you can tweak at runtime to change the behavior of your molecule run.
You can change these parameters by setting the environment variable before invoking molecule.

The options supported by the default scenario are:

| Environment variable | Default | Purpose |
| :---                 | :---    | :---    |
| KUBE_VERSION | 1.17 | The Kubernetes version to deploy |
| TEST_CLUSTER_PORT | 9443 | The port on the host to expose the Kubernetes API |
| TEST_OPERATOR_NAMESPACE | osdk-test | The namespace to run your role against |

#### cluster

The cluster scenario runs an end-to-end test of your operator against an existing cluster.
The operator image needs to be available to the cluster for this scenario to succeed.
This scenario will deploy your CRDs, RBAC, and operator into the cluster,
and then creates an instance of your CustomResource and runs your assertions to make sure the Operator responded properly.

You can run this scenario with `molecule test` or `molecule converge`. There is no corresponding `operator-sdk` command for this scenario.

The scenario has the following structure:

```
molecule/default
├── molecule.yml
├── create.yml
├── prepare.yml
├── converge.yml
├── verify.yml
└── destroy.yml
```

- `molecule.yml` for this scenario uses the delegated driver, and does not spin up any additional infrastructure.

- `create.yml` is a no-op, but must be present for the delegated driver to work.

- `prepare.yml` ensures the CRD, namespace, and RBAC resources are present in the cluster.

- `converge.yml` creates your operator deployment, based on the template in `molecule/templates/operator.yaml.j2`.

- `verify.yml` is an Ansible playbook where you can put tasks to verify that the state of your cluster matches what you expect. By default, it creates a Custom Resource and waits for reconciliation to complete successfully. There is an example assertion present as well.

- `destroy.yml` ensures that the namespace, RBAC resources, and CRD are deleted at the end of the run.

##### Configuration

There are a few parameters you can tweak at runtime to change the behavior of your molecule run.
You can change these parameters by setting the environment variable before invoking molecule.

The options supported by the default scenario are:

| Environment variable | Default | Purpose |
| :---                 | :---    | :---    |
| OPERATOR_IMAGE | None | *Required* The image to use when deploying the operator into the cluster |
| OPERATOR_PULL_POLICY | Always | The pull policy to use when deploying the operator into the cluster |
| KUBECONFIG | ~/.kube/config | The path to the Kubeconfig for the cluster to test against |
| TEST_OPERATOR_NAMESPACE | osdk-test | The namespace to run your role against |

#### test-local
The test-local scenario runs a full end-to-end test of your operator that does not require an existing
cluster or external registry, and can run in CI environments that allow users to run privileged containers
(such as Travis).
It brings up a Kubernetes-in-docker cluster, builds your Operator, deploys it into the cluster,
and then creates an instance of your CustomResource and runs your assertions to make sure the Operator responded properly.
You can run this scenario with `molecule test -s local`, or with `molecule converge -s test-local` which will leave the environment up afterward.

The scenario has the following structure:

```
molecule/test-local
├── molecule.yml
├── prepare.yml
├── converge.yml
└── verify.yml
```

- `molecule.yml` for this scenario tells molecule to use the docker driver to bring up a Kubernetes-in-Docker container with the project root mounted, and exposes the API on the host's port 10443. It also specifies a few inventory and environment variables which are used in `prepare.yml` and `converge.yml`. It is very similar to the default scenario's configuration.

- `prepare.yml` first runs the `prepare.yml` from the default scenario to ensure the kubeconfig is present and the API is up.
It then runs the `prepare.yml` from the cluster scenario to configure your cluster's CRDs and RBAC.

- `converge.yml` connects to your Kubernetes-in-Docker container, and uses your mounted project root to build your Operator. This makes your Operator available to the cluster without needing to push it to an external registry. Then, it will ensure that a fresh deployment of your Operator is present in the cluster, using the template `molecule/templates/operator.yaml.j2`.

- `verify.yml` will run the `verify.yml` from the `cluster` scenario, as the main difference between the `test-local` and `cluster` scenarios is the method of deployment, but not the behavior of the operator.

##### Configuration

There are a few parameters you can tweak at runtime to change the behavior of your molecule run.
You can change these parameters by setting the environment variable before invoking molecule.

The options supported by the default scenario are:

| Environment variable | Default | Purpose |
| :---                 | :---    | :---    |
| KUBE_VERSION | 1.17 | The Kubernetes version to deploy |
| TEST_CLUSTER_PORT | 10443 | The port on the host to expose the Kubernetes API |
| TEST_OPERATOR_NAMESPACE | osdk-test | The namespace to deploy the operator and associated resources |

#### converge vs test
The two most common molecule commands for testing during development are `molecule test` and `molecule converge`.
`molecule test` performs a full loop, bringing a cluster up, preparing it, running your tasks, and tearing it down.
`molecule converge` is more useful for iterative development, as it leaves your environment up between runs. This
can cause unexpected problems if you end up corrupting your environment during testing, but running `molecule destroy`
will reset it.

- `molecule test` performs a full loop, bringing a cluster up, preparing it, running your tasks, and tearing it down.
- `molecule converge` is more useful for iterative development, as it leaves your environment up between runs. This can cause unexpected problems if you end up corrupting your environment during testing, but running `molecule destroy` will reset it.

## Writing tests

### Adding a task
The default operator that is generated by `operator-sdk new` doesn't do anything, so first we will need to add an
Ansible task so that the Operator does something we can verify. For this example, we will create a simple ConfigMap
with a single key.
We'll be adding the task to `roles/example/tasks/main.yml`, which should now look like this:

```
---
# tasks file for exampleapp
- name: create Example configmap
  kubernetes.core.k8s:
    definition:
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: 'test-data'
        namespace: '{{ ansible_operator_meta.namespace }}'
      data:
        hello: world
```



### Adding a test

Now that our Operator actually does some work, we can add a corresponding assert to `molecule/cluster/verify.yml`.
We'll also add a debug message so that we can see what the ConfigMap looks like.
The file should now look like this:

```
---

- name: Verify
  hosts: localhost
  connection: local
  tasks:
    - debug: var=cm
      vars:
        cm: '{{ lookup("kubernetes.core.k8s", api_version="v1", kind="ConfigMap", namespace=namespace, resource_name="test-data") }}'
    - assert:
        that: cm.data.hello == 'world'
      vars:
        cm: '{{ lookup("kubernetes.core.k8s", api_version="v1", kind="ConfigMap", namespace=namespace, resource_name="test-data") }}'
```

Now that we have a functional Operator, and an assertion of its behavior, we can verify that everything is working
by running `molecule test -s local`.

#### The Ansible `assert` and `fail` modules
These modules are handy for adding assertions and failure conditions to your Ansible Operator tests:

- [assert](https://docs.ansible.com/ansible/2.9/modules/assert_module.html)
- [fail](https://docs.ansible.com/ansible/2.9/modules/fail_module.html)
