# Operator-SDK CSV generation Design Doc

## Goal

The `operator-sdk build` sub-command should generate a [**Cluster Service Version (CSV)**][olm-csv-definition] customized using information contained in user-defined yaml manifests and operator source files. Operator developers, *users*, should be able to run `operator-sdk build` and have their operators' state encapsulated in a CSV; this command should be idempotent and only update the CSV when a yaml manifest or source file is changed. Users should not have to directly interact with a CSV, and should not be required to have in-depth CSV knowledge in order for their operator to interact with [**Operator Lifecycle Manager (OLM)**][olm-description] or publish metadata to the [**Catalog**][catalog-description].

## Background

CSV's are yaml manifests created from a users' operator metadata that assist the OLM in running their operator in a cluster:

> A CSV is the metadata that accompanies your Operator container image. It can be used to populate user interfaces with info like your logo/description/version and it is also a source of technical information needed to run the Operator, like the RBAC rules it requires and which Custom Resources it manages or depends on.

The `operator-sdk generate olm-catalog` command currently produces a generic CSV with minimal customization. Defaults and simple metadata components are used to fill fields, with no options to customize. Users must modify the CSV manually or use custom scripts to pull data from various operator component files. These solutions are not scalable because we cannot assume users are, or have time to become, familiar with CSV format or know where to find information required by CSV yaml objects.

## Proposed Solution

Functionality of `operator-sdk generate olm-catalog` is now a branch in `operator-sdk build`, and the former command no longer exists. `operator-sdk build` writes to the `deploy` directory, which is the standard location for all manifests and scripts required to deploy an operator. The SDK can use data from manifests in `deploy` to write a CSV.

`operator-sdk build` will call the function `renderCSV` to either:

1. Create a new CSV, with the same location and naming convention as exists currently, using available data in yaml manifests and source files.

    1. `renderCSV` will check for an existing CSV in `deploy`. Upon not finding one, a [`ClusterServiceVersion` object][olm-csv-struct-code], defined by the OLM, is created and fields easily derived from operator metadata, such as Kubernetes API `ObjectMeta`, are populated.
    1. `renderCSV` will search `deploy` for manifests that contain yaml objects a CSV uses, such as a `Deployment` Kubernetes API resource, and set the appropriate CSV fields with object data.
    1. Once the search completes, every object field populated will be written back to the CSV yaml file.
		- Note that individual yaml objects are overwritten and not the entire file, as descriptions and other non-generated parts of a CSV should be preserved.

1. Update an existing CSV at the currently pre-defined location, using available data in yaml manifests and source files.
    
    1. `renderCSV` will check for an existing CSV in `deploy`. Upon finding one, the CSV yaml file contents will be marshalled into a `ClusterServiceVersion` object.
    1. Same as above.
    1. Same as above.
    
### Extensible `CSVUpdater` CSV update mechanism

The CSV spec will likely change over time as new Kubernetes and OLM features are implemented; we need the ability to easily extend the update system. The SDK will define the `CSVUpdater` interface as follows to encapsulate individual CSV field updates in methods:

```Go
import "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

type CSVUpdater interface {
	Apply(*v1alpha1.ClusterServiceVersion) error
}
```

`Apply` will use data from the `CSVUpdater` implementer to operate on `*v1alpha1.ClusterServiceVersion` object fields relevant to that updater. The OLM defines the entire CSV spec [in a Golang struct][olm-csv-spec-code] the SDK can alias to implement `CSVUpdater`s.

Once sub-step two is reached when creating or updating a CSV, `renderCSV` will extract each yaml document discovered, and pass document data into a dispatcher function. The dispatcher selects the correct `CSVUpdater` to call based on the documents' `Kind` yaml object, a manifest type identifier used in all operator manifests. A CSV should reflect the current state of an operators' yaml manifests and any codified components in general, so any fields that correspond to data gathered from codified components will be overwritten; data like English descriptions will not be unless updated data is found.

