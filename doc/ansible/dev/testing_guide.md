# Testing Ansible Operators with Molecule

## Getting started

### Requirements
To begin, you sould have:
- The latest version of the [operator-sdk](https://github.com/operator-framework/operator-sdk) installed.
- Docker installed and running
- [Molecule](https://github.com/ansible/molecule) >= v2.20 (currently that will require installation from source, `pip install git+https://github.com/ansible/molecule.git`)
- [Ansible](https://github.com/ansible/ansible) >= v2.7
- [jmespath](https://pypi.org/project/jmespath/)
- [The OpenShift Python client](https://github.com/openshift/openshift-restclient-python) >= v0.8
- An initialized Ansible Operator project, with the molecule directory present. If you initialized a project with a previous 
  version of operator-sdk, you can generate a new dummy project and copy in the `molecule` directory. Just be sure
  to generate the dummy project with the same `api-version` and `kind`, or some of the generated files will not work
  without modification. Your top-level project structure should look like this:
    ```
    .
    ├── build
    ├── deploy
    ├── molecule
    ├── roles
    ├── playbook.yml (optional)
    └── watches.yaml
    ```

### Molecule scenarios
If you look into the `molecule` directory, you will see three directories (`default`, `test-local`, `test-cluster`).
Each of those directories contains a set of files that together make up what is known as a molecule *scenario*.

Our molecule scenarios have the following basic structure:

```
.
├── molecule.yml
├── prepare.yml
└── playbook.yml
```

`molecule.yml` is a configuration file for molecule. It defines what driver to use to stand up an environment and the associated configuration, linting rules, and a variety of other configuration options. For full documentation on the options available here, see the [molecule configuration documentation](https://molecule.readthedocs.io/en/latest/configuration.html)

`prepare.yml` is an Ansible playbook that is run once during the set up of a scenario. You
can put any arbitrary Ansible in this playbook. It is used for one-time configuration
of your test environment, for example, creating the cluster-wide `CustomResourceDefinition`
that your Operator will watch.

`playbook.yml` is an Ansible playbook that contains your core logic for the scenario. In a
normal molecule scenario, this would import and run the associated role. For Ansible
Operator, we mostly use this to create the Kubernetes resources and then execute a
series of asserts that verify your cluster state.

Below we will walk through the structure and function of each file for each scenario.

#### default
The default scenario is intended for use during the development of your Ansible role or playbook, and will run it
outside of the context of an operator.
You can run this scenario with 
`molecule test`
or 
`molecule converge`. There is no corresponding `operator-sdk` command for this scenario.

The scenario has the following structure:

```
molecule/default
├── asserts.yml
├── molecule.yml
├── playbook.yml
└── prepare.yml
```

`asserts.yml` is an Ansible playbook contains Ansible assert tasks that will be run by all three scenarios. 
If you would like to write specific asserts for individual scenarios, you can instead remove the `asserts.yml`
playbook import from that scenario's `playbook.yml`, or if you only want to add additional asserts, you can
create a new playbook in that scenario and import it at the bottom of that scenario's `playbook.yml`.

`molecule.yml` for this scenario tells molecule to use the docker driver to bring up a Kubernetes-in-Docker container, 
and exposes the API on the host's port 9443. It also specifies a few inventory and environment
variables which are used in `prepare.yml` and `playbook.yml`.

`prepare.yml` ensures that a kubeconfig properly configured to connect to the Kubernetes-in-Docker cluster exists and is mapped to the proper port, and also waits for the Kubernetes API to become
available before allowing testing to begin.

`playbook.yml` only imports your role or playbook and then imports the `asserts.yml` playbook.

#### test-local
The test-local scenario is a more full integration test of your operator. It brings up a Kubernetes-in-docker cluster, builds your Operator, deploys it
into the cluster, and then creates an instance of your CustomResource and runs your assertions to make sure the Operator responded properly. You can run
this scenario with 
`molecule test -s local`, which is equivalent to `operator-sdk test local`, or with `molecule converge -s test-local`, which will leave the environment up
afterward.

The scenario has the following structure:

```
molecule/test-local
├── molecule.yml
├── playbook.yml
└── prepare.yml
```

`molecule.yml` for this scenario tells molecule to use the docker driver to bring up a Kubernetes-in-Docker container with the project root mounted,
and exposes the API on the host's port 10443. It also specifies a few inventory and environment
variables which are used in `prepare.yml` and `playbook.yml`. It is very similar to the default scenario's configuration.

`prepare.yml` first runs the `prepare.yml` from the default scenario to ensure the kubeconfig is present and the API is up. It then creates the CustomResourceDefinition, namespace, and RBAC
resources specified in the `deploy/` directory.

`playbook.yml` is the most complicated file in this project. First, it connects to your
Kubernetes-in-Docker container, and uses your mounted project root to build your Operator.
This makes your Operator available to the cluster without needing to push it to an external
registry. Next, it will ensure that a fresh deployment of your Operator is present in the
cluster, and once there is it will create an instance of your Custom Resource 
(specified in `deploy/crds/`). It will then wait for the CustomResource to report a successful
run, and once it has, will import the `asserts.yml` from the default scenario.

#### test-cluster
The test-cluster scenario is intended as a full integration test against
an existing Kubernetes cluster, and assumes that the cluster is already available, the dependent resources from the `deploy/` directory
are created, the operator image is built with `--enable-tests`, and that the image is available in a container registry. It connects
to the existing Kubernetes cluster and deploys the test Operator, creates a Custom Resource, and runs your asserts.  You shouldn't
call this scenario directly, rather you should build your operator with the `--enable-tests` flag, in which case a new entrypoint will 
be added that runs this scenario when the container starts up. It is recommended that you only interact with this scenario through
`operator-sdk test cluster`.

The scenario has the following structure:

```
molecule/test-cluster
├── molecule.yml
└── playbook.yml
```
`molecule.yml` for this scenario is very simple, as it assumes an environment is already
present. It essentially is just specifying the metadata of the scenario, and telling molecule 
not to try and create or destroy anything when run.

`playbook.yml` is also pretty simple, compared to the previous scenarios. All it does is create
an instance of your Custom Resource (specified in `deploy/crds`), and then import the `asserts.yml` from the `default` scenario.

#### converge vs test
The two most common molecule commands for testing during development are `molecule test` and `molecule converge`. 
`molecule test` performs a full loop, bringing a cluster up, preparing it, running your tasks, and tearing it down.
`molecule converge` is more useful for iterative development, as it leaves your environment up between runs. This
can cause unexpected problems if you end up corrupting your environment during testing, but running `molecule destroy`
will reset it.



## operator-sdk test commands

### test local

The `operator-sdk test local` command kicks off an end-to-end test of your Operator. It will bring up a [Kubernetes-in-Docker (kind)](https://github.com/bsycorp/kind) cluster, builds your Operator
image and make it available to that cluster, create all the required resources from the `deploy/` directory, create an instance of your
Custom Resource (specified in the `deploy/crds` directory), and then verify that the Operator has responded appropriately by running
the asserts from `molecule/default/asserts.yml`.


### test cluster

The `operator-sdk test cluster` command does much less than the `test local` command. It is intended as a full integration test against
an existing Kubernetes cluster, and assumes that the cluster is already available, the dependent resources from the `deploy/` directory
are created, the operator image is built with `--enable-tests`, and that the image is available in a container registry. When you run the command, it will connect
to the existing Kubernetes cluster and deploy the test Operator, create a Custom Resource, and run the asserts in `molecule/default/asserts.yml`.

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
  k8s:
    definition:
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: 'test-data'
        namespace: '{{ meta.namespace }}'
      data:
        hello: world
```



### Adding a test

Now that our Operator actually does some work, we can add a corresponding assert to `molecule/default/asserts.yml`. 
We'll also add a debug message so that we can see what the ConfigMap looks like.
The file should now look like this:

```
---

- name: Verify
  hosts: localhost
  connection: local
  vars:
    ansible_python_interpreter: '{{ ansible_playbook_python }}'
  tasks:
    - debug: var=cm
      vars:
        cm: '{{ lookup("k8s", api_version="v1", kind="ConfigMap", namespace=namespace, resource_name="test-data") }}'
    - assert:
        that: cm.data.hello == 'world'
      vars:
        cm: '{{ lookup("k8s", api_version="v1", kind="ConfigMap", namespace=namespace, resource_name="test-data") }}'
```

Now that we have a functional Operator, and an assertion of its behavior, we can verify that everything is working
by running `operator-sdk test local`.

#### The Ansible `assert` and `fail` modules
These modules are handy for adding assertions and failure conditions to your Ansible Operator tests:

- [assert](https://docs.ansible.com/ansible/latest/modules/assert_module.html)
- [fail](https://docs.ansible.com/ansible/latest/modules/fail_module.html)
