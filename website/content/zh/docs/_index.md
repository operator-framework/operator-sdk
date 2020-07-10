---
title: Operator SDK 文档
linkTitle: 文档
menu:
  main:
    weight: 2
---
<!--
title: Operator SDK 
linkTitle: Documentation
menu:
  main:
    weight: 2
-->

<!--
## Overview

This project is a component of the [Operator Framework][of-home], an open source toolkit to manage Kubernetes native applications, called Operators, in an effective, automated, and scalable way. Read more in the [introduction blog post][of-blog].

[Operators][operator_link] make it easy to manage complex stateful applications on top of Kubernetes. However writing an operator today can be difficult because of challenges such as using low level APIs, writing boilerplate, and a lack of modularity which leads to duplication.

The Operator SDK is a framework that uses the [controller-runtime][controller_runtime] library to make writing operators easier by providing:

  - High level APIs and abstractions to write the operational logic more intuitively
  - Tools for scaffolding and code generation to bootstrap a new project fast
  - Extensions to cover common operator use cases
-->
## 概述
本项目是 [Operator Framework][of-home]
的一个组件，是一个以高效、自动化且可扩缩的方式来管理称为 Operators 的
Kubernetes 原生应用的工具。可参考[介绍博文][of-blog]进一步阅读。

[Operators][operator_link] 使得用户能够方便地在 Kubernetes
之上管理复杂的有状态应用。不过，目前编写 Operator
可能是一项很困难的工作，因为其中需要使用底层的
API、需要使用预制代码、以及因为模块化不好而导致重复劳动等等。

Operator SDK 是一个使用 [controller-runtime][controller_runtime]
库的框架，它提供以下特性以使得编写 Operator 的工作变得容易：

- 用来编写更为直观清晰的操作逻辑的高层 API 和抽象概念
- 用来快速启动新项目的架构布局和代码生成的工具
- 能够覆盖 Operator 常见使用场景的扩展机制

<!--
## Workflow

SDK provides workflows to develop operators in Go, Ansible, or Helm.

The following workflow is for a new [Golang operator][golang-guide]:

  1. Create a new operator project using the SDK Command Line Interface(CLI)
  2. Define new resource APIs by adding Custom Resource Definitions(CRD)
  3. Define Controllers to watch and reconcile resources
  4. Write the reconciling logic for your Controller using the SDK and controller-runtime APIs
  5. Use the SDK CLI to build and generate the operator deployment manifests
-->
## 工作流程

SDK 为使用 Go、Ansible 或 Helm 开发 Operator 提供以下工作流程。

下面是一个用于新的 [Go 语言 Operator][golang-guide] 项目的流程：

1. 使用 SDK 的命令行界面（CLI）创建一个新的 Operator 项目
2. 通过添加定制资源定义（CRD）定义新的资源 API
3. 定义用来监测（Watch）和调解（Reconcile）资源的 Controller
4. 使用 SDK 和 controller-runtime 的 API，为你的 Controller 编写调解逻辑
5. 使用 SDK 的 CLI 来构建和生成 Operator 部署用的资产和清单

<!--
The following workflow is for a new Ansible operator:

  1. Create a new operator project using the SDK Command Line Interface(CLI)
  2. Write the reconciling logic for your object using ansible playbooks and roles
  3. Use the SDK CLI to build and generate the operator deployment manifests
  4. Optionally add additional CRD's using the SDK CLI and repeat steps 2 and 3
-->
下面是一个用于新的 [Ansible Operator][ansible-guide] 项目的流程：

1. 使用 SDK 的命令行界面（CLI）创建一个新的 Operator 项目
2. 使用 Ansible 的 Playbook 和 Role 来为对象编写调解逻辑
3. 使用 SDK 的 CLI 来构建和生成 Operator 部署用的资产和清单
4. 作为可选步骤，使用 SDK 的 CLI 添加额外的 CRD，并重复步骤 2 和 3

<!--
The following workflow is for a new [Helm operator][helm-guide]:

  1. Create a new operator project using the SDK Command Line Interface(CLI)
  2. Create a new (or add your existing) Helm chart for use by the operator's reconciling logic
  3. Use the SDK CLI to build and generate the operator deployment manifests
  4. Optionally add additional CRD's using the SDK CLI and repeat steps 2 and 3