The following is an example implementation of an [install strategy][olm-csv-install-strat-doc] `CSVUpdater`:

```Go
import (
  ...
  appsv1 "k8s.io/api/apps/v1"
  rbacv1beta1 "k8s.io/api/rbac/v1beta1"
  "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
  "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
)

// CSVInstallStrategyUpdate embeds the OLM's install strategy spec.
type CSVInstallStrategyUpdate struct {
	*install.StrategyDetailsDeployment
  
  // Future utility fields go here.
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

### User-defined yaml objects

Many CSV component objects cannot be populated using generated, non-SDK-specific manifests. These objects are mostly human-written, English metadata about the operator and various CRD's. Users must directly modify their CSV yaml file, adding personalized data to the following required objects. A lack of data in any of the required objects will generate an error on `operator-sdk build`.

Required:
- `displayName`: a public name to identify the operator.
- `description`: a short description of the operator's functionality.
- `keywords`: 1..N keywords describing the operator.
- `maintainers`: 1..N human or organizational entities maintaining the operator, with a `name` and `email`.
- `provider`: the operators' provider, with a `name`; usually an organization.
- `labels`: 1..N `key`:`value` pairs to be used by operator internals.
- `version`: semantic version of the operator, ex. `0.1.1`.
- `customresourcedefinitions`: any CRD's the operator uses.
	- **Note**: this field will be populated automatically by the SDKif any CRD yaml files are present in `deploy`; however, several objects require user input (more details in the [CSV CRD spec section][olm-csv-crd-doc]):
		- `description`: description of the CRD.
		- `resources`: any Kubernetes resources leveraged by the CRD, ex. `Pod`'s, `StatefulSet`'s.
		- `specDescriptors`: UI hints for inputs and outputs of the operator.

Optional:
- `replaces`: the CSV being replaced by this CSV.
- `links`: 1..N URL's to websites, documentation, etc. pertaining to the operator or application being managed, each with a `name` and `url`.
- `selector`: selectors by which the operator can pair resources in a cluster.
- `icon`: a base64-encoded icon unique to the operator, set in a `base64data` object with a `mediatype`. 
- `maturity`: the operators' stage of development, ex. `beta`.

Further details on what data each field above should hold are found in the [CSV spec][olm-csv-spec-doc].

**Note**: Several yaml objects currently requiring user intervention can potentially be parsed from operator code; such SDK functionality will be addressed in a future design document.


[olm-csv-definition]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#what-is-a-cluster-service-version-csv
[olm-description]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/README.md
[catalog-description]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/architecture.md#catalog-registry-design
[olm-csv-struct-code]:https://github.com/operator-framework/operator-lifecycle-manager/blob/8799f39ef342dc1ff7430eba7a88c1c3c70cbdcc/pkg/api/apis/operators/v1alpha1/clusterserviceversion_types.go#L261
[olm-csv-spec-code]:https://github.com/operator-framework/operator-lifecycle-manager/blob/8799f39ef342dc1ff7430eba7a88c1c3c70cbdcc/pkg/api/apis/operators/v1alpha1/clusterserviceversion_types.go
[olm-csv-spec-doc]:https://github.com/operator-framework/operator-lifecycle-manager/blob/16ff8f983b50503c4d8b8015bd0c14b5c7d6786a/Documentation/design/building-your-csv.md#building-a-cluster-service-version-csv-for-the-operator-framework
[olm-csv-install-strat-doc]:https://github.com/operator-framework/operator-lifecycle-manager/blob/16ff8f983b50503c4d8b8015bd0c14b5c7d6786a/Documentation/design/building-your-csv.md#operator-install
[olm-csv-crd-doc]:https://github.com/operator-framework/operator-lifecycle-manager/blob/16ff8f983b50503c4d8b8015bd0c14b5c7d6786a/Documentation/design/building-your-csv.md#owned-crds