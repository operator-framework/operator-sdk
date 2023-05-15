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
  -o, --output string            Output format for results. Valid values: text, json, xunit (default "text")
      --pod-security string      option to run scorecard with legacy pod security context (default "legacy")
  -l, --selector string          label selector to determine which tests are run
  -s, --service-account string   Service account to use for tests (default "default")
  -x, --skip-cleanup             Disable resource cleanup after tests are run
  -b, --storage-image string     Storage image to be used by the Scorecard pod (default "quay.io/operator-framework/scorecard-storage@sha256:5f9640f6eb6a6976676f2936b9eb4cd7170c5eebbc7536cc2891ec6cba74f0dd")
  -t, --test-output string       Test output directory. (default "test-output")
  -u, --untar-image string       Untar image to be used by the Scorecard pod (default "quay.io/operator-framework/scorecard-untar@sha256:e7b0222764d1d1c16614009f38e7fe9bc643ef9e2b88559712ec3fd439b796c8")
  -w, --wait-time duration       seconds to wait for tests to complete. Example: 35s (default 30s)
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

