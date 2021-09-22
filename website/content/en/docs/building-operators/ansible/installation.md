---
title: Installation Guide
linkTitle: Installation
weight: 1
---

## Install `operator-sdk`

Follow the steps in the [installation guide][install-guide] to learn how to install the `operator-sdk` CLI tool.

## Additional Prerequisites

- [docker][docker_tool] version 17.03+
- [python][python] version 3.8.6+
- [ansible][ansible] version v2.9.0+
- [ansible-runner][ansible-runner] version 2.0.2+
- [ansible-runner-http][ansible-runner-http-plugin] version v1.0.0+
- [openshift][openshift-module] version v0.12.0+
- [kubectl][kubectl_tool] and access to a Kubernetes cluster of a [compatible version][k8s-version-compat].

[docker_tool]:https://docs.docker.com/install/
[install-guide]:/docs/installation/
[python]:https://www.python.org/downloads/
[ansible]:https://docs.ansible.com/ansible/latest/index.html
[ansible-runner]:https://ansible-runner.readthedocs.io/en/latest/install.html
[ansible-runner-http-plugin]:https://github.com/ansible/ansible-runner-http
[openshift-module]:https://pypi.org/project/openshift/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[k8s-version-compat]:/docs/overview#kubernetes-version-compatibility
