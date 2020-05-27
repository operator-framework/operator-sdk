---
title: "operator-sdk run local"
---
## operator-sdk run local

Run an Operator locally

### Synopsis

This command will run your Operator locally by building the operator binary
with the ability to access a kubernetes cluster using a kubeconfig file

```
operator-sdk run local [flags]
```

### Options

```
      --enable-delve             Start the operator using the delve debugger
      --go-ldflags string        Set Go linker options
  -h, --help                     help for local
      --kubeconfig string        The file path to kubernetes configuration file. Defaults to location specified by $KUBECONFIG, or to default file rules if not set
      --operator-flags string    The flags that the operator needs. Example: "--flag1 value1 --flag2=value2"
      --watch-namespace string   The namespace where the operator watches for changes. Set "" for AllNamespaces, set "ns1,ns2" for MultiNamespace
```

### SEE ALSO

* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments

