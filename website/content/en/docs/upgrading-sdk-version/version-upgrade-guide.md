---
title: Operator SDK Version upgrade guide
linkTitle: v0.2.x to v0.17.x
weight: 999983000
description: A guide to upgrading the Operator SDK version for an existing operator project from v0.2.x all the way through to 0.17.x.
---

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
    # **revision for tag "kubernetes-1.12.3"**
    revision = "3dcf91f64f638563e5106f21f50c31fa361c918d"

  [[override]]
    name = "k8s.io/api"
    # **revision for tag "kubernetes-1.12.3"**
    revision = "b503174bad5991eb66f18247f52e41c3258f6348"

  [[override]]
    name = "k8s.io/apiextensions-apiserver"
    # **revision for tag "kubernetes-1.12.3"**
    revision = "0cd23ebeb6882bd1cdc2cb15fc7b2d72e8a86a5b"

  [[override]]
    name = "k8s.io/apimachinery"
    # **revision for tag "kubernetes-1.12.3"**
    revision = "eddba98df674a16931d2d4ba75edc3a389bf633a"

  [[override]]
    name = "k8s.io/client-go"
    # **revision for tag "kubernetes-1.12.3"**
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
    # **revision for tag "kubernetes-1.13.1"**
    revision = "c2090bec4d9b1fb25de3812f868accc2bc9ecbae"

  [[override]]
    name = "k8s.io/api"
    # **revision for tag "kubernetes-1.13.1"**
    revision = "05914d821849570fba9eacfb29466f2d8d3cd229"

  [[override]]
    name = "k8s.io/apiextensions-apiserver"
    # **revision for tag "kubernetes-1.13.1"**
    revision = "0fe22c71c47604641d9aa352c785b7912c200562"

  [[override]]
    name = "k8s.io/apimachinery"
    # **revision for tag "kubernetes-1.13.1"**
    revision = "2b1284ed4c93a43499e781493253e2ac5959c4fd"

  [[override]]
    name = "k8s.io/client-go"
    # **revision for tag "kubernetes-1.13.1"**
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

**`dep`**

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

**`modules`**

To get familiar with Go modules read the [modules wiki][modules-wiki]. In particular the section on [migrating to modules][migrating-to-modules].

- Ensure that you have Go 1.12+ and [Mercurial][mercurial] 3.9+ installed.
- Activate Go modules support for your project in `$GOPATH/src` by setting the env `GO111MODULE=on`. See [activating modules][activating-modules] for more details.
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

**Breaking changes**

Upon updating the project to `v0.8.2` the following breaking changes apply:

- On running the command `operator-sdk generate openapi`, the CRD manifests at `deploy/crds/<group>_<version>_<kind>.crd` for all API types will now be regenerated based on their source files `pkg/apis/..._types.go`. So if you have made any manual edits to the default generated CRD manifest, e.g manually written the validation block or specified the naming (`spec.names`), then that information be overwritten when the CRD is regenerated.

  The correct way to specify CRD fields like naming, validation, subresources etc is by using `// +kubebuilder` marker comments. Consult the [legacy kubebuilder documentation][legacy-kubebuilder-doc-crd] to see what CRD fields can be generated via `// +kubebuilder` marker comments.

  **Note:** The version of controller-tools tied to this release does not support settting the `spec.scope` field of the CRD. Use the marker comment `+genclient:nonNamespaced` to set `spec.scope=Cluster` if necessary. See the example below:
  ```Go
  // MemcachedSpec defines the desired state of Memcached
  type MemcachedSpec struct {
    //+kubebuilder:validation:Maximum=5
    //+kubebuilder:validation:Minimum=1
    Size int32 `json:"size"`
  }

  // MemcachedStatus defines the observed state of Memcached
  type MemcachedStatus struct {
    //+kubebuilder:validation:MaxItems=5
    //+kubebuilder:validation:MinItems=1
    Nodes []string `json:"nodes"`
  }

  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

  // Memcached is the Schema for the memcacheds API
  //+kubebuilder:subresource:status
  //+kubebuilder:resource:shortName="mc"
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

**`dep`**

- Update the SDK constraint in `Gopkg.toml` to version `v0.9.0`, the kubernetes dependencies to `kubernetes-1.13.4` revisions, and the controller-runtime version to `v0.1.12`.
  ```TOML
  [[override]]
    name = "k8s.io/api"
    # **revision for tag "kubernetes-1.13.4"**
    revision = "5cb15d34447165a97c76ed5a60e4e99c8a01ecfe"
  [[override]]
    name = "k8s.io/apiextensions-apiserver"
    # **revision for tag "kubernetes-1.13.4"**
    revision = "d002e88f6236312f0289d9d1deab106751718ff0"
  [[override]]
    name = "k8s.io/apimachinery"
    # **revision for tag "kubernetes-1.13.4"**
    revision = "86fb29eff6288413d76bd8506874fddd9fccdff0"
  [[override]]
    name = "k8s.io/client-go"
    # **revision for tag "kubernetes-1.13.4"**
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

