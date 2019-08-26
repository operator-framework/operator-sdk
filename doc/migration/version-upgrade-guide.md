# Version upgrade guide

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
    version = "=v0.4.1"
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



[print-deps-cli]: ../sdk-cli-reference.md#print-deps
[changelog]: ../../CHANGELOG.md
[release-notes]: https://github.com/operator-framework/operator-sdk/releases
[v0.1.0-migration-guide]: ./v0.1.0-migration-guide.md
[manifest-format]: https://github.com/operator-framework/operator-registry#manifest-format