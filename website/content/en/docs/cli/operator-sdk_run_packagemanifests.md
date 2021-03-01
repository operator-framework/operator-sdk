---
title: "operator-sdk run packagemanifests"
---
## operator-sdk run packagemanifests

Deploy an Operator in the package manifests format with OLM

### Synopsis

'run packagemanifests' deploys an Operator's package manifests with OLM. The command's argument
will default to './packagemanifests' if unset; if set, the argument must be a package manifests root directory,
ex. '&lt;project-root&gt;/packagemanifests'.

```
operator-sdk run packagemanifests [packagemanifests-root-dir] [flags]
```

### Options

```
      --install-mode InstallModeValue   install mode
      --version string                  Packaged version of the operator to deploy
      --timeout duration                install timeout (default 2m0s)
      --kubeconfig string               Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string                If present, namespace scope for this CLI request
  -h, --help                            help for packagemanifests
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments

