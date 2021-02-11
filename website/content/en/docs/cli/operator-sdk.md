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
    operator-sdk init --project-version=<PROJECT VERSION> --plugins=<PLUGIN KEYS>

<PLUGIN KEYS> is a comma-separated list of plugin keys from the following table
and <PROJECT VERSION> a supported project version for these plugins.

                         Plugin keys | Supported project versions
-------------------------------------+----------------------------
 ansible.sdk.operatorframework.io/v1 |                          3
                go.kubebuilder.io/v2 |                       2, 3
                go.kubebuilder.io/v3 |                          3
    helm.sdk.operatorframework.io/v1 |                          3

For more specific help for the init command of a certain plugins and project version
configuration please run:
    operator-sdk init --help --project-version=<PROJECT VERSION> --plugins=<PLUGIN KEYS>

Default project version: 3
Default plugin keys: "go.kubebuilder.io/v3"

After the project has been initialized, run
    operator-sdk --help
to obtain further info about available commands.
```

### Options

```
  -h, --help                     help for operator-sdk
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata
* [operator-sdk cleanup](../operator-sdk_cleanup)	 - Clean up an Operator deployed with the 'run' subcommand
* [operator-sdk completion](../operator-sdk_completion)	 - Generators for shell completions
* [operator-sdk create](../operator-sdk_create)	 - Scaffold a Kubernetes API or webhook
* [operator-sdk edit](../operator-sdk_edit)	 - This command will edit the project configuration
* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator
* [operator-sdk init](../operator-sdk_init)	 - Initialize a new project
* [operator-sdk olm](../operator-sdk_olm)	 - Manage the Operator Lifecycle Manager installation in your cluster
* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments
* [operator-sdk scorecard](../operator-sdk_scorecard)	 - Runs scorecard
* [operator-sdk version](../operator-sdk_version)	 - Print the operator-sdk version

