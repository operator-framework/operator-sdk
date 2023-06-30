---
title: Development Tips
weight: 5
---

This document provides some useful information and tips for a developer
creating an operator powered by Ansible.

## Getting started with the Kubernetes Collection for Ansible

Since we are interested in using Ansible for the lifecycle management of our
application on Kubernetes, it is beneficial for a developer to get a good
grasp of the [Kubernetes Collection for Ansible][kubernetes_collection].
This Ansible collection allows a developer to either leverage their existing
Kubernetes resource files (written in YAML) or express the lifecycle
management in native Ansible. One of the biggest benefits of using Ansible in
conjunction with existing Kubernetes resource files is the ability to use
Jinja templating so that you can customize deployments with the simplicity of
a few variables in Ansible.

The easiest way to get started is to install the collection on your local
machine and test it using a playbook.

### Installing the Kubernetes Collection for Ansible

To install the Kubernetes Collection, one must first install Ansible 2.9+.
For example, on Fedora/Centos:

```sh
sudo dnf install ansible
```

In addition to Ansible, a user must install the
[Python Kubernetes Client][python-kubernetes-client] package:

```sh
pip3 install kubernetes
```

Finally, install the Kubernetes Collection from ansible-galaxy:

```sh
ansible-galaxy collection install kubernetes.core
```

Alternatively, if you've already initialized your operator, you may have a
`requirements.yml` file at the top level of your project. This file specifies
Ansible dependencies that need to be installed for your operator to function.
By default it will install the `kubernetes.core` collection as well as
the `operator_sdk.util` collection, which provides modules and plugins for
operator-specific operations.

To install the dependent modules from this file, run:

```sh
ansible-galaxy collection install -r requirements.yml
```

### Testing the Kubernetes Collection locally

Sometimes it is beneficial for a developer to run the Ansible code from their
local machine as opposed to running/rebuilding the operator each time. To do
this, initialize a new project:

```sh
mkdir memcached-operator && cd memcached-operator
operator-sdk init --plugins=ansible --domain=example.com --group=cache --version=v1alpha1 --kind=Memcached --generate-role
ansible-galaxy collection install -r requirements.yml
```

Modify `roles/memcached/tasks/main.yml` with desired Ansible logic. For this example
we will create and delete a ConfigMap based on the value of a variable named
`state`:

```yaml
---
- name: set ConfigMap example-config to {{ state }}
  kubernetes.core.k8s:
    api_version: v1
    kind: ConfigMap
    name: example-config
    namespace: default
    state: "{{ state }}"
  ignore_errors: true
```

{{% alert title="Note" color="primary" %}}
Setting `ignore_errors: true` is done so that deleting a nonexistent
ConfigMap doesn't error out.
{{% /alert %}}

Modify `roles/memcached/defaults/main.yml` to set `state` to `present` as default.

```yaml
---
state: present
```

Create an Ansible playbook `playbook.yml` in the top-level directory which
includes role `memcached`:

```yaml
---
- hosts: localhost
  roles:
    - memcached
```

Run the playbook:

```console
$ ansible-playbook playbook.yml
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [memcached : set ConfigMap example-config to present]
changed: [localhost]

PLAY RECAP *********************************************************************************
localhost                  : ok=2    changed=1    unreachable=0    failed=0
```

Check that the ConfigMap was created:

```console
$ kubectl get configmaps
NAME                    STATUS    AGE
example-config          Active    3s
```

Rerun the playbook setting `state` to `absent`:

```console
$ ansible-playbook playbook.yml --extra-vars state=absent
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [memcached : set ConfigMap example-config to absent]
changed: [localhost]

PLAY RECAP *********************************************************************************
localhost                  : ok=2    changed=1    unreachable=0    failed=0
```

Check that the ConfigMap was deleted:

```console
$ kubectl get configmaps
No resources found in default namespace.
```

## Using Ansible inside an Operator

Now that we have demonstrated using the Kubernetes Collection, we want to
trigger this Ansible logic when a custom resource changes. In the above
example, we want to map a role to a specific Kubernetes resource that the
operator will watch. This mapping is done in a file called `watches.yaml`.

