<img src="doc/images/operator_logo_sdk_color.svg" height="125px"></img>

[![Build Status](https://travis-ci.org/operator-framework/operator-sdk.svg?branch=master)](https://travis-ci.org/operator-framework/operator-sdk)

### Project Status: alpha

The project is currently alpha which means that there are still new features and APIs planned that will be added in the future. Due to this breaking changes may still happen.

**Note:** The core APIs provided by the [controller-runtime][controller_runtime] will most likely stay unchanged however the expectation is that any breaking changes should be relatively minor and easier to handle than the changes from SDK `v0.0.7` to `v0.1.0`.

See the [proposal docs][proposals_docs] and issues for ongoing or planned work.

## Overview

This project is a component of the [Operator Framework][of-home], an open source toolkit to manage Kubernetes native applications, called Operators, in an effective, automated, and scalable way. Read more in the [introduction blog post][of-blog].

[Operators][operator_link] make it easy to manage complex stateful applications on top of Kubernetes. However writing an operator today can be difficult because of challenges such as using low level APIs, writing boilerplate, and a lack of modularity which leads to duplication.

The Operator SDK is a framework that uses the [controller-runtime][controller_runtime] library to make writing operators easier by providing:
- High level APIs and abstractions to write the operational logic more intuitively
- Tools for scaffolding and code generation to bootstrap a new project fast
- Extensions to cover common operator use cases

## Workflow

The SDK provides workflows to develop operators in Go, Ansible, or Helm.

The following workflow is for a new **Go** operator:
1. Create a new operator project using the SDK Command Line Interface(CLI)
2. Define new resource APIs by adding Custom Resource Definitions(CRD)
3. Define Controllers to watch and reconcile resources
4. Write the reconciling logic for your Controller using the SDK and controller-runtime APIs
5. Use the SDK CLI to build and generate the operator deployment manifests

The following workflow is for a new **Ansible** operator:
1. Create a new operator project using the SDK Command Line Interface(CLI)
2. Write the reconciling logic for your object using ansible playbooks and roles
3. Use the SDK CLI to build and generate the operator deployment manifests
4. Optionally add additional CRD's using the SDK CLI and repeat steps 2 and 3

The following workflow is for a new **Helm** operator:
1. Create a new operator project using the SDK Command Line Interface(CLI)
2. Create a new (or add your existing) Helm chart for use by the operator's reconciling logic
3. Use the SDK CLI to build and generate the operator deployment manifests
4. Optionally add additional CRD's using the SDK CLI and repeat steps 2 and 3

## Prerequisites

- [dep][dep_tool] version v0.5.0+.
- [git][git_tool]
- [go][go_tool] version v1.10+.
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.0+.
- Access to a kubernetes v.1.11.0+ cluster.

## Quick Start

First, checkout and install the operator-sdk CLI:

```sh
$ mkdir -p $GOPATH/src/github.com/operator-framework
$ cd $GOPATH/src/github.com/operator-framework
$ git clone https://github.com/operator-framework/operator-sdk
$ cd operator-sdk
$ git checkout master
$ make dep
$ make install
```

Create and deploy an app-operator using the SDK CLI:

```sh
# Create an app-operator project that defines the App CR.
$ mkdir -p $GOPATH/src/github.com/example-inc/
# Create a new app-operator project
$ cd $GOPATH/src/github.com/example-inc/
$ operator-sdk new app-operator
$ cd app-operator

# Add a new API for the custom resource AppService
$ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService

# Add a new controller that watches for AppService
$ operator-sdk add controller --api-version=app.example.com/v1alpha1 --kind=AppService

# Build and push the app-operator image to a public registry such as quay.io
$ operator-sdk build quay.io/example/app-operator
$ docker push quay.io/example/app-operator

# Update the operator manifest to use the built image name (if you are performing these steps on OSX, see note below)
$ sed -i 's|REPLACE_IMAGE|quay.io/example/app-operator|g' deploy/operator.yaml
# On OSX use:
$ sed -i "" 's|REPLACE_IMAGE|quay.io/example/app-operator|g' deploy/operator.yaml

# Setup Service Account
$ kubectl create -f deploy/service_account.yaml
# Setup RBAC
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
# Setup the CRD
$ kubectl create -f deploy/crds/app_v1alpha1_appservice_crd.yaml
# Deploy the app-operator
$ kubectl create -f deploy/operator.yaml

# Create an AppService CR
# The default controller will watch for AppService objects and create a pod for each CR
$ kubectl create -f deploy/crds/app_v1alpha1_appservice_cr.yaml

# Verify that a pod is created
$ kubectl get pod -l app=example-appservice
NAME                     READY     STATUS    RESTARTS   AGE
example-appservice-pod   1/1       Running   0          1m

# Cleanup
$ kubectl delete -f deploy/crds/app_v1alpha1_appservice_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/app_v1alpha1_appservice_crd.yaml
```

## User Guides

To learn more about the writing an operator in Go, see the [user guide][guide].

The SDK also supports developing an operator using Ansible or Helm. See the [Ansible][ansible_user_guide] and [Helm][helm_user_guide] operator user guides.

## Samples

To explore any operator samples built using the operator-sdk, see the [operator-sdk-samples][samples].

## Contributing

See [CONTRIBUTING][contrib] for details on submitting patches and the contribution workflow.

## Reporting bugs

See [reporting bugs][bug_guide] for details about reporting any issues.

## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file for details.

[operator_link]: https://coreos.com/operators/
[proposals_docs]: ./doc/proposals
[guide]: ./doc/user-guide.md
[samples]: https://github.com/operator-framework/operator-sdk-samples
[of-home]: https://github.com/operator-framework
[of-blog]: https://coreos.com/blog/introducing-operator-framework
[contrib]: ./CONTRIBUTING.MD
[bug_guide]:./doc/dev/reporting_bugs.md
[license_file]:./LICENSE
[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[controller_runtime]: https://github.com/kubernetes-sigs/controller-runtime
[ansible_user_guide]:./doc/ansible/user-guide.md
[helm_user_guide]:./doc/helm/user-guide.md
