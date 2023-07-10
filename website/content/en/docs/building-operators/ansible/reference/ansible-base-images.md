---
title: Ansible Operator Base Images
linkTitle: Base Images
weight: 20
---

Ansible-based operators are built on top of base images built for use
with the Operator-SDK.


## Ansible Versions

For Operator-SDK versions > `v1.30.0` the `quay.io/operator-framework/ansible-operator`
base image has been updated to use Ansible 2.15. The Ansible 2.11 preview base image
has been removed and will no longer be built/supported past Operator-SDK `v1.30`.

For Operator-SDK versions <= `v1.30.0`, the below information applies:

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

For Operator-SDK versions > `v1.30.0` - Operators are scaffolded with
the latest version of the base image (using Ansible 2.15). The base image
name is the same as previous versions of the Operator-SDK.

For Operator-SDK versions <= `v1.30.0`, the below information applies:

Operators are scaffolded with the latest version of the base image
(using Ansible 2.9) in the first line of the operator Dockerfile.

`FROM quay.io/operator-framework/ansible-operator:v1.16`

Operator authors who want to try out 2.11 can simply replace their FROM with:
`FROM quay.io/operator-framework/ansible-operator-2.11-preview:v1.16`