### Custom Resource file

The Custom Resource (CR) file format is Kubernetes resource file. The object
has some mandatory fields:

- `apiVersion`:  The version of the Custom Resource that will be created.
- `kind`:  The kind of the Custom Resource that will be created
- `metadata`:  Kubernetes specific metadata to be created
- `spec`:  This is the key-value list of variables which are passed to Ansible.
  This field is optional and empty by default.
- `annotations`: Kubernetes specific annotations to be appended to the CR. See
  the below section for Ansible Operator specific annotations. This field is optional.

#### Annotations for Custom Resource

This is the list of CR annotations which will modify the behavior of the operator:

- `ansible.operator-sdk/reconcile-period`: Specifies the maximum time before a
  reconciliation is triggered. Note that at scale, this can reduce
  performance, see [watches][watches] reference for more information. This value
  is parsed using the standard Go package [time][time_pkg]. Specifically
  [ParseDuration][time_parse_duration] is used which will apply the default
  suffix of `s` giving the value in seconds.

  Example:

  ```yaml
  apiVersion: cache.example.com/v1alpha1
  kind: Memcached
  metadata:
    name: example
    annotations:
      ansible.operator-sdk/reconcile-period: "30s"
  ```

Note that a lower period will correct entropy more quickly, but reduce
responsiveness to change if there are many watched resources. Typically, this
option should only be used in advanced use cases where
`watchDependentResources` is set to `False`  and when is not possible to use
the watch feature. E.g To managing external resources that don’t raise
Kubernetes events.

### Testing an Ansible Operator locally

Once a developer is comfortable working with the above workflow, it will be
beneficial to test the logic inside an operator.

**Prerequisites**:
- Read the [Ansible Operator tutorial][tutorial].
- Install `ansible-operator` [dependencies][py-deps] using [`pipenv`][pipenv]
and their OS prerequisite [packages][os-pkgs] (these will differ depending on OS) locally.

The `run` Makefile target runs the `ansible-operator` binary locally, which reads from
`./watches.yaml` and uses `~/.kube/config` to communicate with a Kubernetes
cluster just as the `k8s` modules do. The `install` target registers the operator's
`Memcached` CustomResourceDefinition (CRD) with the apiserver.

{{% alert title="Note" color="primary" %}}
You can customize the roles path by setting the environment variable
`ANSIBLE_ROLES_PATH` or using the flag `ansible-roles-path`. Note that if the
role is not found in `ANSIBLE_ROLES_PATH`, then the operator will look for it
in `{{current directory}}/roles`.   
{{% /alert %}}

```console
$ make install run
/home/user/memcached-operator/bin/kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/memcacheds.cache.example.com created
/home/user/go/bin/ansible-operator run
{"level":"info","ts":1595899073.9861593,"logger":"cmd","msg":"Version","Go Version":"go1.13.12","GOOS":"linux","GOARCH":"amd64","ansible-operator":"v0.19.0+git"}
{"level":"info","ts":1595899073.987384,"logger":"cmd","msg":"WATCH_NAMESPACE environment variable not set. Watching all namespaces.","Namespace":""}
{"level":"info","ts":1595899074.9504397,"logger":"controller-runtime.metrics","msg":"metrics server is starting to listen","addr":":8080"}
{"level":"info","ts":1595899074.9522583,"logger":"watches","msg":"Environment variable not set; using default value","envVar":"ANSIBLE_VERBOSITY_MEMCACHED_CACHE_EXAMPLE_COM","default":2}
{"level":"info","ts":1595899074.9524004,"logger":"cmd","msg":"Environment variable not set; using default value","Namespace":"","envVar":"ANSIBLE_DEBUG_LOGS","ANSIBLE_DEBUG_LOGS":false}
{"level":"info","ts":1595899074.9524298,"logger":"ansible-controller","msg":"Watching resource","Options.Group":"cache.example.com","Options.Version":"v1","Options.Kind":"Memcached"}
```

