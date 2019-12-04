## operator-sdk up local

Launches the operator locally

### Synopsis

The operator-sdk up local command launches the operator on the local machine
by building the operator binary with the ability to access a
kubernetes cluster using a kubeconfig file.


```
operator-sdk up local [flags]
```

### Options

```
      --enable-delve            Start the operator using the delve debugger
      --go-ldflags string       Set Go linker options
  -h, --help                    help for local
      --kubeconfig string       The file path to kubernetes configuration file; defaults to location specified by $KUBECONFIG with a fallback to $HOME/.kube/config if not set
      --namespace string        The namespace where the operator watches for changes.
      --operator-flags string   The flags that the operator needs. Example: "--flag1 value1 --flag2=value2"
```

### SEE ALSO

* [operator-sdk up](operator-sdk_up.md)	 - Launches the operator

