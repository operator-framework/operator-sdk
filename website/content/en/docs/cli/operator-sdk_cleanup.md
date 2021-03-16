---
title: "operator-sdk cleanup"
---
## operator-sdk cleanup

Clean up an Operator deployed with the 'run' subcommand

### Synopsis

This command has subcommands that will destroy an Operator deployed with OLM.

```
operator-sdk cleanup <operatorPackageName> [flags]
```

### Options

```
      --delete-all               If set to true, all other delete options will be enabled (default true)
      --delete-crds              If set to false, owned CRDs and CRs will not be deleted (default true)
      --delete-operator-groups   If set to false, operator groups will not be deleted (default true)
  -h, --help                     help for cleanup
      --kubeconfig string        Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string         If present, namespace scope for this CLI request
      --timeout duration         Time to wait for the command to complete before failing (default 2m0s)
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