Now that the operator is watching resource `Memcached` for events, the creation of a
Custom Resource will trigger our Ansible Role to be executed. Take a look at
`config/samples/cache_v1alpha1_memcached.yaml`:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: "memcached-sample"
```

Since `spec` is not set, Ansible is invoked with no extra variables. The next
section covers how extra variables are passed from a Custom Resource to
Ansible. This is why it is important to set sane defaults for the operator.

Create a Custom Resource instance of Memcached with variable `state` default to
`present`:

```sh
kubectl create -f config/samples/cache_v1alpha1_memcached.yaml
```

Check that ConfigMap `example-config` was created:

```console
$ kubectl get configmaps
NAME                    STATUS    AGE
example-config          Active    3s
```

Modify `config/samples/cache_v1alpha1_memcached.yaml` to set `state` to
`absent`:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: memcached-sample
spec:
  state: absent
```

Apply the changes to Kubernetes and confirm that the ConfiMap is deleted:

```sh
kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
kubectl get configmaps
```

### Testing an Ansible Operator on a cluster

Now that a developer is confident in the operator logic, testing the operator
inside of a pod on a Kubernetes cluster is desired. Running as a pod inside a
Kubernetes cluster is preferred for production use.

To build the `memcached-operator` image and push it to a registry:

```sh
make docker-build docker-push IMG=example.com/memcached-operator:v0.0.1
```

Deploy the memcached-operator:

```sh
make install
make deploy IMG=example.com/memcached-operator:v0.0.1
```

Verify that the memcached-operator is up and running:

```console
$ kubectl get deployment -n memcached-operator-system
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

### Viewing the Ansible logs

In order to see the logs from a particular operator you can run:

```sh
kubectl logs deployment/memcached-operator-controller-manager -n memcached-operator-system
```

The logs contain the information about the Ansible run and are useful for
debugging your Ansible tasks. Note that the logs may contain much more
detailed information about the Ansible Operator's internals and its
interactions with Kubernetes as well.

Also, you can set the environment variable `ANSIBLE_DEBUG_LOGS` to `True` to
check the full Ansible result in the logs in order to be able to debug it.

**Example**

In `config/manager/manager.yaml` and `config/default/manager_auth_proxy_patch.yaml`:

```yaml
...
      containers:
      - name: manager
        env:
        - name: ANSIBLE_DEBUG_LOGS
          value: "True"
...
```

Occasionally while developing additional debug in the Operator logs is nice to have.
Using the memcached operator as an example, we can simply add the
`"ansible.sdk.operatorframework.io/verbosity"` annotation to the Custom
Resource with the desired verbosity.

```yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
  annotations:
    "ansible.sdk.operatorframework.io/verbosity": "4"
spec:
  size: 4
```

## Custom Resource Status Management

By default, an Ansible Operator will include the generic output from previous
Ansible run as the `status` subresource of a CR. This includes the number of
successful and failed tasks and relevant error messages as shown below:

```yaml
status:
  conditions:
  - ansibleResult:
      changed: 3
      completion: 2018-12-03T13:45:57.13329
      failures: 1
      ok: 6
      skipped: 0
    lastTransitionTime: 2018-12-03T13:45:57Z
    message: 'Status code was -1 and not [200]: Request failed: <urlopen error [Errno
      113] No route to host>'
    reason: Failed
    status: "True"
    type: Failure
  - lastTransitionTime: 2018-12-03T13:46:13Z
    message: Running reconciliation
    reason: Running
    status: "True"
    type: Running
```

An Ansible Operator also allows you to supply custom status values with the
`k8s_status` Ansible module, which is included in
[operator_sdk.util][operator_sdk_util] collection.
You can update the `status` from within Ansible with any key/value pairs as
desired. If you do not want the operator to update the status with Ansible
output, and you want to track the CR status manually from your application,
you can update the `watches.yaml` file with `manageStatus`, as shown below:

```yaml
- version: v1
  group: api.example.com
  kind: Memcached
  role: memcached
  manageStatus: false
