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
      --timeout duration    Time to wait for the command to complete before failing (default 2m0s)
      --delete-crds         If set to false, owned CRDs and CRs will not be deleted (default true)
      --delete-all          If set to true, it will enable all the delete flags (default true)
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string    If present, namespace scope for this CLI request
  -h, --help                help for cleanup
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

