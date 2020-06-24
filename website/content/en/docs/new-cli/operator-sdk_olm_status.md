---
title: "operator-sdk olm status"
---
## operator-sdk olm status

Get the status of the Operator Lifecycle Manager installation in your cluster

### Synopsis

Get the status of the Operator Lifecycle Manager installation in your cluster

```
operator-sdk olm status [flags]
```

### Options

```
  -h, --help                   help for status
      --olm-namespace string   namespace where OLM is installed (default "olm")
      --timeout duration       time to wait for the command to complete before failing (default 2m0s)
      --version string         version of OLM installed on cluster; if unsetoperator-sdk attempts to auto-discover the version
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk olm](../operator-sdk_olm)	 - Manage the Operator Lifecycle Manager installation in your cluster

