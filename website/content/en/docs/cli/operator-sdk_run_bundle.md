---
title: "operator-sdk run bundle"
---
## operator-sdk run bundle

Deploy an Operator in the bundle format with OLM

### Synopsis

Deploy an Operator in the bundle format with OLM

```
operator-sdk run bundle <bundle-image> [flags]
```

### Options

```
      --index-image string              index image in which to inject bundle (default "quay.io/operator-framework/upstream-opm-builder:latest")
      --install-mode InstallModeValue   install mode
      --timeout duration                install timeout (default 2m0s)
      --kubeconfig string               Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string                If present, namespace scope for this CLI request
  -h, --help                            help for bundle
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments

