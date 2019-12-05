# Migrating Existing Kubernetes APIs

Kubernetes APIs are assumed to evolve over time, hence the well-defined API [versioning scheme][k8s-versioning]. Upgrading your operator's APIs can be a non-trivial task, one that will involve changing quite a few source files and manifests. This document aims to identify the complexities of migrating an operator project's API using examples from existing operators.

While examples in this guide follow particular types of API migrations, most of the documented migration steps can be generalized to all migration types. A thorough discussion of migration types for a particular project type (Go, Ansible, Helm) is found at the end of each project type's section.

## Go: Upgrading one Kind to a new Version from a Version with multiple Kinds

**Scenario:** your Go operator test-operator has one API version `v1` for group `operators.example.com`. You would like to migrate (upgrade) one kind `CatalogSourceConfig` to `v2` while keeping the other `v1` kind `OperatorGroup` in `v1`. These kinds will remain in group `operators.example.com`. Your project structure looks like the following:

```console
$ tree pkg/apis
pkg/apis/
├── addtoscheme_operators_v1.go
├── apis.go
└── operators
    └── v1
        ├── catalogsourceconfig_types.go
        ├── catalogsourceconfig.go
        ├── doc.go
        ├── operatorgroup_types.go
        ├── operatorgroup.go
        ├── phase.go
        ├── phase_types.go
        ├── register.go
        ├── shared.go
        ├── zz_generated.deepcopy.go
```

Relevant files:

- `catalogsourceconfig_types.go` and `catalogsourceconfig.go` contain types and functions used by API kind type `CatalogSourceConfig`.
- `operatorgroup_types.go` and `operatorgroup.go` contain types and functions used by API kind type `OperatorGroup`.
- `phase_types.go` and `phase.go` contain types and functions used by *non-API* type `Phase`, which is used by both `CatalogSourceConfig` and `OperatorGroup` types.
- `shared.go` contain types and functions used by both `CatalogSourceConfig` and `OperatorGroup` types.

#### Questions to ask yourself
1. **Scope:** what files, Go source and YAML, must I modify when migrating?
1. **Shared code:** do I have shared types and functions between `CatalogSourceConfig` and `OperatorGroup`? How do I want shared code refactored?
1. **Imports:** which packages import those I am migrating? How do I modify these packages to import `v2` and new shared package(s)?
1. **Backwards-compatibility:** do I want to remove code being migrated from `v1` entirely, forcing the use of `v2`, or support both `v1` and `v2` going forward?  

---

### Creating a new API Version

Creating the new version `v2` is the first step in upgrading your kind `CatalogSourceConfig`. Use the `operator-sdk` to do so by running the following command:

```console
$ operator-sdk add api --api-version operators.example.com/v2 --kind CatalogSourceConfig
```

This command creates a new API version `v2` under group `operators`:

```console
$ tree pkg/apis
pkg/apis/
├── addtoscheme_operators_v1.go
├── addtoscheme_operators_v2.go           # new addtoscheme source file for v2
├── apis.go
└── operators
    └── v1
    |   ├── catalogsourceconfig_types.go
    |   ├── catalogsourceconfig.go
    |   ├── doc.go
    |   ├── operatorgroup_types.go
    |   ├── operatorgroup.go
    |   ├── phase.go
    |   ├── phase_types.go
    |   ├── register.go
    |   ├── shared.go
    |   ├── zz_generated.deepcopy.go
    └── v2                                # new version dir with source files for v2
        ├── catalogsourceconfig_types.go
        ├── doc.go
        ├── register.go
        ├── zz_generated.deepcopy.go
```

In addition to creating a new API version, the command creates an `addtoscheme_operators_v2.go` file that exposes an `AddToScheme()` function for registering `v2.CatalogSourceConfig` and `v2.CatalogSourceConfigList`.

### Copying shared type definitions and functions to a separate package

Now that the `v2` package and related files exist, we can begin moving types and functions around. First, we must copy anything shared between `CatalogSourceConfig` and `OperatorGroup` to a separate package that can be imported by `v1`, `v2`, and future versions. We've identified the files containing these types above: `phase.go`, `phase_types.go`, and `shared.go`.

#### Creating a new `shared` package

Lets create a new package `shared` at `pkg/apis/operators/shared` for these files:

```console
$ pwd
/home/user/projects/test-operator
$ mkdir pkg/apis/operators/shared
```