-->
下面是一个用于新的 [Helm Operator][ansible-guide] 项目的流程：

1. 使用 SDK 的命令行界面（CLI）创建一个新的 Operator 项目
2. 创建新的 Helm Chart 或者添加已有 Helm Chart 以供 Operator 的调解逻辑使用
3. 使用 SDK 的 CLI 来构建和生成 Operator 部署用的资产和清单
4. 作为可选步骤，使用 SDK 的 CLI 添加额外的 CRD，并重复步骤 2 和 3
<!--
## Command Line Interface

To learn more about the SDK CLI, see the [SDK CLI Reference][sdk_cli_ref], or run `operator-sdk [command] -h`.
-->
## 命令行界面 {#command-line-interface}

要进一步学习 SDK 的 CLI，请参阅 [SDK CLI 参考][sdk_cli_ref]，
或者运行 `operator-sdk [command] -h`。

<!--
### Operator capability level

Note that each operator type has a different set of capabilities. When choosing what type to use for your project, it is important to understand the features and limitations of each of the project types and the use cases for your operator.

![operator-capability-level](/operator-capability-level.png)

Find more details about the various levels and the feature requirements for them in the [capability level documentation][capability_levels].
-->
### Operator 能力级别  {#operator-capability-level}

注意每种 Operator 类型都有不同的能力。
在选择为你的项目使用哪种类型时，很重要的一点是要理解每种项目类型
的功能特性和限制，以及你的 Operator 的使用场景。

![operator-capability-level](/operator-capability-level.png)

关于不同级别的更多细节及其功能特性需求，请参考[能力级别文档][capability_levels]。

<!--
## Samples

To explore any operator samples built using the operator-sdk, see the [operator-sdk-samples][samples].

## FAQ

For common Operator SDK related questions, see the [FAQ][faq].
-->
## 示例 {#samples}

要浏览使用 operator-sdk 所构建的 Operater 示例，请参考
[operator-sdk-samples][samples] 项目。

## 常见问题  {#faq}

关于一些与 Operator SDK 相关的常见问题，请参阅[常见问题][faq]小节。

<!--
## Contributing

See [CONTRIBUTING][contrib] for details on submitting patches and the contribution workflow.

See the [proposal docs][proposals_docs] and issues for ongoing or planned work.

## Reporting bugs

See [reporting bugs][bug_guide] for details about reporting any issues.
-->
## 贡献  {#contributing}

请参阅[贡献说明][contrib]文档，了解提交补丁以及贡献流程的细节。

关于正在进行或计划开展的工作，请参见[提案（Proposals）文档][proposals_docs]
和 Issues 列表。

## 报告问题  {#reporting-bugs}

关于如何报告所遇到的任何问题，请参阅[报告问题][bug_guide]了解详细信息。

<!--
## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file for details.
-->

## 授权许可

Operator SDK 是采用 Apache 2.0 授权许可的。详情可参见 [LICENSE][license_file] 文件。

[ansible-guide]:/docs/ansible/quickstart/
[bug_guide]:/docs/contribution-guidelines/reporting-issues/
[capability_levels]: /docs/operator-capabilities/
[contrib]: https://github.com/operator-framework/operator-sdk/blob/master/CONTRIBUTING.MD
[controller_runtime]: https://github.com/kubernetes-sigs/controller-runtime
[faq]: /docs/faq/
[getting_started]: https://github.com/operator-framework/getting-started/blob/master/README.md
[golang-guide]:/docs/golang/quickstart/
[helm-guide]:/docs/helm/quickstart/
[install_guide]: /docs/install-operator-sdk/
[license_file]:https://github.com/operator-framework/operator-sdk/blob/master/LICENSE
[of-blog]: https://coreos.com/blog/introducing-operator-framework
[of-home]: https://github.com/operator-framework
[operator_link]: https://coreos.com/operators/
[proposals_docs]: https://github.com/operator-framework/operator-sdk/tree/master/proposals
[samples]: https://github.com/operator-framework/operator-sdk-samples
[sdk_cli_ref]: /docs/cli/

<!-- 2020-07-10 4fa12a42 -->
