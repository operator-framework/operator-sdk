<img src="doc/images/operator_logo_sdk_color.svg" height="125px"></img>

[![Build Status](https://travis-ci.org/operator-framework/operator-sdk.svg?branch=master)](https://travis-ci.org/operator-framework/operator-sdk)

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

- [git][git_tool]
- [go][go_tool] version v1.12+.
- [mercurial][mercurial_tool] version 3.9+
- [docker][docker_tool] version 17.03+.
  - Alternatively [podman][podman_tool] `v1.2.0+` or [buildah][buildah_tool] `v1.7+`
- [kubectl][kubectl_tool] version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- Optional: [dep][dep_tool] version v0.5.0+.
- Optional: [delve](https://github.com/go-delve/delve/tree/master/Documentation/installation) version 1.2.0+ (for `up local --enable-delve`).

## Quick Start

### Install the Operator SDK CLI

Follow the steps in the [installation guide][install_guide] to learn how to install the Operator SDK CLI tool.

**Note:** If you are using a release version of the SDK, make sure to follow the documentation for that version. e.g.: For version 0.8.1, see docs in https://github.com/operator-framework/operator-sdk/tree/v0.8.1.

### Create and deploy an app-operator

```sh
# Create an app-operator project that defines the App CR.
$ mkdir -p $HOME/projects/example-inc/
# Create a new app-operator project
$ cd $HOME/projects/example-inc/
$ export GO111MODULE=on
$ operator-sdk new app-operator --repo github.com/example-inc/app-operator
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

# Test the new Resource Type
$ kubectl describe appservice example-appservice
Name:         example-appservice
Namespace:    myproject
Labels:       <none>
Annotations:  <none>
API Version:  app.example.com/v1alpha1
Kind:         AppService
Metadata:
  Cluster Name:        
  Creation Timestamp:  2018-12-17T21:18:43Z
  Generation:          1
  Resource Version:    248412
  Self Link:           /apis/app.example.com/v1alpha1/namespaces/myproject/appservices/example-appservice
  UID:                 554f301f-0241-11e9-b551-080027c7d133
Spec:
  Size:  3

# Cleanup
$ kubectl delete -f deploy/crds/app_v1alpha1_appservice_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/service_account.yaml
$ kubectl delete -f deploy/crds/app_v1alpha1_appservice_crd.yaml
```
## Command Line Interface

To learn more about the SDK CLI, see the [SDK CLI Reference][sdk_cli_ref], or run `operator-sdk [command] -h`.

## User Guides

To learn more about the writing an operator in Go, see the [user guide][guide].

The SDK also supports developing an operator using Ansible or Helm. See the [Ansible][ansible_user_guide] and [Helm][helm_user_guide] operator user guides.

## Samples

To explore any operator samples built using the operator-sdk, see the [operator-sdk-samples][samples].

## FAQ

For common Operator SDK related questions, see the [FAQ][faq].

## Contributing

See [CONTRIBUTING][contrib] for details on submitting patches and the contribution workflow.

See the [proposal docs][proposals_docs] and issues for ongoing or planned work.

## Reporting bugs

See [reporting bugs][bug_guide] for details about reporting any issues.

## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file for details.

[install_guide]: ./doc/user/install-operator-sdk.md
[operator_link]: https://coreos.com/operators/
[proposals_docs]: ./doc/proposals
[sdk_cli_ref]: ./doc/sdk-cli-reference.md
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
[mercurial_tool]:https://www.mercurial-scm.org/downloads
[docker_tool]:https://docs.docker.com/install/
[podman_tool]:https://github.com/containers/libpod/blob/master/install.md
[buildah_tool]:https://github.com/containers/buildah/blob/master/install.md
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[controller_runtime]: https://github.com/kubernetes-sigs/controller-runtime
[ansible_user_guide]:./doc/ansible/user-guide.md
[helm_user_guide]:./doc/helm/user-guide.md
[faq]: ./doc/faq.md
