---
title: Development Tips
weight: 5
---

This document provides some useful information and tips for a developer
creating an operator powered by Ansible.

## Getting started with the k8s Ansible modules

Since we are interested in using Ansible for the lifecycle management of our
application on Kubernetes, it is beneficial for a developer to get a good grasp
of the [k8s Ansible module][k8s_ansible_module]. This Ansible module allows a
developer to either leverage their existing Kubernetes resource files (written
in YaML) or express the lifecycle management in native Ansible. One of the
biggest benefits of using Ansible in conjunction with existing Kubernetes
resource files is the ability to use Jinja templating so that you can customize
deployments with the simplicity of a few variables in Ansible.

The easiest way to get started is to install the modules on your local machine
and test them using a playbook.

### Installing the k8s Ansible modules

To install the k8s Ansible modules, one must first install Ansible 2.9+. On
Fedora/Centos:
```bash
$ sudo dnf install ansible
```

In addition to Ansible, a user must install the [OpenShift Restclient
Python][openshift_restclient_python] package. This can be installed from pip:
```bash
$ pip3 install openshift
```

Finally, a user must install the Ansible Kubernetes collection from ansible-galaxy:
```bash
$ ansible-galaxy collection install community.kubernetes
```

Alternatively, if you've already initialized your operator, you will have a `requirements.yml`
file at the top level of your project. This file specifies Ansible dependencies that
need to be installed for your operator to function. By default it will install the
`community.kubernetes` collection, which are used to interact with the Kubernetes API, as well
as the `operator_sdk.util` collection, which provides modules and plugins for operator-specific
operations. To install the Ansible modules from this file, run
```bash
$ ansible-galaxy collection install -r requirements.yml
```

### Testing the k8s Ansible modules locally

Sometimes it is beneficial for a developer to run the Ansible code from their
local machine as opposed to running/rebuilding the operator each time. To do
this, initialize a new project:
```bash
$ mkdir foo-operator && cd foo-operator
$ operator-sdk init --plugins=ansible --domain=example.com --group=foo --version=v1alpha1 --kind=Foo --generate-role
$ ansible-galaxy collection install -r requirements.yml
```
Modify `roles/Foo/tasks/main.yml` with desired Ansible logic. For this example
we will create and delete a namespace with the switch of a variable:
```yaml
---
- name: set foo-sample namespace to {{ state }}
  community.kubernetes.k8s:
    api_version: v1
    kind: Namespace
    name: foo-sample
    state: "{{ state }}"
  ignore_errors: true
```
**note**: Setting `ignore_errors: true` is done so that deleting a nonexistent
project doesn't error out.

Modify `roles/Foo/defaults/main.yml` to set `state` to `present` by default.
```yaml
---
state: present
```

Create an Ansible playbook `playbook.yml` in the top-level directory which
includes role `Foo`:
```yaml
---
- hosts: localhost
  roles:
    - Foo
```

Run the playbook:
```bash
$ ansible-playbook playbook.yml
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [Foo : set foo-sample namespace to present]
changed: [localhost]

PLAY RECAP *********************************************************************************
localhost                  : ok=2    changed=1    unreachable=0    failed=0

```

Check that the namespace was created:
```bash
$ kubectl get namespace
NAME          	           STATUS    AGE
default       	           Active    28d
kube-public   		   Active    28d
kube-system   	           Active    28d
foo-sample                 Active    3s
```

Rerun the playbook setting `state` to `absent`:
```bash
$ ansible-playbook playbook.yml --extra-vars state=absent
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [Foo : set foo-sample namespace to absent]
changed: [localhost]

PLAY RECAP *********************************************************************************
localhost                  : ok=2    changed=1    unreachable=0    failed=0

```

Check that the namespace was deleted:
```bash
$ kubectl get namespace
NAME          STATUS    AGE
default       Active    28d
kube-public   Active    28d
kube-system   Active    28d
```

## Using Ansible inside of an Operator

Now that we have demonstrated using the Ansible Kubernetes modules, we want to
trigger this Ansible logic when a custom resource changes. In the above
example, we want to map a role to a specific Kubernetes resource that the
operator will watch. This mapping is done in a file called `watches.yaml`.

### Custom Resource file

The Custom Resource file format is Kubernetes resource file. The object has
mandatory fields:

**apiVersion**:  The version of the Custom Resource that will be created.

**kind**:  The kind of the Custom Resource that will be created

**metadata**:  Kubernetes specific metadata to be created

