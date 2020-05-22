---
title: "operator-sdk alpha scorecard"
---
## operator-sdk alpha scorecard

Runs scorecard

### Synopsis

Has flags to configure dsl, bundle, and selector.

```
operator-sdk alpha scorecard [flags]
```

### Options

```
      --bundle string            path to the operator bundle contents on disk
  -c, --config string            path to scorecard config file
  -h, --help                     help for scorecard
      --kubeconfig string        kubeconfig path
  -L, --list                     Option to enable listing which tests are run
  -n, --namespace string         namespace to run the test images in (default "default")
  -o, --output string            Output format for results.  Valid values: text, json (default "text")
  -l, --selector string          label selector to determine which tests are run
  -s, --service-account string   Service account to use for tests (default "default")
  -x, --skip-cleanup             Disable resource cleanup after tests are run
  -w, --wait-time duration       seconds to wait for tests to complete. Example: 35s (default 30s)
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Run an alpha subcommand

