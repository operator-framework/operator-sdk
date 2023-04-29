---
title: "Project Layout"
linkTitle: "Project Layout"
weight: 2
description: A description of the layout of projects built with Operator SDK
---

## Operator SDK Project Layout

All projects initialized with `operator-sdk init` have a common base structure which builds on [kubebuilder's project layout][kb-whats in-a-basic-project?]. Each project type is customized further with code of that type's language.

### Common Base

The common structure contains the following items:

| File/Directory | Description | 
| ------ | ----- |
| `Dockerfile` | The Dockerfile of your operator project, used to build the image with `make docker-build`. |
| `Makefile` | Build file with helper targets to help you work with your project. |
| `PROJECT` | This file represents the project's configuration and is used to track useful information for the CLI and plugins. |
| `bin/` | This directory contains useful binaries such as the `manager` which is used to run your project locally and  the `kustomize` utility used for the project configuration. For other language types, it might have other binaries useful for developing your operator. |
| `bundle/` | This directory contains all the files used to [integrate your project][olm-integrate] with [OLM][olm] with the [bundle][bundle] format. It is built from the Makefile target `make bundle`. |
| `bundle/manifests/` | This directory has the [OLM manifests][olm-manifests] of your [bundle][bundle]. |
| `bundle/metadata/`  | This directory has the [OLM metadata][olm-metadata] of your [bundle][bundle] e.g the index image annotations. |
| `bundle/tests/` | This directory has the [Scorecard][scorecard] tests shipped with your operator bundle. |
| `config/` | Contains configuration files to launch your project on a cluster. Plugins might use it to provide functionality. For example, for the CLI  to help create your operator bundle it will look for the CRD's and CR's which are scaffolded in this directory. You will also find all [Kustomize][Kustomize] YAML definitions as well. |
| `config/crd/` | Contains the [Custom Resources Definitions][k8s-crd-doc]. |
| `config/default/` | Contains a [Kustomize base][kustomize-base] for launching the controller in a standard configuration. |
| `config/manager/` | Contains the manifests to launch your operator project as pods on the cluster. |
| `config/manifests/` | Contains the base to generate your OLM manifests in the bundle directory. |
| `config/prometheus/` | Contains the manifests required to enable project to serve metrics to [Prometheus][kb-metrics] such as the `ServiceMonitor` resource. |
| `config/scorecard/` | Contains the manifests required to allow you test your project with [Scorecard][scorecard]. |
| `config/rbac/` | Contains the [RBAC][k8s-rbac] permissions required to run your project. |
| `config/samples/` | Contains the [Custom Resources][k8s-cr-doc]. |
| `bundle.Dockerfile` |  The Dockerfile to build the [bundle][bundle] image. Used to build the operator bundle image with `make bundle-build. |

### Ansible

Now, let's look at the files and directories specific to Ansible-based operators.

| File/Directory | Description | 
| ------ | ----- |
|`config/testing/` | Manifest files to help you test your project. For example, to change the image policy for your [Molecule tests][ansible-test-guide] or to enable debug level in the Ansible logs. |
|`molecule/` | Contain the manifests for your [Molecule][molecule] tests. |
|`molecule/default` | Contains the default [Molecule][molecule] task. |
|`molecule/kind` | Contains the [Molecule][molecule] task to be executed on the cluster. |
|`playbooks/` | Contains the Ansible playbooks.|
|`roles/` | Contains the Ansible role files for each Kind scaffold. |
|`requirements.yml` | This file specifies Ansible dependencies that need to be installed for your operator to function. |
|`watches.yaml` | Contains Group, Version, Kind, and the playbooks and rules location. Used to configure the [Ansible watches][ansible-watches]. |

### Golang 

Now, let's look at the files and directories specific to Go-based operators.

| File/Directory | Description |
| ------ | ----- |
|`api/` | Contains the api definition |
|`config/certmanager` |  Contains the Kustomize manifests which configure the [cert-manager][cert-manager] by the Webhooks. |
|`config/webhook` | Contains the Kustomize manifests to configure the Webhook. |
|`controllers` |  Contains the controllers. |
|`main.go` | Implements the project initialization. |
| `hack/` | Contains utility files, e.g. the file used to scaffold the license header for your project files. |

### Helm 

Now, let's look at the files and directories specific to Helm-based operators.

| File/Directory | Description | 
| ------ | ----- |
|`helm-charts` | Contains the Helm charts for each Kind scaffold which can be initialized with `operator-sdk init --plugins=helm [options]` or `operator-sdk create api [options]` . |
|`watches.yaml` | Contains Group, Version, Kind, and Helm chart location. Used to configure the [Helm watches][helm-watches]. |

[kb-whats in-a-basic-project?]: https://book.kubebuilder.io/cronjob-tutorial/basic-project.html
[olm]: https://github.com/operator-framework/operator-lifecycle-manager
[Kustomize]: https://github.com/kubernetes-sigs/kustomize
[kustomize-base]: https://github.com/operator-framework/operator-sdk/blob/v1.4.2/testdata/go/v3/memcached-operator/config/default/kustomization.yaml
[kb-metrics]: https://book.kubebuilder.io/reference/metrics.html
[k8s-rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[k8s-cr-doc]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-resources
[k8s-crd-doc]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
[scorecard]: /docs/testing-operators/scorecard/
[olm-integrate]: /docs/olm-integration/
[olm-manifests]: https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format  
[olm-metadata]: https://github.com/operator-framework/operator-registry/blob/v1.16.1/docs/design/operator-bundle.md#bundle-manifest-format
[bundle]:https://github.com/operator-framework/operator-registry/blob/v1.16.1/docs/design/operator-bundle.md
[molecule]: https://molecule.readthedocs.io/
[ansible-watches]: /docs/building-operators/ansible/reference/watches
[ansible-test-guide]: /docs/building-operators/ansible/testing-guide
[helm-watches]: /docs/building-operators/helm/reference/watches
[cert-manager]: https://cert-manager.io/docs/