**modules**

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

## `v0.10.x`

- The scorecard configuration format for the `operator-sdk scorecard` command has changed. See [`doc/test-framework/scorecard`](https://github.com/operator-framework/operator-sdk/blob/v0.10.x/doc/test-framework/scorecard.md) for more info.
- The CSV config field `role-path` is now `role-paths` and takes a list of strings.
    Replace:
    ```yaml
    role-path: path/to/role.yaml
    ```
    with:
    ```yaml
    role-paths:
    - path/to/role.yaml
    ```

**modules**

- Ensure the the following `replace` directives are present in your `go.mod` file:
    ```
    replace (
            github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
            // Pinned to v2.9.2 (kubernetes-1.13.1) so https://proxy.golang.org can
            // resolve it correctly.
            github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20190424153033-d3245f150225
            k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.6.0
            sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.12
            sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
    )

    replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.10.0
    ```

## `v0.11.x`

**NOTE:** this version uses Kubernetes v1.14.x and controller-runtime v0.2.x, both of which have breaking API changes. See the [changelog][changelog] for more details.

**dep**

- Remove the `required = [ ... ]` section and comment from the top of your `Gopkg.toml` file.
- Update the following overrides in `Gopkg.toml`:
    ```TOML
    [[override]]
      name = "k8s.io/api"
      # **revision for tag "kubernetes-1.14.1"**
      revision = "6e4e0e4f393bf5e8bbff570acd13217aa5a770cd"
    [[override]]
      name = "k8s.io/apiextensions-apiserver"
      # **revision for tag "kubernetes-1.14.1"**
      revision = "727a075fdec8319bf095330e344b3ccc668abc73"
    [[override]]
      name = "k8s.io/apimachinery"
      # **revision for tag "kubernetes-1.14.1"**
      revision = "6a84e37a896db9780c75367af8d2ed2bb944022e"
    [[override]]
      name = "k8s.io/client-go"
      # **revision for tag "kubernetes-1.14.1"**
      revision = "1a26190bd76a9017e289958b9fba936430aa3704"
    [[override]]
      name = "github.com/coreos/prometheus-operator"
      version = "=v0.31.1"
    [[override]]
      name = "sigs.k8s.io/controller-runtime"
      version = "=v0.2.2"
    [[constraint]]
      name = "github.com/operator-framework/operator-sdk"
      version = "=v0.11.0"
    ```
- Append an override for `gopkg.in/fsnotify.v1`, which is required when resolving controller-runtime dependencies:
    ```TOML
    [[override]]
      name = "gopkg.in/fsnotify.v1"
      source = "https://github.com/fsnotify/fsnotify.git"
    ```
- Remove the `k8s.io/kube-state-metrics` override.
- Run `dep ensure` to update the vendor directory.

**modules**

- Ensure the the following `replace` directives are present in your `go.mod` file:
    ```
    // Pinned to kubernetes-1.14.1
    replace (
    	k8s.io/api => k8s.io/api kubernetes-1.14.1
    	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver kubernetes-1.14.1
    	k8s.io/apimachinery => k8s.io/apimachinery kubernetes-1.14.1
    	k8s.io/client-go => k8s.io/client-go kubernetes-1.14.1
    	k8s.io/cloud-provider => k8s.io/cloud-provider kubernetes-1.14.1
    )

    replace (
    	// Indirect operator-sdk dependencies use git.apache.org, which is frequently
    	// down. The github mirror should be used instead.
    	// Locking to a specific version (from 'go mod graph'):
    	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
    	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.31.1
    	// Pinned to v2.10.0 (kubernetes-1.14.1) so https://proxy.golang.org can
    	// resolve it correctly.
    	github.com/prometheus/prometheus => github.com/prometheus/prometheus d20e84d0fb64aff2f62a977adc8cfb656da4e286
    )

    replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.11.0
    ```

**Import updates**

- Replace import `sigs.k8s.io/controller-runtime/pkg/runtime/scheme` with `sigs.k8s.io/controller-runtime/pkg/scheme` in:
  - `./pkg/apis/<group>/<version>/register.go`
- Replace import `sigs.k8s.io/controller-runtime/pkg/runtime/log` with `sigs.k8s.io/controller-runtime/pkg/log` in:
  - `cmd/manager/main.go`
  - `./pkg/controller/<kind>/<kind>_controller.go`
- Replace import `sigs.k8s.io/controller-runtime/pkg/runtime/signals` with `sigs.k8s.io/controller-runtime/pkg/manager/signals` in:
  - `cmd/manager/main.go`
- Remove import `sigs.k8s.io/controller-tools/pkg/crd/generator` from:
  - `tools.go`

**controller-runtime API updates**

All method signatures for [`sigs.k8s.io/controller-runtime/pkg/client.Client`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L104) and [`sigs.k8s.io/controller-runtime/pkg/client.StatusWriter`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L91) (except for `Client.Get()`) have been updated. Each now uses a variadic option interface parameter typed for each method.
- `Client.List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error` is now [`Client.List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L61).
    Replace:
    ```go
    listOpts := &client.ListOptions{}
    listOpts.InNamespace("namespace")
    err = r.client.List(context.TODO(), listOps, podList)
    ```
    with:
    ```go
    listOpts := []client.ListOption{
      client.InNamespace("namespace"),
    }
    err = r.client.List(context.TODO(), podList, listOpts...)
    ```
- `Client.Create(ctx context.Context, obj runtime.Object) error` is now [`Client.Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L67). No updates need to be made. See the [client doc][client-doc] for a discussion of `client.CreateOption`.
- `Client.Update(ctx context.Context, obj runtime.Object) error` is now [`Client.Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L74). No updates need to be made. See the [client doc][client-doc] for a discussion of `client.UpdateOption`.
- `Client.Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error` is now [`Client.Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOption) error`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L70). Although the option interface has changed, the way each `client.DeleteOption` is created is the same as before. No updates need to be made. See the [client doc][client-doc] for a discussion of `client.DeleteOption`.
- `StatusWriter.Update(ctx context.Context, obj runtime.Object) error` is now [`Update(ctx context.Context, obj runtime.Object, opts ...UpdateOption) error`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.2.0/pkg/client/interfaces.go#L95). No updates need to be made. See the [client doc][client-doc] for a discussion of `client.UpdateOption`.

**OpenAPI updates**

- Run the command `operator-sdk generate openapi` and ensure that no errors such as `API rule violation` are raised. For further information see the [API rules][api-rules] documentation.

**NOTE:** You may need to add or remove markers (code annotations) to fix issues found when running `generate openapi`. Usage of markers in API code is discussed in the kubebuilder CRD generation [documentation][generating-crd] and in marker [documentation][markers]. A full list of OpenAPIv3 validation markers can be found [here](https://book.kubebuilder.io/reference/markers/crd-validation.html).

**TIPS:**
- If the `+kubebuilder:validation:Pattern` has commas, then surround the expressions in backticks.
- If you are using `+kubebuilder:validation:Enum` then either surround the expression list in curly braces and quote each expression, or separate each expression using semicolons.

**Operator SDK updates**

- [`pkg/test.FrameworkClient`](https://github.com/operator-framework/operator-sdk/blob/947a464/pkg/test/client.go#L33) `List()` and `Delete()` method invocations should be updated to match those of `Client.List()` and `Client.Delete()`, described above.
- The annotation to assign a scope to your CRD has changed. For the following changes, note that `<resource>` is the plural lower-case CRD Kind found at `spec.names.plural`.
    - For `Namespaced`-scoped operators, add a `+kubebuilder:resource:path=<resource>,scope=Namespaced` comment above your kind type in `pkg/apis/<group>/<version>/<kind>_types.go`.
    - For `Cluster`-scoped operators, replace the `+genclient:nonNamespaced` comment above your kind type in `pkg/apis/<group>/<version>/<kind>_types.go` with `+kubebuilder:resource:path=<resource>,scope=Cluster`.
- CRD file names now have the form `<full group>_<resource>_crd.yaml`, and CR file names now have the form `<full group>_<version>_<kind>_cr.yaml`. `<full group>` is the full group name of your CRD found at `spec.group`, and `<resource>` is the plural lower-case CRD Kind found at `spec.names.plural`. To migrate:
    - Run `operator-sdk generate openapi`. CRD manifest files with new names containing versioned validation and subresource blocks will be generated.
    - Delete the old CRD manifest files.
    - Rename CR manifest file names from `<group>_<version>_<kind>_cr.yaml` to `<full group>_<version>_<kind>_cr.yaml`.

## `v0.12.x`

**Go version**

- Ensure that you are using a go version 1.13+

**dep**

Using `dep` is no longer supported. Follow [Go's official blog post about migrating to modules](https://blog.golang.org/migrating-to-go-modules) to learn how to migrate your project.

**modules**

- Ensure the the following `require` modules and `replace` directives with the specific versions are present in your `go.mod` file:

```
require (
    github.com/go-openapi/spec v0.19.0
    github.com/operator-framework/operator-sdk v0.12.1-0.20191112211508-82fc57de5e5b
    github.com/spf13/pflag v1.0.3
    k8s.io/api v0.0.0
    k8s.io/apimachinery v0.0.0
    k8s.io/client-go v11.0.0+incompatible
    k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
    sigs.k8s.io/controller-runtime v0.3.0
)

// Pinned to kubernetes-1.15.4
replace (
    k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
    k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604
    k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
    k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918200908-1e17798da8c1
    k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918202139-0b14c719ca62
    k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
    k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918203125-ae665f80358a
    k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918202959-c340507a5d48
    k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
    k8s.io/component-base => k8s.io/component-base v0.0.0-20190918200425-ed2f0867c778
    k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79
    k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918203248-97c07dcbb623
    k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918201136-c3a845f1fbb2
    k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918202837-c54ce30c680e
    k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918202429-08c8357f8e2d
    k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918202713-c34a54b3ec8e
    k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918202550-958285cf3eef
    k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918203421-225f0541b3ea
    k8s.io/metrics => k8s.io/metrics v0.0.0-20190918202012-3c1ca76f5bda
    k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918201353-5cc279503896
)
```

**NOTE**: Check [here](https://github.com/operator-framework/operator-sdk-samples/pull/90/files#diff-e15cac8b95d260726ca9db9fb25d9230) an example of this upgrade to see the changes from the version `0.11.0` to `0.12.0`.

- Run `go mod tidy` to update the project modules
- Run the command `operator-sdk generate k8s` to ensure that your resources will be updated
- Run the command `operator-sdk generate openapi` and ensure that no errors such as `API rule violation` are raised. For further information see the [API rules][api-rules] documentation.

**(Optional) Update your operator to print its version**

In v0.12.0, the SDK team updated the scaffold for `cmd/manager/main.go` to include the operator's version in the output produced by the `printVersion()` function. See [#1953](https://github.com/operator-framework/operator-sdk/pull/1953)

To add this feature to your operator, add the following lines in `<project>/cmd/manager/main.go`:

```go
import (
	...
	"<your_module_path>/version"
	...
)

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	...
}
```

## `v0.13.x`

**modules**

- Ensure the the following `require` modules and `replace` directives with the specific versions are present in your `go.mod` file:

```
require (
	github.com/operator-framework/operator-sdk v0.13.0
	sigs.k8s.io/controller-runtime v0.4.0
)