**spec**:  This is the key-value list of variables which are passed to Ansible.
This field is optional and will be empty by default.

**annotations**: Kubernetes specific annotations to be appended to the CR. See
the below section for Ansible Operator specific annotations.

#### Ansible Operator annotations
This is the list of CR annotations which will modify the behavior of the operator:

**ansible.sdk.operatorframework.io/reconcile-period**: Specifies the maximum time before a reconciliation is triggered. Note that at scale, this can reduce performance, see [watches][watches] reference for more information. This value is parsed using the standard Golang package
[time][time_pkg]. Specifically [ParseDuration][time_parse_duration] is used
which will apply the default suffix of `s` giving the value in seconds.

Example:
```
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "foo-sample"
annotations:
  ansible.sdk.operatorframework.io/reconcile-period: "30s"
```

Note that a lower period will correct entropy more quickly, but reduce responsiveness to change 
if there are many watched resources. Typically, this option should only be used in advanced use cases where `watchDependentResources` is set to `False`  and when is not possible to use the watch feature. E.g To managing external resources that don’t raise Kubernetes events.

### Testing an Ansible operator locally

**Prerequisites**: Ensure that [Ansible Runner][ansible-runner-tool] and [Ansible Runner
HTTP Plugin][ansible-runner-http-plugin] is installed or else you will see
unexpected errors from Ansible Runner when a Custom Resource is created.

Once a developer is comfortable working with the above workflow, it will be
beneficial to test the logic inside of an operator. To accomplish this, we can
use `make run` from the top-level directory of our project. The `make run`
Makefile target runs the `ansible-operator` binary locally, which reads from
`./watches.yaml` and uses `~/.kube/config` to communicate with a Kubernetes
cluster just as the `k8s` modules do. This section assumes the developer has
read the [Ansible Operator user guide][ansible_operator_user_guide] and has
the proper dependencies installed.

**NOTE:** You can customize the roles path by setting the environment variable
`ANSIBLE_ROLES_PATH` or using the flag `ansible-roles-path`. Note that if the
role is not found in `ANSIBLE_ROLES_PATH`, then the operator will look for it
in `{{current directory}}/roles`.   

Create a Custom Resource Definition (CRD) and proper Role-Based Access Control
(RBAC) definitions for resource Foo.
```bash
$ make install
```

Run the `make run` command:
```bash
$ make run
/home/user/go/bin/ansible-operator
{"level":"info","ts":1595899073.9861593,"logger":"cmd","msg":"Version","Go Version":"go1.13.12","GOOS":"linux","GOARCH":"amd64","ansible-operator":"v0.19.0+git"}
{"level":"info","ts":1595899073.987384,"logger":"cmd","msg":"WATCH_NAMESPACE environment variable not set. Watching all namespaces.","Namespace":""}
{"level":"info","ts":1595899074.9504397,"logger":"controller-runtime.metrics","msg":"metrics server is starting to listen","addr":":8080"}
{"level":"info","ts":1595899074.9522583,"logger":"watches","msg":"Environment variable not set; using default value","envVar":"ANSIBLE_VERBOSITY_MEMCACHED_CACHE_EXAMPLE_COM","default":2}
{"level":"info","ts":1595899074.9524004,"logger":"cmd","msg":"Environment variable not set; using default value","Namespace":"","envVar":"ANSIBLE_DEBUG_LOGS","ANSIBLE_DEBUG_LOGS":false}
{"level":"info","ts":1595899074.9524298,"logger":"ansible-controller","msg":"Watching resource","Options.Group":"cache.example.com","Options.Version":"v1","Options.Kind":"Memcached"}
```

Now that the operator is watching resource `Foo` for events, the creation of a
Custom Resource will trigger our Ansible Role to be executed. Take a look at
`config/samples/foo_v1alpha1_foo.yaml`:
```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "foo-sample"
```

Since `spec` is not set, Ansible is invoked with no extra variables. The next
section covers how extra variables are passed from a Custom Resource to
Ansible. This is why it is important to set sane defaults for the operator.

Create a Custom Resource instance of Foo with default var `state` set to
`present`:
```bash
$ kubectl create -f config/samples/foo_v1alpha1_foo.yaml
```

Check that namespace `foo-sample` was created:
```bash
$ kubectl get namespace
NAME          	           STATUS    AGE
default       		   Active    28d
kube-public   		   Active    28d
kube-system   		   Active    28d
foo-sample                 Active    3s
```

