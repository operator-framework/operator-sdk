## Unreleased

### Changed

- The SDK now uses logr as the default logger to unify the logging output with the controller-runtime logs. Users can still use a logger of their own choice. See the [logging doc](https://github.com/operator-framework/operator-sdk/blob/master/doc/user/logging.md) on how the SDK initializes and uses logr.
- Add `image` flag to `test local` subcommand ([#768](https://github.com/operator-framework/operator-sdk/pull/768))

### Added

- A new command [`operator-sdk print-deps`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#print-deps) which prints Golang packages and versions expected by the current Operator SDK version. Supplying `--as-file` prints packages and versions in Gopkg.toml format. ([#772](https://github.com/operator-framework/operator-sdk/pull/772))

### Bug fixes

- Make operator-sdk command work with composed GOPATH ([#676](https://github.com/operator-framework/operator-sdk/pull/676))

## v0.1.1

### Bug fixes
- Fix hardcoded CRD version in crd scaffold ([#690](https://github.com/operator-framework/operator-sdk/pull/690))

## v0.1.0

### Changed

- Use [controller runtime](https://github.com/kubernetes-sigs/controller-runtime) library for controller and client APIs
- See [migration guide](https://github.com/operator-framework/operator-sdk/blob/master/doc/migration/v0.1.0-migration-guide.md) to migrate your project to `v0.1.0`

## v0.0.7

### Added

- Service account generation ([#454](https://github.com/operator-framework/operator-sdk/pull/454))
- Leader election ([#530](https://github.com/operator-framework/operator-sdk/pull/530))
- Incluster test support for test framework ([#469](https://github.com/operator-framework/operator-sdk/pull/469))
- Ansible type operator generation support ([#486](https://github.com/operator-framework/operator-sdk/pull/486), [#559](https://github.com/operator-framework/operator-sdk/pull/559))

### Changed

- Moved the rendering of `deploy/operator.yaml` to the `operator-sdk new` command instead of `operator-sdk build`

## v0.0.6

### Added

- Added `operator-sdk up` command to help deploy an operator. Currently supports running an operator locally against an existing cluster e.g `operator-sdk up local --kubeconfig=<path-to-kubeconfig> --namespace=<operator-namespace>`. See `operator-sdk up -h` for help. [#219](https://github.com/operator-framework/operator-sdk/pull/219) [#274](https://github.com/operator-framework/operator-sdk/pull/274)
- Added initial default metrics to be captured and exposed by Prometheus. [#323](https://github.com/operator-framework/operator-sdk/pull/323) exposes the metrics port and [#349](https://github.com/operator-framework/operator-sdk/pull/323) adds the initial default metrics.
- Added initial test framework for operators [#377](https://github.com/operator-framework/operator-sdk/pull/377), [#392](https://github.com/operator-framework/operator-sdk/pull/392), [#393](https://github.com/operator-framework/operator-sdk/pull/393)

### Changed

- All the modules in [`pkg/sdk`](https://github.com/operator-framework/operator-sdk/tree/master/pkg/sdk) have been combined into a single package. `action`, `handler`, `informer` `types` and `query` pkgs have been consolidated into `pkg/sdk`. [#242](https://github.com/operator-framework/operator-sdk/pull/242)
- The SDK exposes the Kubernetes clientset via `k8sclient.GetKubeClient()` #295
- The SDK now vendors the k8s code-generators for an operator instead of using the prebuilt image `gcr.io/coreos-k8s-scale-testing/codegen:1.9.3` [#319](https://github.com/operator-framework/operator-sdk/pull/242)
- The SDK exposes the Kubernetes rest config via `k8sclient.GetKubeConfig()` #338
- Use `time.Duration` instead of `int` for `sdk.Watch` [#427](https://github.com/operator-framework/operator-sdk/pull/427)

### Fixed

- The cache of available clients is being reset every minute for discovery of newely added resources to a cluster. [#280](https://github.com/operator-framework/operator-sdk/pull/280)
