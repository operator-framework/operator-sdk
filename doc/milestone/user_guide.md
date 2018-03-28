# Getting Started

## Prerequisites

- [dep][dep_tool] version v0.4.1+.
- [go][go_tool] version v1.10+.
- [docker][docker_tool] version 17.03+.
- Access to a public registry such as quay.io. 
- [kubectl][kubectl_tool] version v1.9.0+.
- Access to a kubernetes v.1.9.0+ cluster.

**Note**: This guide uses [minikube][minikube_tool] version v0.25.0+ as the local kubernetes cluster and quay.io for the public registry.

## Installing Operator SDK CLI

The Operator SDK comes with a CLI tool that manages the development lifecycle. It helps create the project scaffolding, preprocess custom resource API to generate Kubernetes related code, and generate deployment scripts.

Checkout the desired release tag and install the SDK CLI tool:
```
git checkout tags/v0.0.2
go install github.com/coreos/operator-sdk/commands/operator-sdk
```

This will install the CLI binary `operator-sdk` at `$GOPATH/bin`.

## Creating a new project

TODO

[scaffold_doc]:./doc/project_layout.md
[mc_protocol]:https://github.com/memcached/memcached/blob/master/doc/protocol.txt
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
[operator_link]:https://coreos.com/operators/
[design_docs]:./doc/design