## [Unreleased]

### Added

- Added `operator-sdk up` command to help deploy an operator. Currently supports running an operator locally against an existing cluster e.g `operator-sdk up local --kubeconfig=<path-to-kubeconfig> --namespace=<operator-namespace>`. See `operator-sdk up -h` for help. [#219](https://github.com/operator-framework/operator-sdk/pull/219) [#274](https://github.com/operator-framework/operator-sdk/pull/274)
- Added initial default metrics to be captured and exposed by Prometheus. [#323](https://github.com/operator-framework/operator-sdk/pull/323) exposes the metrics port and [#349](https://github.com/operator-framework/operator-sdk/pull/323) adds the initial default metrics.

### Changed

- All the modules in [`pkg/sdk`](https://github.com/operator-framework/operator-sdk/tree/master/pkg/sdk) have been combined into a single package. `action`, `handler`, `informer` `types` and `query` pkgs have been consolidated into `pkg/sdk`. [#242](https://github.com/operator-framework/operator-sdk/pull/242)
- The SDK exposes the Kubernetes clientset via `k8sclient.GetKubeClient()` #295
- The SDK now vendors the k8s code-generators for an operator instead of using the prebuilt image `gcr.io/coreos-k8s-scale-testing/codegen:1.9.3` [#319](https://github.com/operator-framework/operator-sdk/pull/242)
- The SDK exposes the Kubernetes rest config via `k8sclient.GetKubeConfig()` #338

### Removed

### Fixed

- The cache of available clients is being reset every minute for discovery of newely added resources to a cluster. [#280](https://github.com/operator-framework/operator-sdk/pull/280)

### Deprecated

### Security