// Pinned to kubernetes-1.16.2
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
```

- Run `go mod tidy` to update the project modules
- Run the command `operator-sdk generate k8s` to ensure that your resources will be updated
- Run the command `operator-sdk generate crds` to regenerate CRDs

**(Optional) Update the roles.yaml file**

Replace `*` per verbs in order to solve the issue [671](https://github.com/operator-framework/operator-sdk/issues/671) and make clear the permissions used.

**Example**

```
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
```

**Notable changes**

- **Deprecated:** Deprecated the `operator-sdk generate openapi` command. CRD generation is still supported with `operator-sdk generate crds`. It is now recommended to use [openapi-gen](https://github.com/kubernetes/kube-openapi/tree/master/cmd/openapi-gen) directly for OpenAPI code generation. The `generate openapi` subcommand will be removed in a future release.
- **Breaking change:** An existing CSV's `spec.customresourcedefinitions.owned` is now always overwritten except for each name, version, and kind on invoking olm-catalog gen-csv when Go API code annotations are present.
- **Potentially Breaking change:** Be aware that there are potentially other breaking changes due to the controller-runtime and Kubernetes version be upgraded from `v0.4.0` to `v1.16.2`, respectively. There may be breaking changes to Go client code due to both of those changes.

For further detailed information see [CHANGELOG](https://github.com/operator-framework/operator-sdk/blob/v0.14.0/CHANGELOG.md#v0130)

## `v0.14.x`

**modules**

- Ensure the the following `require` modules and `replace` directives with the specific versions are present in your `go.mod` file:

```
require (
	github.com/operator-framework/operator-sdk v0.14.1
	sigs.k8s.io/controller-runtime v0.4.0
)
// Pinned to kubernetes-1.16.2
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
```

- Run `go mod tidy` to update the project modules
- Run the command `operator-sdk generate k8s` to ensure that your resources will be updated
- Run the command `operator-sdk generate crds` to regenerate CRDs

**(Optional) Skip metrics logs when the operator is running locally**

There are changes to the default implementation of the metrics export. These changes require `cmd/manager/main.go` to be updated as follows.

Update imports:

```go
import (
	...
	"errors"
	...
)
```

Replace:

```go
func main() {
	...
	if err = serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
  ...
}
```

With:

```go
func main() {
	...
	// Add the Metrics Service
	addMetrics(ctx, cfg, namespace)
  ...
}
```

And then, add implementation for `addMetrics`:

```go
// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config, namespace string) {
	if err := serveCRMetrics(cfg); err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}
