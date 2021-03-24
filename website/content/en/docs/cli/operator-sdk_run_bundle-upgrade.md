---
title: "operator-sdk run bundle-upgrade"
---
## operator-sdk run bundle-upgrade

Upgrade an Operator previously installed in the bundle format with OLM

```
operator-sdk run bundle-upgrade <bundle-image> [flags]
```

### Options

```
  -h, --help                     help for bundle-upgrade
      --kubeconfig string        Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string         If present, namespace scope for this CLI request
      --secret-name string       Name of image pull secret required to pull bundle images. This secret must be in docker config format, and tied to the namespace, and optionally service account, that this command is configured to run in
      --service-account string   Service account name to bind registry objects to. If unset, the default service account is used. This value does not override the operator's service account
      --timeout duration         Duration to wait for the command to complete before failing (default 2m0s)
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments

