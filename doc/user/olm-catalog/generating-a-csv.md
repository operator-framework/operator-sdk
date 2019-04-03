# Generating a Cluster Service Version (CSV)

This document describes how to manage the following lifecycle for your Operator using the SDK's [`operator-sdk olm-catalog gen-csv`][doc-gen-csv] command:

- **Generate your first release** - encapsulate the metadata needed to install your Operator and configure the permissions it needs from the generated SDK files.
- **Upgrade your Operator** - Carry over any customizations you have made and ensure a rolling update to the next version of your Operator.
- **Refresh your CRDs** - If a new version has updated CRDs, refresh those definitions within the CSV automatically.

## Configuration

Operator SDK projects have an expected [project layout][doc-project-layout]. In particular, a few manifests are expected to be present in the `deploy` directory:

* Roles: `role.yaml`
* Deployments: `operator.yaml`
* Custom Resources (CR's): `crds/<group>_<version>_<kind>_cr.yaml`
* Custom Resource Definitions (CRD's): `crds/<group>_<version>_<kind>_crd.yaml`.

`gen-csv` reads these files and adds their data to a CSV in an alternate form.

By default, a `deploy/olm-catalog/csv-config.yaml` file is generated when `gen-csv` is first run. The defaults written in the following fields contain paths to the aforementioned files. From the [design doc][doc-csv-design]:

>Users can configure CSV composition by populating several fields in the file `deploy/olm-catalog/csv-config.yaml`:
>
>- `crd-cr-path-list`: (string(, string)\*) a list of CRD and CR manifest file/directory paths. Defaults to `[deploy/crds]`.
>- `operator-path`: (string) the operator resource manifest file path. Defaults to `deploy/operator.yaml`.
>- `rbac-path-list`: (string(, string)\*) a list of RBAC role manifest file paths. Defaults to `[deploy/role.yaml]`.

Fields in this config file can be modified to point towards alternate manifest locations. For example, if I have one set of production CR/CRD manifests under `deploy/crds/production`, and a set of test manifests under `deploy/crds/test`, and I only want to include production manifests in my CSV, I can set `crd-cr-path-list: [deploy/crds/production]`. `gen-csv` will then ignore `deploy/crds/test` when getting CR/CRD data.

## Versioning

CSV's are versioned in path, file name, and in their `metadata.name` field. For example, running `operator-sdk olm-catalog gen-csv --csv-version 0.0.1` will generate a CSV at `deploy/olm-catalog/<operator-name>/0.0.1/<operator-name>.v0.0.1.clusterserviceversion.yaml`. A versioned directory such as `deploy/olm-catalog/<operator-name>/0.0.1` is known as a [*bundle*][doc-bundle]. Versions allow the OLM to upgrade or downgrade your Operator at runtime, i.e. in a cluster. A valid semantic version is required.

`gen-csv` allows you to upgrade your CSV using the `--from-version` flag. If you have an existing CSV with version `0.0.1` and want to write a new version `0.0.2`, you can run `operator-sdk olm-catalog gen-csv --csv-version 0.0.2 --from-version 0.0.1`. This will write a new CSV manifest to `deploy/olm-catalog/<operator-name>/0.0.2/<operator-name>.v0.0.2.clusterserviceversion.yaml` containing user-defined data from `0.0.1` and any modifications you've made to `roles.yaml`, `operator.yaml`, CR's, or CRD's.

The SDK can manage CRD's in your Operator bundle as well. You can pass the `--update-crds` flag to `gen-csv` to add or update your CRD's in your bundle by copying manifests pointed to by `crd-cr-path-list` in your config. CRD's in a bundle are not updated by default.

## First Generation

Now that you've configured the generator, assuming version `0.0.1` is being generated, running `operator-sdk olm-catalog gen-csv --csv-version 0.0.1` will generate a CSV defining your Operator under `deploy/olm-catalog/<operator-name>/0.0.1/<operator-name>.v0.0.1.clusterserviceversion.yaml`. No CSV existed previously in `deploy/olm-catalog/<operator-name>/0.0.1`, so no manifests were overwritten or modified.

Some fields might not have values after running `gen-csv` the first time. The SDK will warn you to fill required fields and make suggestions for values for other fields:

```console
$ operator-sdk olm-catalog gen-csv --csv-version 0.0.1
INFO[0000] Generating CSV manifest version 0.0.1        
INFO[0000] Required csv fields not filled in file deploy/olm-catalog/app-operator/0.0.1/app-operator.v0.0.1.clusterserviceversion.yaml:
	spec.keywords
	spec.maintainers
	spec.provider
INFO[0000] Created deploy/olm-catalog/app-operator/0.0.1/app-operator.v0.0.1.clusterserviceversion.yaml
```

When running `gen-csv` with a version that already exists, the `Required csv fields...` info statement will become a warning, as these fields are useful for displaying your Operator in Operator Hub.

A note on `specDescriptors` and `statusDescriptors` fields in `spec.customresourcedefinitions.owned`:
* Code comments are parsed to create `description`'s for each item in `specDescriptors` and `statusDescriptors`, so these comments should be kept up-to-date with Operator semantics.
* `displayName` is guessed from type names, but will not overwrite values already present.
* `path` and `x-descriptors` are guessed from JSON tags and their corresponding UI element from [this list][x-desc-list]. These values are presented as suggestions by `gen-csv` if they are not filled.

## Updating your CSV

Let's say you added a new CRD `deploy/crds/group_v1beta1_yourkind_crd.yaml` to your Operator project, and added a port to your Deployment manifest `operator.yaml`. Assuming you're using the same version as above, updating your CSV is as simple as running `operator-sdk olm-catalog gen-csv --csv-version 0.0.1`. `gen-csv` will append your new CRD to `spec.customresourcedefinitions.owned`, replace the old data at `spec.install.spec.deployments` with your updated Deployment, and write an updated CSV to the same location.

The SDK will not overwrite user-defined fields like `spec.maintainers` or descriptions of CSV elements, with the exception of `specDescriptors[].displayName` and `statusDescriptors[].displayName` in `spec.customresourcedefinitions.owned`, as mentioned [above](#first-generation).

Including the `--update-crds` flag will update the CRD's in your Operator bundle.

## Upgrading your CSV

New versions of your CSV are created by running `operator-sdk gen-csv --csv-version <new-version> --from-version <old-version>`. Running this command will copy user-defined fields from the old CSV to the new CSV and make updates to the new version if any manifest data was changed. This command fills the `spec.replaces` field with the old CSV versions' name.

Be sure to include the `--update-crds` flag if you want to add CRD's to your bundle alongside your CSV.

## CSV fields

Below are two lists of fields: the first is a list of all fields the SDK and OLM expect in a CSV, and the second are optional.

Several fields require user input. The set of fields with this requirement may change as the SDK becomes better at generating CSV's. For now, those are marked with a `(user)` qualifier.

Required:

* `metadata.name`: a *unique* name for this CSV. Operator version should be included in the name to ensure uniqueness, ex. `app-operator.v0.1.1`.
* `spec.description` (user): a thorough description of the Operator's functionality.
* `spec.displayName` (user): a name to display for the Operator in Operator Hub.
* `spec.keywords`: (user) a list of keywords describing the Operator.
* `spec.maintainers`: (user) a list of human or organizational entities maintaining the Operator, with a `name` and `email`.
* `spec.provider`: (user) the Operator provider, with a `name`; usually an organization.
* `spec.labels`: (user) a list of `key:value` pairs to be used by Operator internals.
* `spec.version`: semantic version of the Operator, ex. `0.1.1`.
* `spec.installModes`: what mode of [installation namespacing][install-modes] OLM should use. Currently all but `MultiNamespace` are supported by SDK Operators.
* `spec.customresourcedefinitions`: any CRD's the Operator uses. This field will be filled by the SDK if any CRD manifests pointed to by `crd-cr-path-list` in your config.
  * `description`: description of the CRD.
  * `resources`: any Kubernetes resources used by the CRD, ex. `Pod`'s and `ConfigMap`'s.
  * `specDescriptors`: UI hints for inputs and outputs of the Operator's spec.
  * `statusDescriptors`: UI hints for inputs and outputs of the Operator's status.

Optional:

* `metadata.annotations.alm-examples`: CR examples, in JSON string literal format, for your CRD's. Ideally one per CRD.
* `metadata.annotations.capabilities`: level of Operator capability. See the [Operator maturity model][olm-capabilities] for a list of valid values.
* `spec.replaces`: the name of the CSV being replaced by this CSV.
* `spec.links`: (user) a list of URL's to websites, documentation, etc. pertaining to the Operator or application being managed, each with a `name` and `url`.
* `spec.selector`: (user) selectors by which the Operator can pair resources in a cluster.
* `spec.icon`: (user) a base64-encoded icon unique to the Operator, set in a `base64data` field with a `mediatype`.
* `spec.maturity`: the Operator's maturity, ex. `alpha`.

## Further Reading

* [Information][doc-csv] on what goes in your CSV and CSV semantics.
* The original `gen-csv` [design doc][doc-csv-design], which contains a thorough description how CSV's are generated by the SDK.

[doc-csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md
[olm]:https://github.com/operator-framework/operator-lifecycle-manager
[doc-gen-csv]:../../sdk-cli-reference.md#gen-csv
[doc-project-layout]:../../project_layout.md
[doc-csv-design]:../../design/milestone-0.2.0/csv-generation.md
[doc-bundle]:https://github.com/operator-framework/operator-registry#manifest-format
[x-desc-list]:https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L5-L14
[install-modes]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md#operator-metadata
[olm-capabilities]:../../images/operator-maturity-model.png