```

**NOTE**: For more information check the PR which is responsible for the above changes [#2190](https://github.com/operator-framework/operator-sdk/pull/2190).

**Deprecations**

The `github.com/operator-framework/operator-sdk/pkg/restmapper` package was deprecated in favor of the `DynamicRESTMapper` implementation in [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client/apiutil#NewDiscoveryRESTMapper). Users should migrate to controller-runtime's implementation, which is a drop-in replacement.

Replace:
```
github.com/operator-framework/operator-sdk/pkg/restmapper.DynamicRESTMapper
```

With:

```
sigs.k8s.io/controller-runtime/pkg/client/apiutil.DynamicRESTMapper
```

**Breaking Changes**

**Add `operator_sdk.util` Ansible collection**

The Ansible module `k8s_status` was extracted and is now provided by the `operator_sdk.util` Ansible collection. See [developer_guide](/docs/building-operators/ansible/development-tips/#custom-resource-status-management) for new usage.

To use the collection in a role, declare it at the root level in `meta/main.yaml`:
```yaml
collections:
- operator_sdk.util
```

To use it in a playbook, declare it in the play:
```yaml
- hosts: all
  collections:
   - operator_sdk.util
  tasks:
   - k8s_status:
       api_version: app.example.com/v1
       kind: Foo
       name: "{{ meta.name }}"
       namespace: "{{ meta.namespace }}"
       status:
         foo: bar