This package is not a typical API because it contains types only to be used as parts of larger schema, and therefore should not be created with `operator-sdk add api`. It should contain a `doc.go` file with some package-level documentation and annotations:

```console
$ cat > pkg/apis/operators/shared/doc.go <<EOF
// +k8s:deepcopy-gen=package,register

// Package shared contains types and functions used by API definitions in the
// operators package
// +groupName=operators.example.com
package shared
EOF
```

Global annotations necessary for using `shared` types in API type fields:

- `+k8s:deepcopy-gen=package,register`: directs [`deepcopy-gen`][deepcopy-gen] to generate `DeepCopy()` functions for all types in the `shared` package.
- `+groupName=operators.example.com`: defines the fully qualified API group name for [`client-gen`][client-gen]. Note: this annotation *must* be on the line above `package shared`.

Lastly, if you have any comments in `pkg/apis/operators/v1/doc.go` related to copied source code, ensure they are copied into `pkg/apis/operators/shared/doc.go`. Now that `shared` is a standalone library, more comments explaining what types and functions exist in the package and how they are intended to be used should be added.

**Note:** you may have helper functions or types you do not want to publicly expose, but are required by functions or types in `shared`. If so, create a `pkg/apis/operators/internal/shared` package:

```console
$ pwd
/home/user/projects/test-operator
$ mkdir pkg/apis/operators/internal/shared
```

This package does not need a `doc.go` file as described above.

#### Copying types to package `shared`

The three files containing shared code (`phase.go`, `phase_types.go`, and `shared.go`) can be copied _almost_ as-is from `v1` to `shared`. The only changes necessary are:

- Changing the package statements in each file: `package v1` -> `package shared`.
- Moving and exporting currently unexported (private) types, their methods, and functions used by `v1` types to `pkg/apis/operators/internal/shared/shared.go`. Exported them in an internal shared package will keep them private while allowing functions and types in `shared` to use them.

Additionally, `deepcopy-gen` must be run on the `shared` package to generate `DeepCopy()` and `DeepCopyInto()` methods, which are necessary for all Kubernetes API types. To do so, run the following command:

```console
$ operator-sdk generate k8s
```

Now that shared types and functions have their own package we can update any package that imports those types from `v1` to use `shared`. The `CatalogSourceConfig` controller source file `pkg/controller/catalogsourceconfig/catalogsourceconfig_controller.go` imports and uses a type defined in `v1`, `PhaseRunning`, in its `Reconcile()` method. `PhaseRunning` should be imported from `shared` as follows:

```Go
import (
  "context"

  operatorsv1 "github.com/test-org/test-operator/pkg/apis/operators/v1"
  // New import
  "github.com/test-org/test-operator/pkg/apis/operators/shared"

  "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

...

func (r *ReconcileCatalogSourceConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
  ...

  config := &operatorsv1.CatalogSourceConfig{}
  err := r.client.Get(context.TODO(), request.NamespacedName, config)
  if err != nil {
    ...
  }
  // Old
  if config.Status.CurrentPhase.Phase.Name != operatorsv1.PhaseRunning {
    ...
  }
  // New
  if config.Status.CurrentPhase.Phase.Name != shared.PhaseRunning {
    ...
  }
}
```

Do this for all instances of types previously in `v1` that are now in `shared`.

Following Kubernetes API version upgrade conventions, code moved to `shared` from `v1` should be marked with "Deprecated" comments in `v1` instead of being removed. While leaving these types in `v1` duplicates code, it allows backwards compatibility for API users; deprecation comments direct users to switch to `v2` and `shared` types.

Alternatively, types and functions migrated to `shared` can be removed in `v1` to de-duplicate code. This breaks backwards compatibility because projects relying on exported types previously in `v1`, now in `shared`, will be forced to update their imports to use `shared` when upgrading VCS versions. If following this upgrade path, note that updating package import paths in your project will likely be the most pervasive change lines-of-code-wise in this process. Luckily the Go compiler will tell you which import path's you have missed once `CatalogSourceConfig` types are removed from `v1`!

If any functions or types were moved to `pkg/apis/operator/internal/shared`, remove them from files in `pkg/apis/operator/shared` and import them into `shared` from the internal package.

### Updating empty `v2` types using `v1` types

The `CatalogSourceConfig` type and schema code were generated by `operator-sdk add api`, but the types are not populated. We need to copy existing type data from `v1` to `v2`. This process is similar to migrating shared code, except we do not need to export any types or functions.

