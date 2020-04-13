---
title: Development Tips
weight: 12
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
$ operator-sdk new --type ansible --kind Foo --api-version foo.example.com/v1alpha1 foo-operator
Create foo-operator/tmp/init/galaxy-init.sh
Create foo-operator/tmp/build/Dockerfile
Create foo-operator/tmp/build/test-framework/Dockerfile
Create foo-operator/tmp/build/go-test.sh
Rendering Ansible Galaxy role [foo-operator/roles/Foo]...
Cleaning up foo-operator/tmp/init
Create foo-operator/watches.yaml
Create foo-operator/requirements.yml
Create foo-operator/deploy/rbac.yaml
Create foo-operator/deploy/crd.yaml
Create foo-operator/deploy/cr.yaml
Create foo-operator/deploy/operator.yaml
Run git init ...
Initialized empty Git repository in /home/dymurray/go/src/github.com/dymurray/opsdk/foo-operator/.git/
Run git init done

$ cd foo-operator
$ ansible-galaxy collection install -r requirements.yml
```
Modify `roles/Foo/tasks/main.yml` with desired Ansible logic. For this example
we will create and delete a namespace with the switch of a variable:
```yaml
---
- name: set example-memcached namespace to {{ state }}
  community.kubernetes.k8s:
    api_version: v1
    kind: Namespace
    name: example-memcached
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

Task [Foo : set example-memcached namespace to present]
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
example-memcached          Active    3s
```

Rerun the playbook setting `state` to `absent`:
```bash
$ ansible-playbook playbook.yml --extra-vars state=absent
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [Foo : set example-memcached namespace to absent]
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

**ansible.operator-sdk/reconcile-period**: Used to specify the reconciliation
interval for the CR. This value is parsed using the standard Golang package
[time][time_pkg]. Specifically [ParseDuration][time_parse_duration] is used
which will apply the default suffix of `s` giving the value in seconds.

Example:
```
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "example"
annotations:
  ansible.operator-sdk/reconcile-period: "30s"
```

### Testing an Ansible operator locally

Once a developer is comfortable working with the above workflow, it will be
beneficial to test the logic inside of an operator. To accomplish this, we can
use `operator-sdk run --local` from the top-level directory of our project. The
`run --local` command reads from `./watches.yaml` and uses `~/.kube/config` to
communicate with a Kubernetes cluster just as the `k8s` modules do. This
section assumes the developer has read the [Ansible Operator user
guide][ansible_operator_user_guide] and has the proper dependencies installed.

**NOTE:** You can customize the roles path by setting the environment variable `ANSIBLE_ROLES_PATH` or using the flag `ansible-roles-path`. Note that, if the role not be found in the 
customized path informed in `ANSIBLE_ROLES_PATH` then, the operator will look for it in the `{{current directory}}/roles`.   

Create a Custom Resource Definition (CRD) and proper Role-Based Access Control
(RBAC) definitions for resource Foo. `operator-sdk` auto-generates these files
inside of the `deploy` folder:
```bash
$ kubectl create -f deploy/crds/foo.example.com_foos_crd.yaml
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
```

Run the `run --local` command:
```bash
$ operator-sdk run --local
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: 0.0.6+git
INFO[0000] Starting to serve on 127.0.0.1:8888

INFO[0000] Watching foo.example.com/v1alpha1, Foo, default
```

Now that the operator is watching resource `Foo` for events, the creation of a
Custom Resource will trigger our Ansible Role to be executed. Take a look at
`deploy/cr.yaml`:
```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "example"
```

Since `spec` is not set, Ansible is invoked with no extra variables. The next
section covers how extra variables are passed from a Custom Resource to
Ansible. This is why it is important to set sane defaults for the operator.

Create a Custom Resource instance of Foo with default var `state` set to
`present`:
```bash
$ kubectl create -f deploy/cr.yaml
```

Check that namespace `example-memcached` was created:
```bash
$ kubectl get namespace
NAME          	           STATUS    AGE
default       		   Active    28d
kube-public   		   Active    28d
kube-system   		   Active    28d
example-memcached          Active    3s
```

Modify `deploy/cr.yaml` to set `state` to `absent`:
```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "example"
spec:
  state: "absent"
```

Apply the changes to Kubernetes and confirm that the namespace is deleted:
```bash
$ kubectl apply -f deploy/cr.yaml
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
$ operator-sdk build quay.io/example/foo-operator:v0.0.1
$ docker push quay.io/example/foo-operator:v0.0.1
```

Kubernetes deployment manifests are generated in `deploy/operator.yaml`. The
deployment image in this file needs to be modified from the placeholder
`REPLACE_IMAGE` to the previous built image. To do this run:
```
$ sed -i 's|REPLACE_IMAGE|quay.io/example/foo-operator:v0.0.1|g' deploy/operator.yaml
```

**Note**
If you are performing these steps on OSX, use the following command:
```
$ sed -i "" 's|REPLACE_IMAGE|quay.io/example/foo-operator:v0.0.1|g' deploy/operator.yaml
```

Deploy the foo-operator:

```sh
$ kubectl create -f deploy/crds/foo.example.com_foos_crd.yaml # if CRD doesn't exist already
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
$ kubectl create -f deploy/operator.yaml
```

Verify that the foo-operator is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
foo-operator       1         1         1            1           1m
```

#### Viewing the Ansible logs

In order to see the logs from a particular you can run:

```sh
kubectl logs deployment/foo-operator 
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
    name: "{{ meta.name }}"
    namespace: "{{ meta.namespace }}"
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

Please look over the following sections for help debugging an Ansible Operator:


* [View the Ansible logs](../quickstart#view-the-ansible-logs)
* [Additional Ansible debug](../quickstart#additional-ansible-debug)
* [Testing Ansible Operators with Molecule](../testing-guide#testing-ansible-operators-with-molecule)

## Extra vars sent to Ansible
The extra vars that are sent to Ansible are managed by the operator. The `spec`
section will pass along the key-value pairs as extra vars.  This is equivalent
to how above extra vars are passed in to `ansible-playbook`. The operator also
passes along additional variables under the `meta` field for the name of the CR
and the namespace of the CR.

For the CR example:
```yaml
apiVersion: "app.example.com/v1alpha1"
kind: "Database"
metadata:
  name: "example"
spec:
  message:"Hello world 2"
  newParameter: "newParam"
```

The structure passed to Ansible as extra vars is:


```json
{ "meta": {
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
`meta` provides the relevant metadata for the Custom Resource as defined in the
operator. The `meta` fields can be accesses via dot notation in Ansible as so:
```yaml
---
- debug:
    msg: "name: {{ meta.name }}, {{ meta.namespace }}"
```

[k8s_ansible_module]:https://docs.ansible.com/ansible/2.6/modules/k8s_module.html
[openshift_restclient_python]:https://github.com/openshift/openshift-restclient-python
[ansible_operator_user_guide]:../quickstart
[manage_status_proposal]:../../proposals/ansible-operator-status.md
[time_pkg]:https://golang.org/pkg/time/
[time_parse_duration]:https://golang.org/pkg/time/#ParseDuration
