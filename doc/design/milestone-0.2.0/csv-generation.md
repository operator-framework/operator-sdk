# Operator-SDK CSV generation Design Doc

## Goal

The `operator-sdk olm-catalog gen-csv` sub-command will generate a [**Cluster Service Version (CSV)**][olm_csv_definition] customized using information contained in user-defined yaml manifests and operator source files. Operator developers, *users*, will run `operator-sdk olm-catalog gen-csv` with the `--csv-version $version` flag to have their operators' state encapsulated in a CSV with the supplied version; this action should be idempotent and only update the CSV file when a new version is supplied, or a yaml manifest or source file is changed. Users should not have to directly modify most fields in a CSV manifest. Those that require modification are defined [below](#user-defined-yaml-fields). A CSV-generating command removes the responsibility from users of having in-depth [**Operator Lifecycle Manager (OLM)**][olm_description] knowledge in order for their operator to interact with OLM or publish metadata to the [**Catalog**][catalog_description].

## Background

CSV's are yaml manifests created from a users' operator metadata that assist the OLM in running their operator in a cluster:

> A CSV is the metadata that accompanies your Operator container image. It can be used to populate user interfaces with info like your logo/description/version and it is also a source of technical information needed to run the Operator, like the RBAC rules it requires and which Custom Resources it manages or depends on.

The `operator-sdk generate olm-catalog` command currently produces a generic CSV with minimal customization. Defaults and simple metadata components are used to fill fields, with no options to customize. Users must modify the CSV manually or use custom scripts to pull data from various operator component files. These solutions are not scalable because we cannot assume users are, or have time to become, familiar with CSV format or know where to find information required by CSV yaml fields.

## Proposed Solution

Functionality of `operator-sdk generate olm-catalog` is now in `operator-sdk olm-catalog gen-csv`; the former command no longer exists. `operator-sdk olm-catalog gen-csv --csv-version 0.0.1` writes a CSV yaml file to the `deploy/olm-catalog` directory by default.

