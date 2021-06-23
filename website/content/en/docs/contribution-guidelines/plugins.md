---
title: Plugins
linkTitle: Plugins
weight: 7
---

## Overview

SDK uses the project [Kubebuilder][kubebuilder] as a library for the CLI and plugins features. For further information see the [Plugins][kb-plugins-doc] document. To better understand its motivations, check the design documentation [Integrating Kubebuilder and Operator SDK][kb-int-sdk].

## SDK Language-based plugins

All language-based operator projects provided by SDK (Ansible/Helm/Golang) follow the Kubebuilder standard and have a common base. The common base is generated using [kustomize][kustomize] which is a project maintained and adopted by Kubernetes community to help work with the Kubernertes manifests (YAML files) that are used to configure the resources on the clusters.

The specific files for each language are scaffolded for the language's plugins. The Operator SDK CLI tool can provide custom plugins and helpers which are common and useful for all language types. To check the common base, you can also run:

```sh
operator-sdk init --plugins=kustomize
```

Also, see the topic [Language-based Plugins][kb-language-plugins] to understand how it works.

### Common scaffolds 

Following the default common scaffolds for the projects which are built with SDK.

| File/Directory | Description | 
| ------ | ----- |
|  Dockerfile |  Defines the operator(manager) image |  
|  Makefile |  Provides the helpers and options for the users. (e.g. `make bundle` which generate/update the OLM [bundle][bundle] manifests ) |  
|  PROJECT |  Project configuration. Stores the data used to do the scaffolds. For further information see [Project Config][kb-project] | 
|  bundle.Dockerfile | Docker image which is used to provide the helpers to integrate the project with OLM. (e.g. [operator-sdk run ./bundle][sdk-cli-run-bundle]) | 
|  config/ |  Directory which has all [kustomize's][kustomize] manifest to configure and test the project | 

You can check the [Project Layout][project-layout] to better understand the files and directories scaffolded by SDK CLI, which may be common for each language-type.

## Custom Plugins

By default plugins are used by the Operator SDK to provide the following features:

- [manifests.sdk.operatorframework.io][plugin-manifest]: perform the required scaffolds to provide the helpers to allow the projects to be integrated with OLM. 
- [scorecard.sdk.operatorframework.io][plugin-scorecard]: perform the required scaffolds to provide the [Scorecard][scorecard] feature.

### Optional/custom plugins

Users can also use custom plugins when they execute the SDK CLI sub-commands as a helper which includes the following options:

```sh
operator-sdk create api --plugins="go/v3,declarative"
```

The above example will scaffold custom code in the controllers after an API is created to allow the users to develop solutions using the [kubebuilder declarative pattern][kubebuilder-declarative-pattern]. (e.g [default scaffold][default-scaffold] versus [example][kubebuilder-declarative-pattern-example]).

Note that custom plugins are called via the init sub-command to work as global plugins, and will be added in the `layout` field of the PROJECT file. Any sub-command executed will then also be called.

## Plugins Vision

Contributors are able to create their own plugin(s) using the same standards and approach described by this document. Following them facilitates in-tree (in other words, built with the `operator-sdk` binary) addition of such community-driven plugins to the Operator SDK project by making review easier and code maintainable. Currently, the SDK cannot discover and use plugins that are not in-tree. However, out-of-tree plugins have been discussed in [kubebuilder/issues/1378][kb-issue].

## How to create your own plugins

If you are looking to develop similar solutions to allow users for example to create projects using other languages or that could work as helpers for the projects built with SDK/Kubebuilder projects, then see the [Creating your own plugins][create-your-own-plugins] to see how you can benefit and apply this approach.

[kb-plugins-doc]: https://master.book.kubebuilder.io/plugins/plugins.html
[kb-int-sdk]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
[kb-language-plugins]:https://master.book.kubebuilder.io/plugins/creating-plugins.html#language-based-plugins
[kustomize]: https://github.com/kubernetes-sigs/kustomize
[bundle]: https://github.com/operator-framework/operator-registry/blob/v1.16.1/docs/design/operator-bundle.md#operator-bundle
[kb-project]: https://master.book.kubebuilder.io/reference/project-config.html
[sdk-cli-run-bundle]: /docs/cli/operator-sdk_run
[project-layout]: /docs/overview/project-layout
[plugin-manifest]: https://github.com/operator-framework/operator-sdk/tree/master/internal/plugins/manifests/v2
[plugin-scorecard]: https://github.com/operator-framework/operator-sdk/tree/master/internal/plugins/scorecard/v2
[kubebuilder-declarative-pattern]: https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern
[kubebuilder-declarative-pattern-example]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/project-v3/controllers/firstmate_controller.go
[default-scaffold]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/project-v3/controllers/admiral_controller.go
[kb-issue]: https://github.com/kubernetes-sigs/kubebuilder/issues/1378
[create-your-own-plugins]: https://master.book.kubebuilder.io/plugins/creating-plugins.html
[scorecard]: /docs/testing-operators/scorecard/
[kubebuilder]: https://github.com/kubernetes-sigs/kubebuilder 
