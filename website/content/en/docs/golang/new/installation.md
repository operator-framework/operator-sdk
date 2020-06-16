---
title: Golang Based Operator SDK Installation
linkTitle: Installation
weight: 1
---

Follow the steps in the [installation guide][install-guide] to learn how to install the Operator SDK CLI tool.

## Additional Prerequisites

- [git][git_tool]
- [go][go_tool] version v1.12+.
- [mercurial][mercurial_tool] version 3.9+
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

**Note**: This guide uses [minikube][minikube-tool] version v0.25.0+ as the
local Kubernetes cluster and [quay.io][quay-link] for the public registry.

[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[mercurial_tool]:https://www.mercurial-scm.org/downloads
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[install-guide]: /docs/install-operator-sdk
[minikube-tool]:https://github.com/kubernetes/minikube#installation
[quay-link]:https://quay.io