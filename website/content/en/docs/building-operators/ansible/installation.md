---
title: Ansible Operator SDK Installation
linkTitle: Installation
weight: 1
---

Follow the steps in the [installation guide][install-guide] to learn how to install the Operator SDK CLI tool.

## Additional Prerequisites

- [ansible][ansible-tool] version v2.9.0+
- [ansible-runner][ansible-runner-tool] version v1.1.0+
- [ansible-runner-http][ansible-runner-http-plugin] version v1.0.0+

**Note**: This guide uses [minikube][minikube-tool] version v0.25.0+ as the
local Kubernetes cluster and [quay.io][quay-link] for the public registry.


[ansible-tool]:https://docs.ansible.com/ansible/latest/index.html
[ansible-runner-tool]:https://ansible-runner.readthedocs.io/en/latest/install.html
[ansible-runner-http-plugin]:https://github.com/ansible/ansible-runner-http
[install-guide]: /docs/installation/install-operator-sdk
[minikube-tool]:https://github.com/kubernetes/minikube#installation
[quay-link]:https://quay.io
