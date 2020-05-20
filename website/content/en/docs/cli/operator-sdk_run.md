---
title: "operator-sdk run"
---
## operator-sdk run

Run an Operator in a variety of environments

### Synopsis

This command will run or deploy your Operator in two different modes: locally
and using OLM. These modes are controlled by setting --local and --olm run mode
flags. Each run mode has a separate set of flags that configure 'run' for that
mode. Run 'operator-sdk run --help' for more information on these flags.

Read more about the --olm run mode and configuration options here:
https://sdk.operatorframework.io/docs/olm-integration/cli-overview


```
operator-sdk run [flags]
```

### Options

```
      --kubeconfig string        The file path to kubernetes configuration file. Defaults to location specified by $KUBECONFIG, or to default file rules if not set
      --local                    The operator will be run locally by building the operator binary with the ability to access a kubernetes cluster using a kubeconfig file. Cannot be set with another run-type flag.
      --watch-namespace string   [local only] The namespace where the operator watches for changes. Set "" for AllNamespaces, set "ns1,ns2" for MultiNamespace
      --operator-flags string    [local only] The flags that the operator needs. Example: "--flag1 value1 --flag2=value2"
      --go-ldflags string        [local only] Set Go linker options
      --enable-delve             [local only] Start the operator using the delve debugger
  -h, --help                     help for run
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - An SDK for building operators with ease
* [operator-sdk run packagemanifests](../operator-sdk_run_packagemanifests)	 - Run an Operator organized in the package manifests format with OLM