```

You can also use the fully-qualified name without declaring the collection:
```yaml
   - operator_sdk.util.k8s_status:
       api_version: app.example.com/v1
       kind: Foo
       name: "{{ meta.name }}"
       namespace: "{{ meta.namespace }}"
       status:
         foo: bar
```

**Notable Changes**

These notable changes contain just the most important user-facing changes. See the [CHANGELOG](https://github.com/operator-framework/operator-sdk/blob/v0.15.0/CHANGELOG.md#v0141) for details of the release.

**Ansible version update**

The Ansible version in the init projects was upgraded from `2.6` to `2.9` for collections support. Update the `meta/main.yaml` file.

Replace:
```yaml
...
 min_ansible_version: 2.6
...
```

With:
```yaml
...
 min_ansible_version: 2.9
...
```

**Helm Upgrade to V3**

The Helm operator packages and base image were upgraded from Helm v2 to Helm v3. Note that cluster state for pre-existing CRs using Helm v2-based operators will be automatically migrated to Helm v3's new release storage format, and existing releases may be upgraded due to changes in Helm v3's label injection.

If you are using any external helm v2 tooling with the your helm operator-managed releases, you will need to upgrade to the equivalent helm v3 tooling.

## `v0.15.x`

**modules**

- Ensure the the following `require` modules and `replace` directives with the specific versions are present in your `go.mod` file:

```
require (
	github.com/operator-framework/operator-sdk v0.15.2
	sigs.k8s.io/controller-runtime v0.4.0
)
// Pinned to kubernetes-1.16.2
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190924102528-32369d4db2ad // Required until https://github.com/operator-framework/operator-lifecycle-manager/pull/1241 is resolved
```

- Run `go mod tidy` to update the project modules
- Run the command `operator-sdk generate k8s` to ensure that your resources will be updated
- Run the command `operator-sdk generate crds` to regenerate CRDs

**Breaking Changes on Commands**

This release contains breaking changes in some commands.

- The `operator-sdk olm-catalog gen-csv` was replaced by `operator-sdk generate csv`
- The `operator-sdk up local` is now `operator-sdk run --local`. However, all functionality of this command is retained.
- And then, the `operator-sdk alpha olm [sub-commands] [flags]` was moved from `alpha` to its own sub-command. However, all functionality of this command is retained. To check run; `operator-sdk olm --help`.

**Breaking Changes for Helm and Ansible**

The `operator-sdk run ansible/helm` are now hidden commands in `exec-entrypoint ansible/helm`. However, all functionality of each sub-command is still the same. If you are using this feature then you will need to replace the `run` for `exec-entrypoint` as the following examples.

Replace:

```
oprator-sdk run ansible --watches-file=/opt/ansible/watches.yaml
```

With:

```
oprator-sdk exec-entrypoint ansible --watches-file=/opt/ansible/watches.yaml
```


Replace:

```
oprator-sdk run helm --watches-file=$HOME/watches.yaml
```

With:

```
oprator-sdk run exec-entrypoint helm --watches-file=$HOME/watches.yaml
```

See the [CHANGELOG](https://github.com/operator-framework/operator-sdk/blob/v0.16.0/CHANGELOG.md#v0151) for details of the release.

## v0.16.x

**modules**

- Ensure that the following `require` modules and `replace` directives with the specific versions are present in your `go.mod` file:

```
require (
	github.com/operator-framework/operator-sdk v0.16.0
	sigs.k8s.io/controller-runtime v0.4.0
)

