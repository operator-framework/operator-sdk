# Developer guide

This document provides some useful information and tips for a developer creating an operator powered by Ansible.

## Getting started with the k8s Ansible modules

Since we are interested in using Ansible for the lifecycle management of our application on Kubernetes, it is beneficial for a developer to get a good grasp of the [k8s Ansible module][k8s_ansible_module]. This Ansible module allows a developer to either leverage their existing Kubernetes resource files (written in YaML) or express the lifecycle management in native Ansible. One of the biggest benefits of using Ansible in conjunction with existing Kubernetes resource files is the ability to use Jinja templating so that you can customize deployments with the simplicity of a few variables in Ansible.

The easiest way to get started is to install the modules on your local machine and test them using a playbook.

## Installing the k8s Ansible modules

To install the k8s Ansible modules, you simply need to install Ansible 2.6+. On Fedora/Centos:
```
$ sudo dnf install ansible
```

## 

## Build the Operator SDK CLI

Requirement:
- Go 1.9+

Build the Operator SDK CLI `operator-sdk` binary:

```sh
# TODO: replace this with the ./build script.
$ go install github.com/operator-framework/operator-sdk/commands/operator-sdk 
```

## Testing

Run unit tests:

```sh
TODO: use ./test script
```

[fork_guide]:https://help.github.com/articles/fork-a-repo/
[k8s_ansible_module]:https://docs.ansible.com/ansible/2.6/modules/k8s_module.html
