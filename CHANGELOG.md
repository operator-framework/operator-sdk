## v0.7.0

### Added

- New optional flag `--header-file` for commands [`operator-sdk generate k8s`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#k8s) and [`operator-sdk add api`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#api) to supply a boilerplate header file for generated code. ([#1239](https://github.com/operator-framework/operator-sdk/pull/1239))

### Changed

- Updated the helm-operator to store release state in kubernetes secrets in the same namespace of the custom resource that defines the release. ([#1102](https://github.com/operator-framework/operator-sdk/pull/1102))
  - **WARNING**: Users with active CRs and releases who are upgrading their helm-based operator should not skip this version. Future versions will not seamlessly transition release state to the persistent backend, and will instead uninstall and reinstall all managed releases.
- Change `namespace-manifest` flag in scorecard subcommand to `namespaced-manifest` to match other subcommands
- Subcommands of [`operator-sdk generate`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#generate) are now verbose by default. ([#1271](https://github.com/operator-framework/operator-sdk/pull/1271))
- [`operator-sdk olm-catalog gen-csv`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#gen-csv) parses Custom Resource manifests from `deploy/crds` or a custom path specified in `csv-config.yaml`, encodes them in a JSON array, and sets the CSV's [`metadata.annotations.alm-examples`](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#crd-templates) field to that JSON. ([#1116](https://github.com/operator-framework/operator-sdk/pull/1116))

### Deprecated

### Removed

### Bug Fixes

- Fixed an issue that caused `operator-sdk new --type=helm` to fail for charts that have template files in nested template directories. ([#1235](https://github.com/operator-framework/operator-sdk/pull/1235))
- Fix bug in the YAML scanner used by `operator-sdk test` and `operator-sdk scorecard` that could result in a panic if a manifest file started with `---` ([#1258](https://github.com/operator-framework/operator-sdk/pull/1258))

## v0.6.0

### Added

- New flags for [`operator-sdk new --type=helm`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#new), which can be used to populate the project with an existing chart. ([#949](https://github.com/operator-framework/operator-sdk/pull/949))
- Command [`operator-sdk olm-catalog`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#olm-catalog) flag `--update-crds` optionally copies CRD's from `deploy/crds` when creating a new CSV or updating an existing CSV, and `--from-version` uses another versioned CSV manifest as a base for a new CSV version. ([#1016](https://github.com/operator-framework/operator-sdk/pull/1016))
- New flag `--olm-deployed` to direct the [`scorecard`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#scorecard) command to only use the CSV at `--csv-path` for manifest data, except for those provided to `--cr-manifest`. ([#1044](https://github.com/operator-framework/operator-sdk/pull/1044))
- Command [`version`](https://github.com/operator-framework/operator-sdk/pull/1171) prints the version of operator-sdk. ([#1171](https://github.com/operator-framework/operator-sdk/pull/1171))

### Changed

- Changed the Go, Helm, and Scorecard base images to `registry.access.redhat.com/ubi7-dev-preview/ubi-minimal:7.6` ([#1142](https://github.com/operator-framework/operator-sdk/pull/1142))
- CSV manifest are now versioned according to the `operator-registry` [manifest format](https://github.com/operator-framework/operator-registry#manifest-format). See issue [#900](https://github.com/operator-framework/operator-sdk/issues/900) for more details. ([#1016](https://github.com/operator-framework/operator-sdk/pull/1016))
- Unexported `CleanupNoT` function from `pkg/test`, as it is only intended to be used internally ([#1167](https://github.com/operator-framework/operator-sdk/pull/1167))

### Bug Fixes

- Fix issue where running `operator-sdk test local --up-local` would sometimes leave a running process in the background after exit ([#1089](https://github.com/operator-framework/operator-sdk/pull/1020))

## v0.5.0

### Added

- Updated the Kubernetes dependencies to `1.13.1` ([#1020](https://github.com/operator-framework/operator-sdk/pull/1020))
- Updated the controller-runtime version to `v0.1.10`. See the [controller-runtime `v0.1.10` release notes](https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.1.10) for new features and bug fixes. ([#1020](https://github.com/operator-framework/operator-sdk/pull/1020))
- By default the controller-runtime metrics are exposed on port 8383. This is done as part of the scaffold in the main.go file, the port can be adjusted by modifying the `metricsPort` variable. [#786](https://github.com/operator-framework/operator-sdk/pull/786)
- A new command [`operator-sdk olm-catalog`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#olm-catalog) to be used as a parent for SDK subcommands generating code related to Operator Lifecycle Manager (OLM) Catalog integration, and subcommand [`operator-sdk olm-catalog gen-csv`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#gen-csv) which generates a Cluster Service Version for an operator so the OLM can deploy the operator in a cluster. ([#673](https://github.com/operator-framework/operator-sdk/pull/673))
- Helm-based operators have leader election turned on by default. When upgrading, add environment variable `POD_NAME` to your operator's Deployment using the Kubernetes downward API. To see an example, run `operator-sdk new --type=helm ...` and see file `deploy/operator.yaml`. [#1000](https://github.com/operator-framework/operator-sdk/pull/1000)
- A new command [`operator-sdk generate openapi`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#openapi) which generates OpenAPIv3 validation specs in Go and in CRD manifests as YAML. ([#869](https://github.com/operator-framework/operator-sdk/pull/869))
- The `operator-sdk add api` command now generates OpenAPIv3 validation specs in Go for that API, and in all CRD manifests as YAML.

### Changed

- In new Helm operator projects, the scaffolded CR `spec` field now contains the default values.yaml from the generated chart. ([#967](https://github.com/operator-framework/operator-sdk/pull/967))

### Deprecated

### Removed

### Bug Fixes

## v0.4.1

### Bug Fixes

- Make `up local` subcommand respect `KUBECONFIG` env var ([#996](https://github.com/operator-framework/operator-sdk/pull/996))
- Make `up local` subcommand use default namespace set in kubeconfig instead of hardcoded `default` and also add ability to watch all namespaces for ansible and helm type operators ([#996](https://github.com/operator-framework/operator-sdk/pull/996))
- Added k8s_status modules back to generation ([#972](https://github.com/operator-framework/operator-sdk/pull/972))
- Update checks for gvk registration to cover all cases for ansible ([#973](https://github.com/operator-framework/operator-sdk/pull/973) & [#1019](https://github.com/operator-framework/operator-sdk/pull/1019))
- Update reconciler for ansible and helm to use the cache rather than the API client. ([#1022](https://github.com/operator-framework/operator-sdk/pull/1022) & [#1048](https://github.com/operator-framework/operator-sdk/pull/1048) & [#1054](https://github.com/operator-framework/operator-sdk/pull/1054))
- Update reconciler to will update the status everytime for ansible ([#1066](https://github.com/operator-framework/operator-sdk/pull/1066))
- Update ansible proxy to recover dependent watches when pod is killed ([#1067](https://github.com/operator-framework/operator-sdk/pull/1067))
- Update ansible proxy to handle watching cluster scoped dependent watches ([#1031](https://github.com/operator-framework/operator-sdk/pull/1031))

## v0.4.0

### Added

- A new command [`operator-sdk migrate`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#migrate) which adds a main.go source file and any associated source files for an operator that is not of the "go" type. ([#887](https://github.com/operator-framework/operator-sdk/pull/887) and [#897](https://github.com/operator-framework/operator-sdk/pull/897))
- New commands [`operator-sdk run ansible`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#ansible) and [`operator-sdk run helm`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#helm) which run the SDK as ansible  and helm operator processes, respectively. These are intended to be used when running in a Pod inside a cluster. Developers wanting to run their operator locally should continue to use `up local`. ([#887](https://github.com/operator-framework/operator-sdk/pull/887) and [#897](https://github.com/operator-framework/operator-sdk/pull/897))
- Ansible operator proxy added the cache handler which allows the get requests to use the operators cache. [#760](https://github.com/operator-framework/operator-sdk/pull/760)
- Ansible operator proxy added ability to dynamically watch dependent resource that were created by ansible operator. [#857](https://github.com/operator-framework/operator-sdk/pull/857)
- Ansible-based operators have leader election turned on by default. When upgrading, add environment variable `POD_NAME` to your operator's Deployment using the Kubernetes downward API. To see an example, run `operator-sdk new --type=ansible ...` and see file `deploy/operator.yaml`.
- A new command [`operator-sdk scorecard`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#scorecard) which runs a series of generic tests on operators to ensure that an operator follows best practices. For more information, see the [`Scorecard Documentation`](doc/test-framework/scorecard.md)

### Changed

- The official images for the Ansible and Helm operators have moved! Travis now builds, tags, and pushes operator base images during CI ([#832](https://github.com/operator-framework/operator-sdk/pull/832)).
  - [quay.io/operator-framework/ansible-operator](https://quay.io/repository/operator-framework/ansible-operator)
  - [quay.io/operator-framework/helm-operator](https://quay.io/repository/operator-framework/helm-operator)

### Bug Fixes

- Fixes deadlocks during operator deployment rollouts, which were caused by operator pods requiring a leader election lock to become ready ([#932](https://github.com/operator-framework/operator-sdk/pull/932))

## v0.3.0

### Added

- Helm type operator generation support ([#776](https://github.com/operator-framework/operator-sdk/pull/776))

### Changed

- The SDK's Kubernetes Golang dependency versions/revisions have been updated from `v1.11.2` to `v1.12.3`. ([#807](https://github.com/operator-framework/operator-sdk/pull/807))
- The controller-runtime version has been updated from `v0.1.4` to `v0.1.8`. See the `v0.1.8` [release notes](https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.1.8) for details.
- The SDK now generates the CRD with the status subresource enabled by default. See the [client doc](https://github.com/operator-framework/operator-sdk/blob/master/doc/user/client.md#updating-status-subresource) on how to update the status subresource. ([#787](https://github.com/operator-framework/operator-sdk/pull/787))

### Deprecated

### Removed

### Bug Fixes

## v0.2.1

### Bug Fixes

- Pin controller-runtime version to v0.1.4 to fix dependency issues and pin ansible idna package to version 2.7 ([#831](https://github.com/operator-framework/operator-sdk/pull/831))

## v0.2.0

### Changed

- The SDK now uses logr as the default logger to unify the logging output with the controller-runtime logs. Users can still use a logger of their own choice. See the [logging doc](https://github.com/operator-framework/operator-sdk/blob/master/doc/user/logging.md) on how the SDK initializes and uses logr.
- Ansible Operator CR status better aligns with [conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#typical-status-properties). ([#639](https://github.com/operator-framework/operator-sdk/pull/639))

### Added

- A new command [`operator-sdk print-deps`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#print-deps) which prints Golang packages and versions expected by the current Operator SDK version. Supplying `--as-file` prints packages and versions in Gopkg.toml format. ([#772](https://github.com/operator-framework/operator-sdk/pull/772))
- Add [`cluster-scoped`](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#operator-scope) flag to `operator-sdk new` command ([#747](https://github.com/operator-framework/operator-sdk/pull/747))
- Add [`up-local`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#flags-9) flag to `test local` subcommand ([#781](https://github.com/operator-framework/operator-sdk/pull/781))
- Add [`no-setup`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#flags-9) flag to `test local` subcommand ([#770](https://github.com/operator-framework/operator-sdk/pull/770))
- Add [`image`](https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md#flags-9) flag to `test local` subcommand ([#768](https://github.com/operator-framework/operator-sdk/pull/768))
- Ansible Operator log output includes much more information for troubleshooting ansible errors. ([#713](https://github.com/operator-framework/operator-sdk/pull/713))
- Ansible Operator periodic reconciliation can be disabled ([#739](https://github.com/operator-framework/operator-sdk/pull/739))

### Bug fixes

- Make operator-sdk command work with composed GOPATH ([#676](https://github.com/operator-framework/operator-sdk/pull/676))
- Ansible Operator "--kubeconfig" command line option fixed ([#705](https://github.com/operator-framework/operator-sdk/pull/705))

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

- All the modules in [`pkg/sdk`](https://github.com/operator-framework/operator-sdk/tree/4a9d5a5b0901b24679d36dced0a186c525e1bffd/pkg/sdk) have been combined into a single package. `action`, `handler`, `informer` `types` and `query` pkgs have been consolidated into `pkg/sdk`. [#242](https://github.com/operator-framework/operator-sdk/pull/242)
- The SDK exposes the Kubernetes clientset via `k8sclient.GetKubeClient()` #295
- The SDK now vendors the k8s code-generators for an operator instead of using the prebuilt image `gcr.io/coreos-k8s-scale-testing/codegen:1.9.3` [#319](https://github.com/operator-framework/operator-sdk/pull/242)
- The SDK exposes the Kubernetes rest config via `k8sclient.GetKubeConfig()` #338
- Use `time.Duration` instead of `int` for `sdk.Watch` [#427](https://github.com/operator-framework/operator-sdk/pull/427)

### Fixed

- The cache of available clients is being reset every minute for discovery of newely added resources to a cluster. [#280](https://github.com/operator-framework/operator-sdk/pull/280)