// Pinned to kubernetes-1.16.2
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190924102528-32369d4db2ad // Required until https://github.com/operator-framework/operator-lifecycle-manager/pull/1241 is resolved
```

- Run `go mod tidy` to update the project modules
- Run the command `operator-sdk generate k8s` to ensure that your resources will be updated
- Run the command `operator-sdk generate crds` to regenerate CRDs

**Bug Fixes and Improvements for Metrics**

There are changes to the default implementation of the metrics export. These changes require `cmd/manager/main.go` to be updated as follows.

Replace:

```go
func main() {
  ...
  // Add the Metrics Service
	addMetrics(ctx, cfg, namespace)
  ...
}
```

With:

```go
func main() {
  ...
	// Add the Metrics Service
	addMetrics(ctx, cfg)
  ...
}
```

And then, update the default implementation of `addMetrics` and `serveCRMetrics` with:

```go
// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config) {
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
	}

	if err := serveCRMetrics(cfg, operatorNs); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}

	// The ServiceMonitor is created in the same namespace where the operator is deployed
	_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, operatorNs string) error {
	// The function below returns a list of filtered operator/CR specific GVKs. For more control, override the GVK list below
	// with your own custom logic. Note that if you are adding third party API schemas, probably you will need to
	// customize this implementation to avoid permissions issues.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

	// The metrics will be generated from the namespaces which are returned here.
	// NOTE that passing nil or an empty list of namespaces in GenerateAndServeCRMetrics will result in an error.
	ns, err := kubemetrics.GetNamespacesForMetrics(operatorNs)
	if err != nil {
		return err
	}

	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
```

**NOTE**: For more information check the PRs which are responsible for the above changes [#2606](https://github.com/operator-framework/operator-sdk/pull/2606),[#2603](https://github.com/operator-framework/operator-sdk/pull/2603) and [#2601](https://github.com/operator-framework/operator-sdk/pull/2601).

**(Optional) Support for watching multiple namespaces**

There are changes to add support for watching multiple namespaces. These changes require `cmd/manager/main.go` to be updated as follows.

Update imports:

```go
import (
	...
	"strings"

	...
	"sigs.k8s.io/controller-runtime/pkg/cache"
	...
)
```

Replace:

```go
func main() {
	...
	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
  ...
}
```

With:

```go
func main() {
	...
	// Set default manager options
	options := manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
  ...
}
```

**NOTE**: For more information check the PR which is responsible for the above changes [#2522](https://github.com/operator-framework/operator-sdk/pull/2522).

**Breaking changes**

**`TestCtx` in `pkg/test` has been deprecated**

 The type name `TestCtx` in `pkg/test` has been deprecated and renamed to `Context`. Users of the e2e framework should do the following:

 - Replace `TestCtx` with `Context`
 - Replace `NewTestCtx` with `NewContext`

**Scorecard only supports YAML config files**

The scorecard feature now only supports YAML config files. Config files with other extensions are no longer supported and should be changed to the YAML format. For further information see [`scorecard config file`](https://github.com/operator-framework/operator-sdk/blob/v0.16.x/doc/test-framework/scorecard.md#config-file)

**Breaking Changes for Ansible**

**Remove Ansible container sidecar**

The Ansible logs are now output in the operator container, so there is no longer a need for the Ansible container sidecar. To reflect this change, update the `deploy/operator.yaml` file as follows.

Remove:

```
- name: ansible
  command:
    - /usr/local/bin/ao-logs
    - /tmp/ansible-operator/runner
    - stdout
  # Replace this with the built image name**
  image: "REPLACE_IMAGE"
  imagePullPolicy: "Always"
  volumeMounts:
    - mountPath: /tmp/ansible-operator/runner
    name: runner
    readOnly: true
```

Replace:

```yaml
- name: operator
```

With:

```yaml
- name: {{your operator name which is the value of metadata.name in this file}}
```

By default the full Ansible logs will not be output, however, you can setup it via the `ANSIBLE_DEBUG_LOGS` environment variable in the `deploy/operator.yaml` file. See:

```
...
- name: ANSIBLE_DEBUG_LOGS
  value: "True"
...
```

**Migration to Ansible collections**

The core Ansible Kubernetes modules have been moved to the [`community.kubernetes` Ansible collection][kubernetes-ansible-collection]. Future development of the modules will occur there, with only critical bugfixes going into the modules in core. Additionally, the `operator_sdk.util` collection is no longer installed by default in the base image. Instead, users should add a `requirements.yml` to their project root, with the following content:

```yaml
collections:
  - kubernetes.core
  - operator_sdk.util
  - cloud.common
```

Users should then add the following stages to their `build/Dockerfile`:

```
COPY requirements.yml ${HOME}/requirements.yml
RUN ansible-galaxy collection install -r ${HOME}/requirements.yml \
 && chmod -R ug+rwx ${HOME}/.ansible
```

## v0.17.x

**modules**

- Ensure that the following `require` modules and `replace` directives with the specific versions are present in your `go.mod` file:

```
require (
	github.com/operator-framework/operator-sdk v0.17.2
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
  k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
  github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
)
```

- Run `go mod tidy` to update the project modules
- Run the command `operator-sdk generate k8s` to ensure that your resources will be updated
- Run the command `operator-sdk generate crds` to regenerate CRDs

**Breaking Changes**

**OpenAPI generation**

- The deprecated `operator-sdk generate openapi` command has been removed. This command generated CRDs and
  `zz_generated.openapi` files for operator APIs.

To generate CRDs, use `operator-sdk generate crds`.

To generate Go OpenAPI code, use `openapi-gen` directly. For example:

```bash
# Build the latest openapi-gen from source
which ./bin/openapi-gen > /dev/null || go build -o ./bin/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen

# Run openapi-gen for each of your API group/version packages
./bin/openapi-gen --logtostderr=true \
                  -i ./pkg/apis/<group>/<version> \
                  -o "" \
                  -O zz_generated.openapi \
                  -p ./pkg/apis/<group>/<version> \
                  -h ./hack/boilerplate.go.txt \
                  -r "-"
```

**Molecule Upgrade for Ansible based-operators**

The Molecule version for Ansible based-operators was upgraded from `2.22` to `3.0.2`. The following changes are required in the default scaffold files.

- Remove the `scenario.name` from `molecule.yaml` and then, ensure that any condition with will look for the folder name which determines the scenario name from now on
- Replace the lint with newer syntax from [documentation](https://molecule.readthedocs.io/contributing/#linting). See:

Replace:

```yaml
lint:
  name: yamllint
  options:
    config-data:
      line-length:
        max: 120
```

With:

```yaml
lint: |
  set -e
  yamllint -d "{extends: relaxed, rules: {line-length: {max: 120}}}" .
```

Replace:

```yaml
lint:
  name: ansible-lint
```

With:

```yaml
lint: |
  set -e
  ansible-lint
```

- Rename `molecule/$SCENARIO/playbook.yml` to `molecule/$SCENARIO/converge.yml` to avoid a deprecation message.
- Update the `.travis.yml` file to install the supported lints as follows.

Replace:

```yaml
install:
  - pip3 install docker molecule openshift jmespath
```

With:

```yaml
install:
  - pip3 install docker molecule ansible-lint yamllint flake8 openshift jmespath
```

**NOTE** To know more about how to upgrade your project to use the V3 Molecule version see [here](https://github.com/ansible-community/molecule/issues/2560).

**Deprecations**

**Test Framework**

- The methods `ctx.GetOperatorNamespace()` and `ctx.GetWatchNamespace()` were added to `pkg/test` in order to replace
`ctx.GetNamespace()` which is deprecated. In this way, replace the use of `ctx.GetNamespace()` in your project with
`ctx.GetOperatorNamespace()`.
- The `--namespace` flag from `operator-sdk run --local`, `operator-sdk test --local`, and `operator-sdk cleanup` was
deprecated and is replaced by `--watch-namespace` and `--operator-namespace`.

    The `--operator-namespace` flag can be used to set the namespace where the operator will be deployed. It will set the
    environment variable `OPERATOR_NAMESPACE`. If this value is not set, then it will be the namespace defined as in your
    current kubeconfig context.

    The `--watch-namespace` flag can be used to set the namespace(s) which the operator will watch for changes. It will set
    the environment variable `WATCH_NAMESPACE`. Use an explicit empty string to watch all namespaces or a comma-separated
    list of namespaces (e.g. "ns1,ns2") to watch multiple namespace when the operator is cluster-scoped. If using a list,
    then it should contain the namespace where the operator is deployed since the default metrics implementation will
    manage resources in the Operator's namespace. By default, `--watch-namespace` will be set to the operator namespace.

- If you've run `operator-sdk bundle create --generate-only`, move your bundle Dockerfile at
`<project-root>/deploy/olm-catalog/<operator-name>/Dockerfile` to `<project-root>/bundle.Dockerfile` and update the
first `COPY` from `COPY /*.yaml manifests/` to `COPY deploy/olm-catalog/<operator-name>/manifests manifests/`.

[legacy-kubebuilder-doc-crd]: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
[v0.8.2-go-mod]: https://github.com/operator-framework/operator-sdk/blob/28bd2b0d4fd25aa68e15d928ae09d3c18c3b51da/internal/pkg/scaffold/go_mod.go#L40-L94
[activating-modules]: https://github.com/golang/go/wiki/Modules#how-to-install-and-activate-module-support
[mercurial]: https://www.mercurial-scm.org/downloads
[migrating-to-modules]: https://github.com/golang/go/wiki/Modules#migrating-to-modules
[modules-wiki]: https://github.com/golang/go/wiki/Modules#migrating-to-modules
[print-deps-cli]: https://v0-19-x.sdk.operatorframework.io/docs/cli/operator-sdk_print-deps/
[changelog]: https://github.com/operator-framework/operator-sdk/blob/v1.3.0/CHANGELOG.md
[release-notes]: https://github.com/operator-framework/operator-sdk/releases
[v0.1.0-migration-guide]: ../v0.1.0-migration-guide
[manifest-format]: https://github.com/operator-framework/operator-registry#manifest-format
[client-doc]: https://v0-19-x.sdk.operatorframework.io/docs/golang/legacy/references/client/
[api-rules]: https://github.com/kubernetes/kubernetes/tree/36981002246682ed7dc4de54ccc2a96c1a0cbbdb/api/api-rules
[generating-crd]: https://book.kubebuilder.io/reference/generating-crd.html
[markers]: https://book.kubebuilder.io/reference/markers.html
[kubernetes-ansible-collection]: https://github.com/ansible-collections/kubernetes
