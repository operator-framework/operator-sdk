# Ansible User Guide for Operator SDK

This guide walks through an example of building a simple memcached-operator powered by Ansible using tools and libraries provided by the Operator SDK.

## Prerequisites

- [git][git-tool]
- [docker][docker-tool] version 17.03+.
- [kubectl][kubectl-tool] version v1.9.0+.
- [ansible][ansible-tool] version v2.6.0+
- [ansible-runner][ansible-runner-tool] version v1.1.0+
- [ansible-runner-http][ansible-runner-http-plugin] version v1.0.0+
- [go][go-tool] version v1.13+. (Optional if you aren't installing from source)
- Access to a Kubernetes v.1.9.0+ cluster.

**Note**: This guide uses [minikube][minikube-tool] version v0.25.0+ as the
local Kubernetes cluster and [quay.io][quay-link] for the public registry.

## Install the Operator SDK CLI

Follow the steps in the [installation guide][install-guide] to learn how to install the Operator SDK CLI tool.

## Create a new project

Use the CLI to create a new Ansible-based memcached-operator project:

```sh
$ operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached --type=ansible
$ cd memcached-operator
```

This creates the memcached-operator project specifically for watching the
Memcached resource with APIVersion `cache.example.com/v1apha1` and Kind
`Memcached`.

To learn more about the project directory structure, see [project
layout][layout-doc] doc.

#### Operator scope

Read the [operator scope][operator-scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.

### Watches file

The Watches file contains a list of mappings from custom resources, identified
by it's Group, Version, and Kind, to an Ansible Role or Playbook. The Operator
expects this mapping file in a predefined location: `/opt/ansible/watches.yaml`
These resources, as well as child resources (determined by owner references) will
be monitored for updates and cached.

* **group**:  The group of the Custom Resource that you will be watching.
* **version**:  The version of the Custom Resource that you will be watching.
* **kind**:  The kind of the Custom Resource that you will be watching.
* **role** (default): Specifies a role to be executed. This field is mutually exclusive with the
  "playbook" field. This field can be:
  * an absolute path to a role directory.
  * a relative path within one of the directories specified by `ANSIBLE_ROLES_PATH` environment variable or `ansible-roles-path` flag.
  * a relative path within the current working directory, which defaults to `/opt/ansible/roles`.
  * a fully qualified collection name of an installed Ansible collection. Ansible collections are installed to
    `~/.ansible/collections` or `/usr/share/ansible/collections` by default. If they are installed elsewhere,
    use the `ANSIBLE_COLLECTIONS_PATH` environment variable or the `ansible-collections-path` flag
* **playbook**: This is the playbook name that you have added to the
  container. This playbook is expected to be simply a way to call roles. This
  field is mutually exclusive with the "role" field. When running locally, the playbook is expected to be in the
  current project directory.
* **vars**: This is an arbitrary map of key-value pairs. The contents will be
  passed as `extra_vars` to the playbook or role specified for this watch.
* **reconcilePeriod** (optional): The reconciliation interval, how often the
  role/playbook is run, for a given CR.
* **manageStatus** (optional): When true (default), the operator will manage
  the status of the CR generically. Set to false, the status of the CR is
  managed elsewhere, by the specified role/playbook or in a separate controller.
* **blacklist**: A list of child resources (by GVK) that will not be watched or cached.

An example Watches file:

```yaml
---
# Simple example mapping Foo to the Foo role
- version: v1alpha1
  group: foo.example.com
  kind: Foo
  role: Foo

# Simple example mapping Bar to a playbook
- version: v1alpha1
  group: bar.example.com
  kind: Bar
  playbook: playbook.yml

# More complex example for our Baz kind
# Here we will disable requeuing and be managing the CR status in the playbook,
# and specify additional variables.
- version: v1alpha1
  group: baz.example.com
  kind: Baz
  playbook: baz.yml
  reconcilePeriod: 0
  manageStatus: False
  vars:
    foo: bar

# ConfigMaps owned by a Memcached CR will not be watched or cached.
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: /opt/ansible/roles/memcached
  blacklist:
    - group: ""
      version: v1
      kind: ConfigMap

# Example usage with a role from an installed Ansible collection
- version: v1alpha1
  group: bar.example.com
  kind: Bar
  role: myNamespace.myCollection.myRole
```

## Customize the operator logic

For this example the memcached-operator will execute the following
reconciliation logic for each `Memcached` Custom Resource (CR):
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached`
CR

### Watch the Memcached CR

By default, the memcached-operator watches `Memcached` resource events as shown
in `watches.yaml` and executes Ansible Role `Memcached`:

```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
```

#### Options
**Role**
Specifying a `role` option in `watches.yaml` will configure the operator to use
this specified path when launching `ansible-runner` with an Ansible Role. By
default, the `new` command will fill in an absolute path to where your role
should go.
```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: memcached
```

**Playbook**
Specifying a `playbook` option in `watches.yaml` will configure the operator to
use this specified path when launching `ansible-runner` with an Ansible
Playbook
```yaml
---
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  playbook: playbook.yaml
```

## Building the Memcached Ansible Role

The first thing to do is to modify the generated Ansible role under
`roles/memcached`. This Ansible Role controls the logic that is executed when a
resource is modified.

### Define the Memcached spec

Defining the spec for an Ansible Operator can be done entirely in Ansible. The
Ansible Operator will simply pass all key value pairs listed in the Custom
Resource spec field along to Ansible as extra
[variables](https://docs.ansible.com/ansible/2.5/user_guide/playbooks_variables.html#passing-variables-on-the-command-line).
The names of all variables in the spec field are converted to snake_case
by the operator before running ansible. For example, `serviceAccount` in
the spec becomes `service_account` in ansible.
It is recommended that you perform some type validation in Ansible on the
variables to ensure that your application is receiving expected input.

First, set a default in case the user doesn't set the `spec` field by modifying
`roles/memcached/defaults/main.yml`:
```yaml
size: 1
```

### Defining the Memcached deployment

Now that we have the spec defined, we can define what Ansible is actually
executed on resource changes. Since this is an Ansible Role, the default
behavior will be to execute the tasks in `roles/memcached/tasks/main.yml`. We
want Ansible to create a deployment if it does not exist which runs the
`memcached:1.4.36-alpine` image. Ansible 2.5+ supports the [k8s Ansible
Module](https://docs.ansible.com/ansible/2.6/modules/k8s_module.html) which we
will leverage to control the deployment definition.

Modify `roles/memcached/tasks/main.yml` to look like the following:
```yaml
---
- name: start memcached
  community.kubernetes.k8s:
    definition:
      kind: Deployment
      apiVersion: apps/v1
      metadata:
        name: '{{ meta.name }}-memcached'
        namespace: '{{ meta.namespace }}'
      spec:
        replicas: "{{size}}"
        selector:
          matchLabels:
            app: memcached
        template:
          metadata:
            labels:
              app: memcached
          spec:
            containers:
            - name: memcached
              command:
              - memcached
              - -m=64
              - -o
              - modern
              - -v
              image: "docker.io/memcached:1.4.36-alpine"
              ports:
                - containerPort: 11211

```

It is important to note that we used the `size` variable to control how many
replicas of the Memcached deployment we want. We set the default to `1`, but
any user can create a Custom Resource that overwrites the default.

### Build and run the operator

Before running the operator, Kubernetes needs to know about the new custom
resource definition the operator will be watching.

Deploy the CRD:

```sh
$ kubectl create -f deploy/crds/cache.example.com_memcacheds_crd.yaml
```

Once this is done, there are two ways to run the operator:

- As a pod inside a Kubernetes cluster
- As a go program outside the cluster using `operator-sdk`

#### 1. Run as a pod inside a Kubernetes cluster

Running as a pod inside a Kubernetes cluster is preferred for production use.

Build the memcached-operator image and push it to a registry:
```
$ operator-sdk build quay.io/example/memcached-operator:v0.0.1
$ docker push quay.io/example/memcached-operator:v0.0.1
```

Kubernetes deployment manifests are generated in `deploy/operator.yaml`. The
deployment image in this file needs to be modified from the placeholder
`REPLACE_IMAGE` to the previous built image. To do this run:
```
$ sed -i 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
```

**Note**
If you are performing these steps on OSX, use the following `sed` commands instead:
```
$ sed -i "" 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
```

Deploy the memcached-operator:

```sh
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
$ kubectl create -f deploy/operator.yaml
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

#### 2. Run outside the cluster

This method is preferred during the development cycle to speed up deployment and testing.

**Note**: Ensure that [Ansible Runner][ansible-runner-tool] and [Ansible Runner
HTTP Plugin][ansible-runner-http-plugin] is installed or else you will see
unexpected errors from Ansible Runner when a Custom Resource is created.

It is also important that the `role` path referenced in `watches.yaml` exists
on your machine. Since we are normally used to using a container where the Role
is put on disk for us, we need to manually copy our role to the configured
Ansible Roles path (e.g `/etc/ansible/roles`.

Run the operator locally with the default Kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ operator-sdk run --local
INFO[0000] Go Version: go1.10
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git
```

Run the operator locally with a provided Kubernetes config file:

```sh
$ operator-sdk run --local --kubeconfig=config
INFO[0000] Go Version: go1.10
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git
```

### Create a Memcached CR

Modify `deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml` as shown and create a `Memcached` custom resource:

```sh
$ cat deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 3

$ kubectl apply -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
```

Ensure that the memcached-operator creates the deployment for the CR:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           2m
example-memcached        3         3         3            3           1m
```

Check the pods to confirm 3 replicas were created:

```sh
$ kubectl get pods
NAME                                  READY     STATUS    RESTARTS   AGE
example-memcached-6fd7c98d8-7dqdr     1/1       Running   0          1m
example-memcached-6fd7c98d8-g5k7v     1/1       Running   0          1m
example-memcached-6fd7c98d8-m7vn7     1/1       Running   0          1m
memcached-operator-7cc7cfdf86-vvjqk   2/2       Running   0          2m
```

### View the Ansible logs

In order to see the logs from a particular you can run:

```sh
kubectl logs deployment/memcached-operator
```

The logs contain the information about the Ansible run and will make it much easier to debug issues within your Ansible tasks.
Note that the logs will contain much more detailed information about the Ansible Operator's internals and interface with Kubernetes as well.

Also, you can use the environment variable `ANSIBLE_DEBUG_LOGS` set as `True` to check the full Ansible result in the logs in order to be able to debug it.

**Example**

In the `deploy/operator.yaml`:
```yaml
...
- name: ANSIBLE_DEBUG_LOGS
  value: "True"
...
```

### Additional Ansible Debug

Occasionally while developing additional debug in the Operator logs is nice to have.
Using the memcached operator as an example, we can simply add the
`"ansible.operator-sdk/verbosity"` annotation to the Custom
Resource with the desired verbosity.

```yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
  annotations:
    "ansible.operator-sdk/verbosity": "4"
spec:
  size: 4
```

### Update the size

Change the `spec.size` field in the memcached CR from 3 to 4 and apply the
change:

```sh
$ cat deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 4

$ kubectl apply -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-memcached    4         4         4            4           5m
```

### Cleanup

Clean up the resources:

```sh
$ kubectl delete -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/cache.example.com_memcacheds_crd.yaml
```

**NOTE** Additional CR/CRD's can be added to the project by running, for example, the command :`operator-sdk new api --api-version=cache.example.com/v1alpha1 --kind=AppService --type=ansible`

[operator-scope]:./../operator-scope.md
[install-guide]: ../user/install-operator-sdk.md
[layout-doc]:./project_layout.md
[homebrew-tool]:https://brew.sh/
[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube-tool]:https://github.com/kubernetes/minikube#installation
[ansible-tool]:https://docs.ansible.com/ansible/latest/index.html
[ansible-runner-tool]:https://ansible-runner.readthedocs.io/en/latest/install.html
[ansible-runner-http-plugin]:https://github.com/ansible/ansible-runner-http
[quay-link]:https://quay.io