Remove `pkg/apis/operators/v2/catalogsourceconfig_types.go` and copy `catalogsourceconfig.go` and `catalogsourceconfig_types.go` from `pkg/apis/operators/v1` to `pkg/apis/operators/v2`:

```console
$ rm pkg/apis/operators/v2/catalogsourceconfig_types.go
$ cp pkg/apis/operators/v1/catalogsourceconfig*.go pkg/apis/operators/v2
```

If you have any comments or custom code in `pkg/apis/operators/v1` related to source code in either copied file, ensure that is copied to `doc.go` or `register.go` in `pkg/apis/operators/v2`.

You can now run `operator-sdk generate k8s` to generate deepcopy code for the migrated `v2` types. Once this is done, update all packages that import the migrated `v1` types to use those in `v2`.

### Updating CustomResourceDefinition manifests and generating OpenAPI code

Now that we've migrated all Go types to their destination packages, we must update the corresponding CustomResourceDefinition (CRD) manifests in `deploy/crds`.

Doing so can be as simple as running the following command:

```console
$ operator-sdk generate crds
```

This command will automatically update all CRD manifests.

#### CRD Versioning

<!-- TODO: change SDK version to the last release before controller-tools v0.2.0 refactor -->

Kubernetes 1.11+ supports CRD [`spec.versions`][crd-versions] and `spec.version` is [deprecated][crd-version-deprecated] as of Kubernetes 1.12. SDK versions `v0.10.x` and below leverage [`controller-tools`][controller-tools]' CRD generator `v0.1.x` which generates a now-deprecated `spec.version` value based on the version contained in an APIs import path. Names of CRD manifest files generated by those SDK versions contain the `spec.version`, i.e. one CRD manifest is created *per version in a group* with the form `<group>_<version>_<kind>_crd.yaml`. The SDK is in the process of upgrading to `controller-tools` `v0.2.x`, which generates `spec.versions` but not `spec.version` by default. Once the upgrade is complete, future SDK versions will place all versions in a group in `spec.versions`. File names will then have the format `<full_group>_<resource>_crd.yaml`.

**Note**: `<full group>` is the full group name of your CRD while `<group>` is the last subdomain of `<full group>`, ex. `foo.bar.com` vs `foo`. `<resource>` is the plural lower-case of CRD `Kind` specified at `spec.names.plural`.

**Note:** If your operator does not have custom data manually added to its CRD's, you can skip to the [following section](#golang-api-migrations-types-and-commonalities); `operator-sdk generate crds` will handle CRD updates in that case.

Upgrading from `spec.version` to `spec.versions` will be demonstrated using the following CRD manifest example:

`deploy/crds/operators_v1_catalogsourceconfig_crd.yaml`:
```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: catalogsourceconfigs.operators.coreos.com
spec:
  group: operators.coreos.com
  names:
    kind: CatalogSourceConfig
    listKind: CatalogSourceConfigList
    plural: catalogsourceconfigs
    singular: catalogsourceconfig
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            size:
              format: int32
              type: integer
            test:
              type: string
          required:
          - size
          type: object
        status:
          properties:
            nodes:
              items:
                type: string
              type: array
          required:
          - nodes
          type: object
  version: v1
  subresources:
    status: {}
```

Steps to upgrade the above CRD:

1. Rename your CRD manifest file from `deploy/crds/operators_v1_catalogsourceconfig_crd.yaml` to `deploy/crds/operators.coreos.com_catalogsourceconfigs_crd.yaml`

    ```console
    $ mv deploy/crds/cache_v1alpha1_memcached_crd.yaml deploy/crds/operators.coreos.com_catalogsourceconfigs_crd.yaml
    ```

1. Create a `spec.versions` list that contains two elements for each version that now exists (`v1` and `v2`):

    ```yaml
    spec:
      ...
      # version is now v2, as it must match the first element in versions.
      version: v2
      versions:
      - name: v2
        # Set to true for this CRD version to be enabled in-cluster.
        served: true
        # Exactly one CRD version should be a storage version.
        storage: true
      - name: v1
        served: true
        storage: false
    ```

    The first version in `spec.versions` *must* match that in `spec.version` if `spec.version` exists in the manifest.

