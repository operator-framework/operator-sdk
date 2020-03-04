## operator-sdk test local

Run End-To-End tests locally

### Synopsis

Run End-To-End tests locally

```
operator-sdk test local <path to tests directory> [flags]
```

### Options

```
      --debug                         Enable debug-level logging
      --global-manifest string        Path to manifest for Global resources (e.g. CRD manifests)
      --go-test-flags string          Additional flags to pass to go test
  -h, --help                          help for local
      --image string                  Use a different operator image from the one specified in the namespaced manifest
      --kubeconfig string             Kubeconfig path
      --local-operator-flags string   The flags that the operator needs (while using --up-local). Example: "--flag1 value1 --flag2=value2"
      --molecule-test-flags string    Additional flags to pass to molecule test
      --namespaced-manifest string    Path to manifest for per-test, namespaced resources (e.g. RBAC and Operator manifest)
      --no-setup                      Disable test resource creation
      --operator-namespace string     If non-empty, single namespace to run tests in
      --skip-cleanup-error            If set as true, the cleanup function responsible to remove all artifacts will be skipped if an error is faced.
      --up-local                      Enable running operator locally with go run instead of as an image in the cluster
```

### SEE ALSO

* [operator-sdk test](operator-sdk_test.md)	 - Tests the operator

