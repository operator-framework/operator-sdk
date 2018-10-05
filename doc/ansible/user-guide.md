# User Guide

This guide walks through an example of building a simple memcached-operator
powered by Ansible using tools and libraries provided by the Operator SDK.

## Prerequisites

- [git][git_tool]
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.9.0+.
- [ansible][ansible_tool] version v2.6.0+
- [ansible-runner][ansible_runner_tool] version v1.1.0+
- [ansible-runner-http][ansible_runner_http_plugin] version v1.0.0+
- [dep][dep_tool] version v0.5.0+. (Optional if you aren't installing from source)
- [go][go_tool] version v1.10+. (Optional if you aren't installing from source)
- Access to a kubernetes v.1.9.0+ cluster.

**Note**: This guide uses [minikube][minikube_tool] version v0.25.0+ as the
local kubernetes cluster and quay.io for the public registry.

## Install the Operator SDK CLI

The Operator SDK has a CLI tool that helps the developer to create, build, and
deploy a new operator project.

Checkout the desired release tag and install the SDK CLI tool:

```sh
$ mkdir -p $GOPATH/src/github.com/operator-framework
$ cd $GOPATH/src/github.com/operator-framework
$ git clone https://github.com/operator-framework/operator-sdk
$ cd operator-sdk
$ git checkout master
$ make dep
$ make install
```

This installs the CLI binary `operator-sdk` at `$GOPATH/bin`.

## Create a new project

Use the CLI to create a new Ansible-based memcached-operator project:

```sh
$ mkdir -p $GOPATH/src/github.com/example-inc/
$ cd $GOPATH/src/github.com/example-inc/
$ operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached --type=ansible
$ cd memcached-operator
```

This creates the memcached-operator project specifically for watching the
Memcached resource with APIVersion `cache.example.com/v1apha1` and Kind
`Memcached`.

To learn more about the project directory structure, see [project
layout][layout_doc] doc.

## Customize the operator logic

For this example the memcached-operator will execute the following
reconciliation logic for each `Memcached` Custom Resource (CR):
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached`
CR

### Watch the Memcached CR

By default, the memcached-operator watches `Memcached` resource events as shown
in `watches.yaml` and executes Ansible Role `Memached`:

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
  role: /opt/ansible/roles/Memcached
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
  playbook: /opt/ansible/playbook.yaml
```

## Building the Memcached Ansible Role

The first thing to do is to modify the generated Ansible role under
`roles/Memcached`. This Ansible Role controls the logic that is executed when a
resource is modified.

### Define the Memcached spec

Defining the spec for an Ansible Operator can be done entirely in Ansible. The
Ansible Operator will simply pass all key value pairs listed in the Custom
Resource spec field along to Ansible as
[variables](https://docs.ansible.com/ansible/2.5/user_guide/playbooks_variables.html#passing-variables-on-the-command-line).
It is recommended that you perform some type validation in Ansible on the
variables to ensure that your application is receiving expected input.

First, set a default in case the user doesn't set the `spec` field by modifying
`roles/Memcached/defaults/main.yml`:
```yaml
size: 1
```

### Defining the Memcached deployment

Now that we have the spec defined, we can define what Ansible is actually
executed on resource changes. Since this is an Ansible Role, the default
behavior will be to execute the tasks in `roles/Memcached/tasks/main.yml`. We
want Ansible to create a deployment if it does not exist which runs the
`memcached:1.4.36-alpine` image. Ansible 2.5+ supports the [k8s Ansible
Module](https://docs.ansible.com/ansible/2.6/modules/k8s_module.html) which we
will leverage to control the deployment definition.

Modify `roles/Memcached/tasks/main.yml` to look like the following:
```yaml
---
- name: start memcached
  k8s:
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
$ kubectl create -f deploy/crd.yaml
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

Deploy the memcached-operator:

```sh
$ kubectl create -f deploy/rbac.yaml
$ kubectl create -f deploy/operator.yaml
```

**NOTE**: `deploy/rbac.yaml` creates a `ClusterRoleBinding` and assumes we are
working in namespace `default`. If you are working in a different namespace you
must modify this file before creating it.

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

#### 2. Run outside the cluster

This method is preferred during the development cycle to speed up deployment and testing.

**Note**: Ensure that [Ansible Runner][ansible_runner_tool] and [Ansible Runner
HTTP Plugin][ansible_runner_http_plugin] is installed or else you will see
unexpected errors from Ansible Runner when a Custom Resource is created.

It is also important that the `role` path referenced in `watches.yaml` exists
on your machine. Since we are normally used to using a container where the Role
is put on disk for us, we need to manually copy our role to the configured
Ansible Roles path (e.g `/etc/ansible/roles`.

Run the operator locally with the default kubernetes config file present at
`$HOME/.kube/config`:

```sh
$ operator-sdk up local
INFO[0000] Go Version: go1.10
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git
```

Run the operator locally with a provided kubernetes config file:

```sh
$ operator-sdk up local --kubeconfig=config
INFO[0000] Go Version: go1.10
INFO[0000] Go OS/Arch: darwin/amd64
INFO[0000] operator-sdk Version: 0.0.5+git
```

### Create a Memcached CR

Modify `deploy/cr.yaml` as shown and create a `Memcached` custom resource:

```sh
$ cat deploy/cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 3

$ kubectl apply -f deploy/cr.yaml
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
memcached-operator-7cc7cfdf86-vvjqk   1/1       Running   0          2m
```

### Update the size

Change the `spec.size` field in the memcached CR from 3 to 4 and apply the
change:

```sh
$ cat deploy/cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 4

$ kubectl apply -f deploy/cr.yaml
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
$ kubectl delete -f deploy/cr.yaml
$ kubectl delete -f deploy/operator.yaml
```

[layout_doc]:./project_layout.md
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
[ansible_tool]:https://docs.ansible.com/ansible/latest/index.html
[ansible_runner_tool]:https://ansible-runner.readthedocs.io/en/latest/install.html
[ansible_runner_http_plugin]:https://github.com/ansible/ansible-runner-http
