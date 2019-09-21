# Operator SDK Version upgrade guide

This document aims to facilitate the process of upgrading the Operator SDK version for an existing operator project.

In most cases the upgrading the SDK version should only entail updating the operator's SDK dependency version in the `Gopkg.toml` or `go.mod` file.
For some versions it might also be necessary to update the upstream Kubernetes and controller-runtime dependencies.

The full list of dependencies and their versions required by a particular version of the SDK can be viewed by using the [`operator-sdk print-deps`][print-deps-cli] command. Use this command to update the project `Gopkg.toml`/`go.mod` file accordingly as you upgrade to a new SDK version.


For some SDK versions after `v0.1.0` there might be minor breaking changes in the controller-runtime APIs, `operator-sdk` CLI or the expected project layout and file names. These breaking changes will usually be outlined in the [CHANGELOG][changelog]/[release-notes][release-notes] for each release version.

For releases that have a large number of breaking changes or involve a significant refactoring of the APIs and project layout there will be a migration guide similar to the [v0.1.0 migration guide][v0.1.0-migration-guide].

The following sections outline the upgrade steps for each SDK version along with steps necessary for any associated breaking changes.

## `v0.2.x`

- Update the SDK constraint in `Gopkg.toml` to version `v0.2.1` and run `dep ensure` to update the vendor directory.
  ```TOML
  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.2.1"
  ```

## `v0.3.x`

- Update the SDK constraint in `Gopkg.toml` to version `v0.3.0`, the kubernetes dependencies to `kubernetes-1.12.3` revisions, and the controller-runtime version to `v0.1.8`. Then run `dep ensure` to update the vendor directory.
  ```TOML
  [[override]]
    name = "k8s.io/code-generator"
    # revision for tag "kubernetes-1.12.3"
    revision = "3dcf91f64f638563e5106f21f50c31fa361c918d"

  [[override]]
    name = "k8s.io/api"
    # revision for tag "kubernetes-1.12.3"
    revision = "b503174bad5991eb66f18247f52e41c3258f6348"

  [[override]]
    name = "k8s.io/apiextensions-apiserver"
    # revision for tag "kubernetes-1.12.3"
    revision = "0cd23ebeb6882bd1cdc2cb15fc7b2d72e8a86a5b"

  [[override]]
    name = "k8s.io/apimachinery"
    # revision for tag "kubernetes-1.12.3"
    revision = "eddba98df674a16931d2d4ba75edc3a389bf633a"

  [[override]]
    name = "k8s.io/client-go"
    # revision for tag "kubernetes-1.12.3"
    revision = "d082d5923d3cc0bfbb066ee5fbdea3d0ca79acf8"

  [[override]]
    name = "sigs.k8s.io/controller-runtime"
    version = "=v0.1.8"

  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.3.0"
  ```

## `v0.4.x`

- Update the SDK constraint in `Gopkg.toml` to version `v0.4.1` and run `dep ensure` to update the vendor directory.
  ```TOML
  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.4.1"
  ```

## `v0.5.x`

- Update the SDK constraint in `Gopkg.toml` to version `v0.5.0`, the kubernetes dependencies to `kubernetes-1.13.1` revisions, and the controller-runtime version to `v0.1.10`.
  ```TOML
  [[override]]
    name = "k8s.io/code-generator"
    # revision for tag "kubernetes-1.13.1"
    revision = "c2090bec4d9b1fb25de3812f868accc2bc9ecbae"

  [[override]]
    name = "k8s.io/api"
    # revision for tag "kubernetes-1.13.1"
    revision = "05914d821849570fba9eacfb29466f2d8d3cd229"

  [[override]]
    name = "k8s.io/apiextensions-apiserver"
    # revision for tag "kubernetes-1.13.1"
    revision = "0fe22c71c47604641d9aa352c785b7912c200562"

  [[override]]
    name = "k8s.io/apimachinery"
    # revision for tag "kubernetes-1.13.1"
    revision = "2b1284ed4c93a43499e781493253e2ac5959c4fd"

  [[override]]
    name = "k8s.io/client-go"
    # revision for tag "kubernetes-1.13.1"
    revision = "8d9ed539ba3134352c586810e749e58df4e94e4f"

  [[override]]
    name = "sigs.k8s.io/controller-runtime"
    version = "=v0.1.10"

  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.5.0"
  ```

- Append the following new constraints to your `Gopkg.toml`.
  ```TOML
  [[override]]
    name = "k8s.io/kube-openapi"
    revision = "0cf8f7e6ed1d2e3d47d02e3b6e559369af24d803"

  [[override]]
    name = "github.com/go-openapi/spec"
    branch = "master"

  [[override]]
    name = "sigs.k8s.io/controller-tools"
    version = "=v0.1.8"
  ```

