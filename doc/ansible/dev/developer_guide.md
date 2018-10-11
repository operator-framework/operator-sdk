# Developer guide

This document provides some useful information and tips for a developer creating an operator powered by Ansible.

## Getting started with the k8s Ansible modules

Since we are interested in using Ansible for the lifecycle management of our application on Kubernetes, it is beneficial for a developer to get a good grasp of the [k8s Ansible module][k8s_ansible_module]. This Ansible module allows a developer to either leverage their existing Kubernetes resource files (written in YaML) or express the lifecycle management in native Ansible. One of the biggest benefits of using Ansible in conjunction with existing Kubernetes resource files is the ability to use Jinja templating so that you can customize deployments with the simplicity of a few variables in Ansible.

The easiest way to get started is to install the modules on your local machine and test them using a playbook.

## Installing the k8s Ansible modules

To install the k8s Ansible modules, you simply need to install Ansible 2.6+. On Fedora/Centos:
```bash
$ sudo dnf install ansible
```

## Testing the k8s Ansible modules locally

Sometimes it is beneficial for a developer to run the Ansible code from their local machine as opposed to running/rebuilding the operator each time. To do this, initialize a new project:
```bash
$ operator-sdk new --type ansible --kind Foo --api-version foo.example.com/v1alpha1 foo-operator
Create foo-operator/tmp/init/galaxy-init.sh 
Create foo-operator/tmp/build/Dockerfile 
Create foo-operator/tmp/build/test-framework/Dockerfile 
Create foo-operator/tmp/build/go-test.sh 
Rendering Ansible Galaxy role [foo-operator/roles/Foo]...
Cleaning up foo-operator/tmp/init
Create foo-operator/watches.yaml 
Create foo-operator/deploy/rbac.yaml 
Create foo-operator/deploy/crd.yaml 
Create foo-operator/deploy/cr.yaml 
Create foo-operator/deploy/operator.yaml 
Run git init ...
Initialized empty Git repository in /home/dymurray/go/src/github.com/dymurray/opsdk/foo-operator/.git/
Run git init done

$ cd foo-operator
```

Modify `roles/Foo/tasks/main.yml` with desired Ansible logic. For this example we will create and delete a namespace with the switch of a variable:
```yaml
---
- name: set test namespace to {{ state }}
  k8s:
    api_version: v1
    kind: Namespace
    state: "{{ state }}"
```

Modify `roles/Foo/defaults/main.yml` to set `state` to `present` by default.
```yaml
---
state: present
```

Create an Ansible playbook in the top-level directory which includes role `Foo`:
```yaml
---
- hosts: localhost
  roles:
    - Foo
```

Run the playbook:
```bash
$ ansible-playbook playbook.yaml
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [Foo : set test namespace to present]
changed: [localhost]

PLAY RECAP *********************************************************************************
localhost                  : ok=2    changed=1    unreachable=0    failed=0

```

Check that the namespace was created:
```bash
$ kubectl get namespace
NAME          STATUS    AGE
default       Active    28d
kube-public   Active    28d
kube-system   Active    28d
test          Active    3s
```

Rerun the playbook setting `state` to `absent`:
```bash
$ ansible-playbook playbook.yml --extra-vars state=absent
 [WARNING]: provided hosts list is empty, only localhost is available. Note that the implicit localhost does not match 'all'


PLAY [localhost] ***************************************************************************

TASK [Gathering Facts] *********************************************************************
ok: [localhost]

Task [Foo : set test namespace to absent]
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

## Using playbooks in watches.yaml

By default, `operator-sdk new --type ansible` sets `watches.yaml` to execute a role directly on a resource event. This works well for new projects, but with a lot of Ansible code this can be hard to scale if we are putting everything inside of one role. Using a playbook allows the developer to have more flexibility in consuming other roles and enabling more customized deployments of their application. To do this, modify `watches.yaml` to use a playbook instead of the role:
```yaml
---
- version: v1alpha1
  group: foo.example.com
  kind: Foo
  playbook: /opt/ansible/playbook.yml
```

Modify `tmp/build/Dockerfile` to put `playbook.yml` in `/opt/ansible` in the container in addition to the role (`/opt/ansible` is the `HOME` environment variable inside of the Ansible Operator base image):
```Dockerfile
FROM quay.io/water-hole/ansible-operator

COPY roles/ ${HOME}/roles
COPY playbook.yaml ${HOME}/playbook.yaml
COPY watches.yaml ${HOME}/watches.yaml
```

Alternatively, to generate a skeleton project with the above changes, a developer can also do:
```bash
$ operator-sdk new --type ansible --kind Foo --api-version foo.example.com/v1alpha1 foo-operator --generate-playbook
```

## Testing an Ansible operator locally

Once a developer is comfortable working with the above workflow, it will be beneficial to test the logic inside of an operator. To accomplish this, we can use `operator-sdk up local` from the top-level directory of our project. The `up local` command reads from `./watches.yaml` and uses `~/.kube/config` to communicate with a kubernetes cluster just as the `k8s` modules do. This section assumes the developer has read the [Ansible Operator user guide][ansible_operator_user_guide] and has the proper dependencies installed.

Since `up local` reads from `./watches.yaml`, there are a couple options available to the developer. If `role` is left alone (by default `/opt/ansible/roles/<name>`) the developer must copy the role over to `/opt/ansible/roles` from the operator directly. This is cumbersome because changes will not be reflected from the current directory. It is recommended that the developer instead change the `role` field to point to the current directory and simply comment out the existing line:
```yaml
- version: v1alpha1
  group: foo.example.com
  kind: Foo
  #  role: /opt/ansible/roles/Foo
  role: /home/user/foo-operator/Foo
```

[k8s_ansible_module]:https://docs.ansible.com/ansible/2.6/modules/k8s_module.html
[ansible_operator_user_guide]:../user-guide.md
