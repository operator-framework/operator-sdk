---
title: "operator-sdk olm uninstall"
---
## operator-sdk olm uninstall

Uninstall Operator Lifecycle Manager from your cluster

```
operator-sdk olm uninstall [flags]
```

### Options

```
  -h, --help                   help for uninstall
      --olm-namespace string   namespace from where OLM is to be uninstalled. (default "olm")
      --timeout duration       time to wait for the command to complete before failing (default 2m0s)
      --version string         version of OLM resources to uninstall.
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk olm](../operator-sdk_olm)	 - Manage the Operator Lifecycle Manager installation in your cluster.

Operator SDK officially supports the following OLM versions: unknown.
Any other version installed with this command may work but is not officially tested.