- Update the `required` dependencies in `Gopkg.toml` to include `sigs.k8s.io/controller-tools/pkg/crd/generator` and change `k8s.io/code-generator/cmd/openapi-gen` to `k8s.io/kube-openapi/cmd/openapi-gen`. Then run `dep ensure` to update the vendor directory.
  ```TOML
  required = [
    "k8s.io/code-generator/cmd/defaulter-gen",
    "k8s.io/code-generator/cmd/deepcopy-gen",
    "k8s.io/code-generator/cmd/conversion-gen",
    "k8s.io/code-generator/cmd/client-gen",
    "k8s.io/code-generator/cmd/lister-gen",
    "k8s.io/code-generator/cmd/informer-gen",
    "k8s.io/kube-openapi/cmd/openapi-gen",
    "k8s.io/gengo/args",
    "sigs.k8s.io/controller-tools/pkg/crd/generator",
  ]
  ```

## `v0.6.x`

- Update the SDK constraint in `Gopkg.toml` to version `v0.6.0` and run `dep ensure` to update the vendor directory.
  ```TOML
  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.6.0"
  ```

- The `operator-sdk olm-catalog` command now expects and generates manifests in the operator-registry [manifest format][manifest-format]. Generate your CSVs in the new layout by using `operator-sdk olm-catalog gen-csv` command, or modify the layout of your existing CSV manifests directory similar to the example below.
  ```console
  $ tree deploy/olm-catalog
  deploy/olm-catalog/
  └── memcached-operator
      ├── 0.1.0
      │   └── memcached-operator.v0.1.0.clusterserviceversion.yaml
      ├── 0.2.0
      │   └── memcached-operator.v0.2.0.clusterserviceversion.yaml
      └── memcached-operator.package.yaml
  ```

## `v0.7.x`

- Update the SDK constraint in `Gopkg.toml` to version `v0.7.1` and run `dep ensure` to update the vendor directory.
  ```TOML
  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.7.1"
  ```

## `v0.8.x`

The SDK version `v0.8.x` supports scaffolding projects to use Go modules by default. It is recommended that you migrate your operator project to use modules for dependency management, however you can choose to keep using `dep`. The upgrade steps for both are outlined below:

### `dep`

- Update the SDK constraint in `Gopkg.toml` to version `v0.8.2`.
  ```TOML
  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.8.2"
  ```
