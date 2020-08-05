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
  -h, --help                help for cleanup
      --kubeconfig string   Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string    If present, namespace scope for this CLI request
      --timeout duration    Time to wait for the command to complete before failing (default 2m0s)
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - Development kit for building Kubernetes extensions and tools.

