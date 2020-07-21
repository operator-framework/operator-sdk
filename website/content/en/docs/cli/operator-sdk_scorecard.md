---
title: "operator-sdk scorecard"
---
## operator-sdk scorecard

Runs scorecard

### Synopsis

Has flags to configure dsl, bundle, and selector. This command takes
one argument, either a bundle image or directory containing manifests and metadata.
If the argument holds an image tag, it must be present remotely.

```
operator-sdk scorecard [flags]
```

### Options

```
  -c, --config string            path to scorecard config file
  -h, --help                     help for scorecard
      --kubeconfig string        kubeconfig path
  -L, --list                     Option to enable listing which tests are run
  -n, --namespace string         namespace to run the test images in
  -o, --output string            Output format for results. Valid values: text, json (default "text")
  -l, --selector string          label selector to determine which tests are run
  -s, --service-account string   Service account to use for tests (default "default")
  -x, --skip-cleanup             Disable resource cleanup after tests are run
  -w, --wait-time duration       seconds to wait for tests to complete. Example: 35s (default 30s)
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - Development kit for building Kubernetes extensions and tools.