Modify `config/samples/foo_v1alpha1_foo.yaml` to set `state` to `absent`:
```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: foo-sample
spec:
  state: "absent"
```

Apply the changes to Kubernetes and confirm that the namespace is deleted:
```bash
$ kubectl apply -f config/samples/foo_v1alpha1_foo.yaml
$ kubectl get namespace
NAME          STATUS    AGE
default       Active    28d
kube-public   Active    28d
kube-system   Active    28d
```

### Testing an Ansible operator on a cluster

Now that a developer is confident in the operator logic, testing the operator
inside of a pod on a Kubernetes cluster is desired. Running as a pod inside a
Kubernetes cluster is preferred for production use.

To build the `foo-operator` image and push it to a registry:
```
$ make docker-build docker-push IMG=quay.io/example/foo-operator:v0.0.1
```

Deploy the foo-operator:

```sh
$ make install
$ make deploy IMG=quay.io/example/foo-operator:v0.0.1
```

Verify that the foo-operator is up and running:

```sh
$ kubectl get deployment -n foo-operator-system
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
foo-operator       1         1         1            1           1m
```

### Viewing the Ansible logs

In order to see the logs from a particular you can run:

```sh
kubectl logs deployment/foo-operator-controller-manager -n foo-operator-system
```

The logs contain the information about the Ansible run and will make it much easier to debug issues within your Ansible tasks.
Note that the logs will contain much more detailed information about the Ansible Operator's internals and interface with Kubernetes as well.

Also, you can use the environment variable `ANSIBLE_DEBUG_LOGS` set as `True` to check the full Ansible result in the logs in order to be able to debug it.

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
The operator will automatically update the CR's `status` subresource with
generic information about the previous Ansible run. This includes the number of
successful and failed tasks and relevant error messages as seen below:

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

Ansible Operator also allows you as the developer to supply custom
status values with the `k8s_status` Ansible Module, which is included in
[operator_sdk util collection](https://galaxy.ansible.com/operator_sdk/util).

This allows the developer to update the `status` from within Ansible
with any key/value pair as desired. By default, Ansible Operator will
always include the generic Ansible run output as shown above. If you
would prefer your application *not* update the status with Ansible
output and would prefer to track the status manually from your
application, then simply update the watches file with `manageStatus`:

```yaml
- version: v1
  group: api.example.com
  kind: Foo
  role: Foo
  manageStatus: false
```

The simplest way to invoke the `k8s_status` module is to
use its fully qualified collection name (fqcn). To update the
`status` subresource with key `foo` and value `bar`, `k8s_status` can be
used as shown:

```yaml
- operator_sdk.util.k8s_status:
    api_version: app.example.com/v1
    kind: Foo
    name: "{{ ansible_operator_meta.name }}"
    namespace: "{{ ansible_operator_meta.namespace }}"
    status:
      foo: bar
```

Collections can also be declared in the role's `meta/main.yml`, which is
included for new scaffolded ansible operators.

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
The Ansible Operator has a set of conditions which it will use as it performs
its reconciliation procedure. There are only a few main conditions:

* Running - the Ansible Operator is currently running the Ansible for
  reconciliation.

* Successful - if the run has finished and there were no errors, the Ansible
  Operator will be marked as Successful. It will then wait for the next
  reconciliation action, either the reconcile period, dependent watches triggers
  or the resource is updated.

* Failed - if there is any error during the reconciliation run, the Ansible
  Operator will be marked as Failed with the error message from the error that
  caused this condition. The error message is the raw output from the Ansible
  run for reconciliation. If the failure is intermittent, often times the
  situation can be resolved when the Operator reruns the reconciliation loop.

## Extra vars sent to Ansible
The extra vars that are sent to Ansible are managed by the operator. The `spec`
section will pass along the key-value pairs as extra vars.  This is equivalent
to how above extra vars are passed in to `ansible-playbook`. The operator also
passes along additional variables under the `ansible_operator_meta` field for
the name of the CR and the namespace of the CR.

For the CR example:
```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "foo-sample"
spec:
  message:"Hello world 2"
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
[openshift_restclient_python]:https://github.com/openshift/openshift-restclient-python
[ansible_operator_user_guide]:../tutorial
[manage_status_proposal]:../../proposals/ansible-operator-status.md
[time_pkg]:https://golang.org/pkg/time/
[time_parse_duration]:https://golang.org/pkg/time/#ParseDuration
[watches]:/docs/building-operators/ansible/reference/watches
