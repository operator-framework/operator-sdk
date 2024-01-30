---
title: "operator-sdk"
---
## operator-sdk



### Synopsis

CLI tool for building Kubernetes extensions and tools.


```
operator-sdk [flags]
```

### Examples

```
The first step is to initialize your project:
    operator-sdk init [--plugins=<PLUGIN KEYS> [--project-version=<PROJECT VERSION>]]

<PLUGIN KEYS> is a comma-separated list of plugin keys from the following table
and <PROJECT VERSION> a supported project version for these plugins.

                                   Plugin keys | Supported project versions
-----------------------------------------------+----------------------------
           ansible.sdk.operatorframework.io/v1 |                          3
              declarative.go.kubebuilder.io/v1 |                       2, 3
       deploy-image.go.kubebuilder.io/v1-alpha |                          3
                          go.kubebuilder.io/v2 |                       2, 3
                          go.kubebuilder.io/v3 |                          3
                          go.kubebuilder.io/v4 |                          3
               grafana.kubebuilder.io/v1-alpha |                          3
              helm.sdk.operatorframework.io/v1 |                          3
 hybrid.helm.sdk.operatorframework.io/v1-alpha |                          3
            quarkus.javaoperatorsdk.io/v1-beta |                          3

For more specific help for the init command of a certain plugins and project version
configuration please run:
    operator-sdk init --help --plugins=<PLUGIN KEYS> [--project-version=<PROJECT VERSION>]

Default plugin keys: "go.kubebuilder.io/v4"
Default project version: "3"

```

### Options

```
  -h, --help                     help for operator-sdk
      --plugins strings          plugin keys to be used for this subcommand execution
      --project-version string   project version (default "3")
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Alpha-stage subcommands
* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata
* [operator-sdk cleanup](../operator-sdk_cleanup)	 - Clean up an Operator deployed with the 'run' subcommand
* [operator-sdk completion](../operator-sdk_completion)	 - Load completions for the specified shell
* [operator-sdk create](../operator-sdk_create)	 - Scaffold a Kubernetes API or webhook
* [operator-sdk edit](../operator-sdk_edit)	 - Update the project configuration
* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator
* [operator-sdk init](../operator-sdk_init)	 - Initialize a new project
* [operator-sdk olm](../operator-sdk_olm)	 - Manage the Operator Lifecycle Manager installation in your cluster
* [operator-sdk pkgman-to-bundle](../operator-sdk_pkgman-to-bundle)	 - Migrates packagemanifests to bundles
* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments
* [operator-sdk scorecard](../operator-sdk_scorecard)	 - Runs scorecard
* [operator-sdk version](../operator-sdk_version)	 - Print the operator-sdk version