1. *Optional:* `spec.versions` elements have a `schema` field that holds a version-specific OpenAPIV3 validation block to override the global `spec.validation` block. `spec.validation` will be used by the API server to validate one or more versions in `spec.versions` that do not have a `schema` block. If all versions have the same schema, leave `spec.validation` as-is and skip to the [following section](#golang-api-migrations-types-and-commonalities). If your CRD versions differ in scheme, copy `spec.validation` YAML to the `schema` field in each `spec.versions` element, then modify as needed:

    ```yaml
    spec:
      ...
      version: v2
      versions:
      - name: v2
        served: true
        storage: true
        schema: # v2-specific OpenAPIV3 validation block.
          openAPIV3Schema:
            properties:
              apiVersion:
                type: string
          ...
      - name: v1
        served: true
        storage: false
        schema: # v1-specific OpenAPIV3 validation block.
          openAPIV3Schema:
            properties:
              apiVersion:
                type: string
          ...
    ```

    The API server will validate each version by its own `schema` if the global `spec.validation` block is removed. No validation will be performed if a `schema` does not exist for a version and `spec.validation` does not exist.

    If the CRD targets a Kubernetes 1.13+ cluster with the `CustomResourceWebhookConversion` feature enabled, converting between multiple versions can be done using a [conversion][crd-conv]. The `None` conversion is simple and useful when the CRD spec has not changed; it only updates the `apiVersion` field of custom resources:

    ```yaml
    spec:
      ...
      conversion:
        strategy: None
    ```

    More complex conversions can be done using [conversion webhooks][crd-conv-webhook].

    _TODO:_ document adding and using conversion webhooks to migrate `v1` to `v2` once the SDK `controller-runtime` version is bumped to `v0.2.0`.

    **Note:** read the [CRD versioning][crd-versions] docs for detailed CRD information, notes on conversion webhooks, and CRD versioning case studies.

1. *Optional:* `spec.versions` elements have a `subresources` field that holds CR subresource information to override the global `spec.subresources` block. `spec.subresources` will be used by the API server to assess subresource requirements of any version in `spec.versions` that does not have a `subresources` block. If all versions have the same requirements, leave `spec.subresources` as-is and skip to the [following section](#golang-api-migrations-types-and-commonalities). If CRD versions differ in subresource requirements, add a `subresources` section in each `spec.versions` entry with differing requirements and add each subresource's spec and status as needed:

    ```yaml
    spec:
      ...
      version: v2
      versions:
      - name: v2
        served: true
        storage: true
        subresources:
          ...
      - name: v1
        served: true
        storage: false
        subresources:
          ...
    ```

    Remove the global `spec.subresources` block if all versions have different subresource requirements.

1. *Optional:* remove `spec.version`, as it is deprecated in favor of `spec.versions`.

### Go API Migrations: Types and Commonalities

This version upgrade walkthrough demonstrates only one of several possible migration scenarios:

- Group migration, ex. moving an API from group `operators.example.com/v1` to `new-group.example.com/v1alpha1`.
- Kind change, ex. `CatalogSourceConfig` to `CatalogSourceConfigurer`.
- Some combination of group, version, and kind migration.

Each case is different; one may require many more changes than others. However, there are several themes common to all:

1. Using `operator-sdk add api` to create the necessary directory structure and files used in migration.
    - Group migration using the same version, for each kind in the old group `operators.example.com` you want to migrate:

      ```console
      $ operator-sdk add api --api-version new-group.example.com/v1 --kind YourKind
      ```

    - Kind migration, using the same group and version as `CatalogSourceConfig`:

      ```console
      $ operator-sdk add api --api-version operators.example.com/v1 --kind CatalogSourceConfigurer
      ```

1. Copying code from one Go package to another, ex. from `v1` to `v2` and `shared`.
1. Changing import paths in project Go source files to those of new packages.
1. Updating CRD manifests.
    - In many cases, having sufficient [code annotations][kubebuilder-api-annotations] and running `operator-sdk generate crds` will be enough.

The Go toolchain can be your friend here too. Running `go vet ./...` can tell you what import paths require changing and what type instantiations are using fields incorrectly.

## Helm

TODO

## Ansible

TODO

[k8s-versioning]:https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
[deepcopy-gen]:https://godoc.org/k8s.io/gengo/examples/deepcopy-gen
[client-gen]:https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/generating-clientset.md
[controller-tools]:https://github.com/kubernetes-sigs/controller-tools
[crd-versions]:https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/
[crd-conv]:https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#webhook-conversion
[crd-conv-webhook]:https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#configure-customresourcedefinition-to-use-conversion-webhooks
[kubebuilder-api-annotations]:https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
[crd-version-deprecated]:https://github.com/kubernetes/apiextensions-apiserver/commit/d1c6536f26319513417b12245c6e3aee5ca005ca