`deploy` is the standard location for all manifests required to deploy an operator. The SDK can use data from manifests in `deploy` to write a CSV. Exactly three types of manifests are required to generate a CSV: `operator.yaml`, `*_{crd,cr}.yaml`, and RBAC role files, ex. `role.yaml`. Users may have different versioning requirements for these files and can configure CSV which specific files are included in `deploy/olm-catalog/csv-config.yaml`, described [below](#configuration).

Assuming all configuration defaults are used, `operator-sdk olm-catalog gen-csv` will call `scaffold.Execute()`, which will either:

1. Create a new CSV, with the same location and naming convention as exists currently, using available data in yaml manifests and source files.

    1. The update mechanism will check for an existing CSV in `deploy`. Upon not finding one, a [`ClusterServiceVersion` object][olm_csv_struct_code], referred to here as a *cache*, is created and fields easily derived from operator metadata, such as Kubernetes API `ObjectMeta`, are populated.
    1. The update mechanism will search `deploy` for manifests that contain data a CSV uses, such as a `Deployment` Kubernetes API resource, and set the appropriate CSV fields in the cache with this data.
    1. Once the search completes, every cache field populated will be written back to a CSV yaml file.
        - **Note:** individual yaml fields are overwritten and not the entire file, as descriptions and other non-generated parts of a CSV should be preserved.

1. Update an existing CSV at the currently pre-defined location, using available data in yaml manifests and source files.

    1. The update mechanism will check for an existing CSV in `deploy`. Upon finding one, the CSV yaml file contents will be marshalled into a `ClusterServiceVersion` cache.
    1. The update mechanism will search `deploy` for manifests that contain data a CSV uses, such as a `Deployment` Kubernetes API resource, and set the appropriate CSV fields in the cache with this data.
    1. Once the search completes, every cache field populated will be written back to a CSV yaml file.
        - **Note:** individual yaml fields are overwritten and not the entire file, as descriptions and other non-generated parts of a CSV should be preserved.

### Configuration

Users can configure CSV composition by populating several fields in the file `deploy/olm-catalog/csv-config.yaml`:

- `operator-path`: (string) the operator resource manifest file path. Defaults to `deploy/operator.yaml`.
- `crd-cr-path-list`: (string(, string)\*) a list of CRD and CR manifest file paths. Defaults to `[deploy/crds/*_{crd,cr}.yaml]`.
- `rbac-path-list`: (string(, string)\*) a list of RBAC role manifest file paths. Defaults to `[deploy/role.yaml]`.

### Extensible `CSVUpdater` CSV update mechanism

The CSV spec will likely change over time as new Kubernetes and OLM features are implemented; we need the ability to easily extend the update system. The SDK will define the `CSVUpdater` interface as follows to encapsulate individual CSV field updates in methods:

```Go
import "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

type CSVUpdater interface {
  Apply(*v1alpha1.ClusterServiceVersion) error
}
```

`Apply` will use data from the `CSVUpdater` implementer to operate on `*v1alpha1.ClusterServiceVersion` cache fields relevant to that updater. The OLM defines the entire CSV spec [in a Golang struct][olm_csv_spec_code] the SDK can alias to implement `CSVUpdater`s.

Once sub-step two is reached when creating or updating a CSV, `renderCSV` will extract each yaml document discovered, and pass document data into a dispatcher function. The dispatcher selects the correct `CSVUpdater` to call based on the documents' `Kind` field, a manifest type identifier used in all operator manifests. A CSV should reflect the current state of an operators' yaml manifests and any codified components in general, so any fields that correspond to data gathered from codified components will be overwritten; data like English descriptions will not be unless updated data is found.

The following is an example implementation of an [install strategy][olm_csv_install_strat_doc] `CSVUpdater`:

```Go
import (
  appsv1 "k8s.io/api/apps/v1"
  rbacv1beta1 "k8s.io/api/rbac/v1beta1"
  "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
  "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
)

// CSVInstallStrategyUpdate embeds the OLM's install strategy spec.
type CSVInstallStrategyUpdate struct {
  *install.StrategyDetailsDeployment

  // Future fields go here.
}

// getLocalInstallStrategyCache retrieves the local cache singleton and returns the install strategy cache.
func getLocalInstallStrategyCache() *CSVInstallStrategyUpdate {
  factory := getLocalCacheFactory()
  return factory.InstallStrategyCache
}

// AddDeploymentSpecToCSVInstallStrategyUpdate adds an RBAC Role to the local cache singletons' permissions.
func AddRoleToCSVInstallStrategyUpdate(yamlDoc []byte) error {
  localInstallStrategyUpdate := getLocalInstallStrategyCache()

  newRBACRole := new(rbacv1beta1.Role)
  _ = yaml.Unmarshal(yamlDoc, newRBACRole)

  newPermissions := install.StrategyDeploymentPermissions{
    ServiceAccountName: newRole.ObjectMeta.Name,
    Rules:              newRole.Rules,
  }
  localInstallStrategyUpdate.Permissions = append(localInstallStrategyUpdate.Permissions, newPermissions)

  return nil
}

// AddDeploymentSpecToCSVInstallStrategyUpdate adds a Deployment to the local cache singletons' install strategy.
func AddDeploymentSpecToCSVInstallStrategyUpdate(yamlDoc []byte) error {
  localInstallStrategyUpdate := getLocalInstallStrategyCache()

  newDeployment := new(appsv1.Deployment)
  _ = yaml.Unmarshal(yamlDoc, newDeployment)

  newDeploymentSpec := install.StrategyDeploymentSpec{
    Name: newDeployment.ObjectMeta.Name,
    Spec: newDeployment.Spec,
  }
  localInstallStrategyUpdate.DeploymentSpecs = append(localInstallStrategyUpdate.DeploymentSpecs, newDeploymentSpec)

  return nil
}

// Apply applies cached updates in CSVInstallStrategyUpdate to the appropriate csv fields.
func (us *CSVInstallStrategyUpdate) Apply(csv *v1alpha1.ClusterServiceVersion) error {
  // Get install strategy from csv.
  var resolver *install.StrategyResolver
  strat, _ := resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
  installStrat, _ := strat.(*install.StrategyDetailsDeployment)

  // Update permissions and deployments with custom field update methods.
  us.updatePermissions(installStrat)
  us.updateDeploymentSpecs(installStrat)

  // Re-serialize permissions into csv install strategy.
  updatedStrat, _ := json.Marshal(installStrat)
  csv.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

  return nil
}
```

### User-defined yaml fields

Many CSV fields cannot be populated using generated, non-SDK-specific manifests. These fields are mostly human-written, English metadata about the operator and various CRD's. Users must directly modify their CSV yaml file, adding personalized data to the following required fields. Users will receive a warning from `operator-sdk olm-catalog gen-csv` when a lack of data in any of the required fields is detected.

Required:

- `metadata.name`: a *unique* name for this CSV. Operator version should be included in the name to ensure uniqueness, ex. `app-operator.v0.1.1`.
- `spec.displayName`: a public name to identify the operator.
- `spec.description`: a short description of the operator's functionality.
- `spec.keywords`: 1..N keywords describing the operator.
- `spec.maintainers`: 1..N human or organizational entities maintaining the operator, with a `name` and `email`.
- `spec.provider`: the operators' provider, with a `name`; usually an organization.
- `spec.labels`: 1..N `key`:`value` pairs to be used by operator internals.
- `spec.version`: semantic version of the operator, ex. `0.1.1`.
- `spec.customresourcedefinitions`: any CRD's the operator uses. This field will be populated automatically by the SDK if any CRD yaml files are present in `deploy`; however, several fields not in the CRD manifest spec that require user input (more details in the [CSV CRD spec section][olm_csv_crd_doc]):
        - `description`: description of the CRD.
        - `resources`: any Kubernetes resources leveraged by the CRD, ex. `Pod`'s, `StatefulSet`'s.
        - `specDescriptors`: UI hints for inputs and outputs of the operator.

Optional:

- `spec.replaces`: the name of the CSV being replaced by this CSV.
- `spec.links`: 1..N URL's to websites, documentation, etc. pertaining to the operator or application being managed, each with a `name` and `url`.
- `spec.selector`: selectors by which the operator can pair resources in a cluster.
- `spec.icon`: a base64-encoded icon unique to the operator, set in a `base64data` field with a `mediatype`.
- `spec.maturity`: the operators' stability, ex. `beta`.

Further details on what data each field above should hold are found in the [CSV spec][olm_csv_spec_doc].

**Note**: Several yaml fields currently requiring user intervention can potentially be parsed from operator code; such SDK functionality will be addressed in a future design document.

### CSV versioning

The CSV version is the same as the operators', and should be included somewhere in `metadata.name`. A new CSV will be generated when upgrading operator versions.

**TODO:** discuss whether multiple CSV files can be present, each with a unique file name (ex. `app-operator.csv.0.1.1.yaml`), or a single `app-operator.csv.yaml` file that relies on VCS (git) to version the file.

[olm_csv_definition]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#what-is-a-cluster-service-version-csv
[olm_description]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/README.md
[catalog_description]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/architecture.md#catalog-registry-design
[olm_csv_struct_code]:https://github.com/operator-framework/operator-lifecycle-manager/blob/8799f39ef342dc1ff7430eba7a88c1c3c70cbdcc/pkg/api/apis/operators/v1alpha1/clusterserviceversion_types.go#L261
[olm_csv_spec_code]:https://github.com/operator-framework/operator-lifecycle-manager/blob/8799f39ef342dc1ff7430eba7a88c1c3c70cbdcc/pkg/api/apis/operators/v1alpha1/clusterserviceversion_types.go
[olm_csv_spec_doc]:https://github.com/operator-framework/operator-lifecycle-manager/blob/16ff8f983b50503c4d8b8015bd0c14b5c7d6786a/Documentation/design/building-your-csv.md#building-a-cluster-service-version-csv-for-the-operator-framework
[olm_csv_install_strat_doc]:https://github.com/operator-framework/operator-lifecycle-manager/blob/16ff8f983b50503c4d8b8015bd0c14b5c7d6786a/Documentation/design/building-your-csv.md#operator-install
[olm_csv_crd_doc]:https://github.com/operator-framework/operator-lifecycle-manager/blob/16ff8f983b50503c4d8b8015bd0c14b5c7d6786a/Documentation/design/building-your-csv.md#owned-crds