- Pin the controller-tools dependency to the following revision. See the release notes or [#1278](https://github.com/operator-framework/operator-sdk/pull/1278/) for why this is needed.
  ```TOML
  [[override]]
    name = "sigs.k8s.io/controller-tools"
    revision = "9d55346c2bde73fb3326ac22eac2e5210a730207"
  ```
- Run `dep ensure` to update the vendor directory.

### `modules`

To get familiar with Go modules read the [modules wiki][modules-wiki]. In particular the section on [migrating to modules][migrating-to-modules].

- Ensure that you have Go 1.12+ and [Mercurial][mercurial] 3.9+ installed.
- Activate Go modules support for your project in `$GOPATH/src` by setting the env `GO111MODULES=on`. See [activating modules][activating-modules] for more details.
- Initialize a new `go.mod` file by running `go mod init`.
- Append the following to the end of your `go.mod` file to pin the operator-sdk and other upstream dependencies to the required versions.
```
// Pinned to kubernetes-1.13.1
replace (
	k8s.io/api => k8s.io/api v0.0.0-20181213150558-05914d821849
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181213153335-0fe22c71c476
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/client-go => k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180711000925-0cf8f7e6ed1d
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.10
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
)

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.8.2
```
- Run `go mod tidy` to clean up the `go.mod` file. 
  - In case of any go module loading errors, consult the default [`v0.8.2` go.mod dependencies][v0.8.2-go-mod] scaffolded by the operator-sdk to resolve any differences. You can also view this file by scaffolding a new project with operator-sdk `v0.8.2`.
- Ensure that you can build the project with `operator-sdk build`
- Finally remove `Gopkg.lock`, `Gopkg.toml` and the vendor directory.

### Breaking changes

Upon updating the project to `v0.8.2` the following breaking changes apply:

- On running the command `operator-sdk generate openapi`, the CRD manifests at `deploy/crds/<group>_<version>_<kind>.crd` for all API types will now be regenerated based on their source files `pkg/apis/..._types.go`. So if you have made any manual edits to the default generated CRD manifest, e.g manually written the validation block or specified the naming (`spec.names`), then that information be overwritten when the CRD is regenerated. 

  The correct way to specify CRD fields like naming, validation, subresources etc is by using `// +kubebuilder` marker comments. Consult the [legacy kubebuilder documentation][legacy-kubebuilder-doc-crd] to see what CRD fields can be generated via `// +kubebuilder` marker comments.

  **Note:** The version of controller-tools tied to this release does not support settting the `spec.scope` field of the CRD. Use the marker comment `+genclient:nonNamespaced` to set `spec.scope=Cluster` if necessary. See the example below:
  ```Go
  // MemcachedSpec defines the desired state of Memcached
  type MemcachedSpec struct {
    // +kubebuilder:validation:Maximum=5
    // +kubebuilder:validation:Minimum=1
    Size int32 `json:"size"`
  }

  // MemcachedStatus defines the observed state of Memcached
  type MemcachedStatus struct {
    // +kubebuilder:validation:MaxItems=5
    // +kubebuilder:validation:MinItems=1
    Nodes []string `json:"nodes"`
  }

  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

  // Memcached is the Schema for the memcacheds API
  // +kubebuilder:subresource:status
  // +kubebuilder:resource:shortName="mc"
  // +genclient:nonNamespaced
  type Memcached struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MemcachedSpec   `json:"spec,omitempty"`
    Status MemcachedStatus `json:"status,omitempty"`
  }
  ```

## `v0.9.x`

- The function `ExposeMetricsPort()` has been replaced with `CreateMetricsService()` [#1560](https://github.com/operator-framework/operator-sdk/pull/1560). 
  
  Replace the following line in `cmd/manager/main.go`
  ```Go
    _, err = metrics.ExposeMetricsPort(ctx, metricsPort)
  ```
  with
  ```Go
	servicePorts := []v1.ServicePort{
    {Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
    {Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
  }

  _, err = metrics.CreateMetricsService(ctx, servicePorts)
  ```

### `dep`

- Update the SDK constraint in `Gopkg.toml` to version `v0.9.0`, the kubernetes dependencies to `kubernetes-1.13.4` revisions, and the controller-runtime version to `v0.1.12`.
  ```TOML
  [[override]]
    name = "k8s.io/api"
    # revision for tag "kubernetes-1.13.4"
    revision = "5cb15d34447165a97c76ed5a60e4e99c8a01ecfe"
  [[override]]
    name = "k8s.io/apiextensions-apiserver"
    # revision for tag "kubernetes-1.13.4"
    revision = "d002e88f6236312f0289d9d1deab106751718ff0"
  [[override]]
    name = "k8s.io/apimachinery"
    # revision for tag "kubernetes-1.13.4"
    revision = "86fb29eff6288413d76bd8506874fddd9fccdff0"
  [[override]]
    name = "k8s.io/client-go"
    # revision for tag "kubernetes-1.13.4"
    revision = "b40b2a5939e43f7ffe0028ad67586b7ce50bb675"
  [[override]]
    name = "github.com/coreos/prometheus-operator"
    version = "=v0.29.0"
  [[override]]
    name = "sigs.k8s.io/controller-runtime"
    version = "=v0.1.12"
  [[constraint]]
    name = "github.com/operator-framework/operator-sdk"
    version = "=v0.9.0"
  ```
- Append the contraint for `k8s.io/kube-state-metrics`.
  ```TOML
  [[override]]
    name = "k8s.io/kube-state-metrics"
    version = "v1.6.0"
  ```
- Run `dep ensure` to update the vendor directory.

### modules

- Update the `replace` directives in your `go.mod` file for the SDK, kubernetes, controller-runtime and kube-state metrics dependencies to the following versions.
  ```
  // Pinned to kubernetes-1.13.4
  replace (
    k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
    k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
    k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
    k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
  )
  replace (
    github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
    sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.12
    sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
    k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.6.0
  )
  replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.9.0
  ```

[legacy-kubebuilder-doc-crd]: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
[v0.8.2-go-mod]: https://github.com/operator-framework/operator-sdk/blob/28bd2b0d4fd25aa68e15d928ae09d3c18c3b51da/internal/pkg/scaffold/go_mod.go#L40-L94
[activating-modules]: https://github.com/golang/go/wiki/Modules#how-to-install-and-activate-module-support
[mercurial]: https://www.mercurial-scm.org/downloads
[migrating-to-modules]: https://github.com/golang/go/wiki/Modules#migrating-to-modules
[modules-wiki]: https://github.com/golang/go/wiki/Modules#migrating-to-modules
[print-deps-cli]: ../sdk-cli-reference.md#print-deps
[changelog]: ../../CHANGELOG.md
[release-notes]: https://github.com/operator-framework/operator-sdk/releases
[v0.1.0-migration-guide]: ./v0.1.0-migration-guide.md
[manifest-format]: https://github.com/operator-framework/operator-registry#manifest-format