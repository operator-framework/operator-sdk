---
title: "Overview"
linkTitle: "Overview"
weight: 1
description: >
    What is Operator SDK? Why should I use it?
---

## What is Operator SDK and why should I use it?

This project is a component of the [Operator Framework][of-home], an open source toolkit to manage Kubernetes native applications, called Operators, in an effective, automated, and scalable way. Read more in the [introduction blog post][of-blog].

[Operators][operator_link] make it easy to manage complex stateful applications on top of Kubernetes. However writing an operator today can be difficult because of challenges such as using low level APIs, writing boilerplate, and a lack of modularity which leads to duplication.

The Operator SDK is a framework that uses the [controller-runtime][controller_runtime] library to make writing operators easier by providing:

  - High level APIs and abstractions to write the operational logic more intuitively
  - Tools for scaffolding and code generation to bootstrap a new project fast
  - Extensions to cover common operator use cases

## Workflow

The SDK provides workflows to develop operators in Go, Ansible, or Helm.

The following workflow is for a new [Go operator][golang-guide]:

  1. Create a new operator project using the SDK Command Line Interface(CLI)
  2. Define new resource APIs by adding Custom Resource Definitions(CRD)
  3. Define Controllers to watch and reconcile resources
  4. Write the reconciling logic for your Controller using the SDK and controller-runtime APIs
  5. Use the SDK CLI to build and generate the operator deployment manifests

The following workflow is for a new [Ansible operator][ansible-guide]:

  1. Create a new operator project using the SDK Command Line Interface(CLI)
  2. Write the reconciling logic for your object using ansible playbooks and roles
  3. Use the SDK CLI to build and generate the operator deployment manifests
  4. Optionally add additional CRD's using the SDK CLI and repeat steps 2 and 3

The following workflow is for a new [Helm operator][helm-guide]:

  1. Create a new operator project using the SDK Command Line Interface(CLI)
  2. Create a new (or add your existing) Helm chart for use by the operator's reconciling logic
  3. Use the SDK CLI to build and generate the operator deployment manifests
  4. Optionally add additional CRD's using the SDK CLI and repeat steps 2 and 3

## Command Line Interface

To learn more about the SDK CLI, see the [SDK CLI Reference][sdk_cli_ref], or run `operator-sdk [command] -h`.

### Operator capability level

Note that each operator type has a different set of capabilities. When choosing what type to use for your project, it is important to understand the features and limitations of each of the project types and the use cases for your operator.

![operator-capability-level](/operator-capability-level.png)

Find more details about the various levels and the feature requirements for them in the [capability level documentation][capability_levels].

## Kubernetes version compatibility

Each `operator-sdk` release is tested with a specific version of Kubernetes. This version matches
that of [kubernetes/kubernetes][k-k] or [client-go][client-go] that `operator-sdk` depends on directly,
or that generated Operator projects depend on.

In general, client-go's [compatibility matrix][client-go-compat] will determine whether
a particular Kubernetes version is compatible with a particular `operator-sdk` version
or generated Operator project. The following tables contains the canonical way per
binary or project type to look up a Y-axis version to plug into the compatibility matrix.

By binary:

| Binary                  | Lookup strategy               | Kubernetes version    | `client-go` version        |
|-------------------------|-------------------------------|-----------------------|----------------------------|
| `operator-sdk`          | `$ operator-sdk version`      | {{% kube-version %}}  | {{% client-go-version %}}  |
| `ansible-operator`      | `$ ansible-operator version`  | {{% kube-version %}}  | {{% client-go-version %}}  |
| `helm-operator`         | `$ helm-operator version`     | {{% kube-version %}}  | {{% client-go-version %}}  |

By project type (replace `${IMAGE_VERSION}` with base image version in your project `Dockerfile`):