```

The simplest way to invoke the `k8s_status` module is to use its fully
qualified collection name (fqcn), i.e. `operator_sdk.util.k8s_status`.  The
following example updates the `status` subresource with key `memcached` and value `bar`:

```yaml
- operator_sdk.util.k8s_status:
    api_version: app.example.com/v1
    kind: Memcached
    name: "{{ ansible_operator_meta.name }}"
    namespace: "{{ ansible_operator_meta.namespace }}"
    status:
      foo: bar
```

Collections can also be declared in the role's `meta/main.yml`, which is
included for newly scaffolded Ansible operators.

```yaml
collections:
  - operator_sdk.util
```

Declaring collections in the role meta allows you to invoke the
`k8s_status` module directly.

```yaml
- k8s_status:
    <snip>
    status:
      foo: bar
```

### Ansible Operator Conditions

An Ansible Operator has a set of conditions that are used during reconciliation.
There are only a few main conditions:

* Running - the Ansible Operator is currently running the Ansible for
  reconciliation.
* Successful - if the run has finished and there were no errors, the Ansible
  Operator will be marked as Successful. It will then wait for the next
  reconciliation action, either the reconcile period, dependent watches triggers
  or the resource is updated.
* Failed - if there is any error during the reconciliation run, the Ansible
  Operator will be marked as Failed with the error message from the error that
  caused this condition. The error message is the raw output from the Ansible
  run for reconciliation. If the Failure is intermittent, often times the
  situation can be resolved when the Operator reruns the reconciliation loop.

## Extra vars sent to Ansible

The extra vars that are sent to Ansible are managed by the operator. The `spec`
section will pass along the key-value pairs as extra vars.  This is equivalent
to how above extra vars are passed in to `ansible-playbook`. The operator also
passes along additional variables under the `ansible_operator_meta` field for
the name of the CR and the namespace of the CR.

For the CR example:

```yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "memcached-sample"
spec:
  message: "Hello world 2"
  newParameter: "newParam"
```

The structure passed to Ansible as extra vars is:

```json
{ "ansible_operator_meta": {
        "name": "<cr-name>",
        "namespace": "<cr-namespace>",
  },
  "message": "Hello world 2",
  "new_parameter": "newParam",
  "_app_example_com_database": {
     <Full CR>
   },
  "_app_example_com_database_spec": {
     <Full CR .spec>
   },
}
```

`message` and `newParameter` are set in the top level as extra variables, and
`ansible_operator_meta` provides the relevant metadata for the Custom Resource as defined in the
operator. The `ansible_operator_meta` fields can be accessed via dot notation in Ansible as so:

```yaml
---
- debug:
    msg: "name: {{ ansible_operator_meta.name }}, {{ ansible_operator_meta.namespace }}"
```

[ansible-runner-http-plugin]:https://github.com/ansible/ansible-runner-http
[ansible-runner-tool]: https://ansible-runner.readthedocs.io/en/latest/install.html
[k8s_ansible_module]:https://docs.ansible.com/ansible/2.6/modules/k8s_module.html
[tutorial]:../tutorial
[kubernetes_collection]: https://galaxy.ansible.com/kubernetes/core
[manage_status_proposal]:../../proposals/ansible-operator-status.md
[operator_sdk_util]: https://galaxy.ansible.com/operator_sdk/util
[passing_extra_vars]: https://docs.ansible.com/ansible/latest/user_guide/playbooks_variables.html#passing-variables-on-the-command-line
[python-kubernetes-client]: https://github.com/kubernetes-client/python
[time_pkg]:https://golang.org/pkg/time/
[time_parse_duration]:https://golang.org/pkg/time/#ParseDuration
[watches]:/docs/building-operators/ansible/reference/watches
[py-deps]:https://github.com/operator-framework/operator-sdk/blob/c6796de/images/ansible-operator/Pipfile.lock
[pipenv]:https://pypi.org/project/pipenv/
[os-pkgs]:https://github.com/operator-framework/operator-sdk/blob/c6796de/images/ansible-operator/base.Dockerfile#L29
