---
title: "operator-sdk run"
---
## operator-sdk run

Run an Operator in a variety of environments

### Synopsis

This command has subcommands that will deploy your Operator with OLM.
Currently only the package manifests format is supported via the 'packagemanifests' subcommand.

### Options

```
  -h, --help                help for run
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string    If present, namespace scope for this CLI request
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - Development kit for building Kubernetes extensions and tools.
* [operator-sdk run packagemanifests](../operator-sdk_run_packagemanifests)	 - Deploy an Operator in the package manifests format with OLM