| Project type   | Lookup strategy                           |
|----------------|-------------------------------------------|
| Go             | controller-runtime version (see `go.mod`) |
| Ansible        | `$ docker run --entrypoint ansible-operator quay.io/operator-framework/ansible-operator:${IMAGE_VERSION} version` |
| Helm           | `$ docker run --entrypoint helm-operator quay.io/operator-framework/helm-operator:${IMAGE_VERSION} version` |


[k-k]:https://github.com/kubernetes/kubernetes
[client-go]:https://github.com/kubernetes/client-go
[client-go-compat]:https://github.com/kubernetes/client-go#compatibility-matrix

## OLM version compatibility

Operator SDK officially supports the latest 3 minor versions of OLM present at the time of a given Operator SDK release. These versions of OLM manifests are packaged with the SDK binary in the form of `bindata` to support low-latency installations of OLM with [`operator-sdk olm install`][olm-install-cmd]. Any other version installed with this command may work but is not tested nor officially supported.

Currently, the officially supported OLM Versions are: 0.25.0, 0.26.0, and 0.27.0

## Platform support

Official build architectures for binaries:

| Binary                    | `linux/amd64` | `linux/arm64` |`linux/ppc64le` | `linux/s390x` | `darwin/amd64` | `darwin/arm64` |
|---------------------------|---------------|---------------|----------------|---------------|----------------|----------------|
| `operator-sdk`            | ✓             | ✓             | ✓              | ✓             | ✓              | ✓              |
| `ansible-operator`        | ✓             | ✓             | ✓              | ✓             | ✓              | ✓              |
| `helm-operator`           | ✓             | ✓             | ✓              | ✓             | ✓              | ✓              |

Official build architectures for images:

| Binary                    | `linux/amd64` | `linux/arm64` |`linux/ppc64le` | `linux/s390x` |
|---------------------------|---------------|---------------|----------------|---------------|
| `operator-sdk`            | ✓             | ✓             | ✓              | ✓             |
| `ansible-operator`        | ✓             | ✓             | ✓              | ✓             |
| `helm-operator`           | ✓             | ✓             | ✓              | ✓             |
| `scorecard-test`          | ✓             | ✓             | ✓              | ✓             |
| `scorecard-test-kuttl`    | ✓             | ✓             | ✓              | -             |

Official support for any Windows architecture is not on the roadmap at this time.

## Samples

To explore any operator samples built using the operator-sdk, see the samples in [operator-sdk/testdata/][testdata_samples].

## FAQ

For common Operator SDK related questions, see the [FAQ][faq].

## Contributing

See [CONTRIBUTING][contrib] for details on submitting patches and the contribution workflow.

See the [proposal docs][proposals_docs] and issues for ongoing or planned work.

## Reporting bugs

See [reporting bugs][bug_guide] for details about reporting any issues.

## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file for details.

[ansible-guide]:/docs/building-operators/ansible/quickstart/
[bug_guide]:/docs/contribution-guidelines/reporting-issues/
[capability_levels]: /docs/overview/operator-capabilities/
[contrib]: https://github.com/operator-framework/operator-sdk/blob/master/CONTRIBUTING.MD
[controller_runtime]: https://github.com/kubernetes-sigs/controller-runtime
[faq]: /docs/faqs/
[getting_started]: https://github.com/operator-framework/getting-started/blob/master/README.md
[golang-guide]:/docs/building-operators/golang/quickstart/
[helm-guide]:/docs/building-operators/helm/quickstart/
[install_guide]: /docs/installation/
[license_file]:https://github.com/operator-framework/operator-sdk/blob/master/LICENSE
[of-blog]:https://www.openshift.com/blog/introducing-the-operator-framework
[of-home]: https://github.com/operator-framework
[operator_link]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[proposals_docs]: https://github.com/operator-framework/operator-sdk/tree/master/proposals
[testdata_samples]: https://github.com/operator-framework/operator-sdk/tree/master/testdata
[sdk_cli_ref]: /docs/cli/
[olm-install-cmd]: /docs/cli/operator-sdk_olm_install/
