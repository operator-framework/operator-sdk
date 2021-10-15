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
  -l, --selector string          label selector to determine which tests are run
  -s, --service-account string   Service account to use for tests (default "default")
  -x, --skip-cleanup             Disable resource cleanup after tests are run
  -b, --storage-image string     Storage image to be used by the Scorecard pod (default "docker.io/library/busybox@sha256:c71cb4f7e8ececaffb34037c2637dc86820e4185100e18b4d02d613a9bd772af")
  -t, --test-output string       Test output directory. (default "test-output")
  -u, --untar-image string       Untar image to be used by the Scorecard pod (default "registry.access.redhat.com/ubi8@sha256:910f6bc0b5ae9b555eb91b88d28d568099b060088616eba2867b07ab6ea457c7")
  -w, --wait-time duration       seconds to wait for tests to complete. Example: 35s (default 30s)
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

