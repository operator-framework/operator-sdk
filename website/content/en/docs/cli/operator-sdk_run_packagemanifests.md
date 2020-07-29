---
title: "operator-sdk run packagemanifests"
---
## operator-sdk run packagemanifests

Deploy an Operator in the package manifests format with OLM

### Synopsis

'run packagemanifests' deploys an Operator's package manifests with OLM. The command's argument
must be set to a valid package manifests root directory, ex. '&lt;project-root&gt;/packagemanifests'.

```
operator-sdk run packagemanifests [flags]
```

### Options

```
  -h, --help                  help for packagemanifests
      --install-mode string   InstallMode to create OperatorGroup with. Format: InstallModeType[=ns1,ns2[, ...]]
      --kubeconfig string     The file path to kubernetes configuration file. Defaults to location specified by $KUBECONFIG, or to default file rules if not set
      --namespace string      The namespace where operator resources are created. It must already exist in the cluster
      --timeout duration      Time to wait for the command to complete before failing (default 2m0s)
      --version string        Packaged version of the operator to deploy
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments

