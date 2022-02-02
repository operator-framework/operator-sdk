---
title: Ansible Operator Base Images
linkTitle: Base Images
weight: 20
---

Ansible-based operators are built on top of base images built for use
with the Operator-SDK.


## Ansible Versions

There have been some major changes in the Ansible ecosystem, primarily
the addition of `collections` and the removal of these libraries from
Ansible core. Ansible 2.9 is the last release of the "old way", with
forward compatibility for collections. This is the version officially
supported by the Operator-SDK.

Ansible 2.10 was a transition release, and it is NOT recommended for use
with operators.

Ansible 2.11 is the future, and Operator-SDK will eventually provide only 2.11 images. Currently, 2.11 base images are in tech-preview.
See: [ansible-operator-2.11-preview](https://quay.io/repository/operator-framework/ansible-operator-2.11-preview).

## Changing Ansible Operator Base Images

Operators are scaffolded with the latest version of the base image
(using Ansible 2.9) in the first line of the operator Dockerfile.

`FROM quay.io/operator-framework/ansible-operator:v1.16`

Operator authors who want to try out 2.11 can simply replace their FROM with:
`FROM quay.io/operator-framework/ansible-operator-2.11-preview:v1.16`
