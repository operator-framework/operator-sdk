---
title: Generating a ClusterServiceVersion
linkTitle: Generating ClusterServiceVersion's
weight: 20
---

This document describes how to manage the following lifecycle for your Operator using the SDK's [`operator-sdk generate csv`][generate-csv-cli] command:

- **Generate your first release** - encapsulate the metadata needed to install your Operator with the [Operator Lifecycle Manager][olm] and configure the permissions it needs from the generated SDK files.
- **Update your Operator** - apply any updates to Operator manifests made during development.
- **Upgrade your Operator** - carry over any customizations you have made and ensure a rolling update to the next version of your Operator.

**Note:** `operator-sdk generate csv` only officially supports Go Operators. Ansible and Helm Operators will be fully supported in the future. However, `generate csv` _may_ work with Ansible and Helm Operators if their project structure aligns with that described below.

## Configuration

### Inputs

The [ClusterServiceVersion (CSV)][doc-csv] generator requires certain input
manifests to construct a CSV manifest. Each of these inputs are read every time
`operator-sdk generate csv` is run are used to overwrite data in corresponding
CSV fields (with one exception described [below](#csv-fields)).

1. Path to the Operator manifests root directory. By default `generate csv` extracts manifests from files in `deploy/` for the following kinds and adds them to the CSV. Use the `--deploy-dir` flag to change this path.
    * Roles: `deploy/role.yaml`
    * ClusterRoles: `deploy/cluster_role.yaml`
    * Deployments: `deploy/operator.yaml`
2. Path to the API types root directory. The CSV generator also parses the [CSV annotations][csv-annotations] from the API type definitions to populate certain CSV fields. By default the API types directory is `pkg/apis/`. Use the `--apis-dir` flag to change this path. The CSV generator expects either of the following layouyts for the API types directory.
    * Mulitple groups: `<API-root-dir>/<group>/<version>/`
    * Single groups: `<API-root-dir>/<version>/`
3. Path to a directory containing CustomResourceDefinition's (CRD's) and Custom Resource examples (CR's).
Use the `--crd-dir` flag to change this path.
    * Custom Resources: `deploy/crds/<full group>_<version>_<kind>_cr.yaml`
    * CustomResourceDefinitions: `deploy/crds/<full group>_<resource>_crd.yaml`

### Output

By default `generate csv` will create a [*bundle*][doc-bundle] directory
`deploy/olm-catalog/<operator-name>/manifests` containing your CSV and CRDs.
To change where the CSV bundle directory is generated use the `--ouput-dir` flag.

## Creating a new CSV

CSV's are versioned by their `metadata.name` and `spec.version` fields and stored
in bundle directories. To create a CSV for version `0.0.1`, run:

```console
$ operator-sdk generate csv --csv-version 0.0.1
```

A CSV should now exist at `deploy/olm-catalog/<operator-name>/manifests/<operator-name>.clusterserviceversion.yaml`
with `<operator-name>.v0.0.1` and version `0.0.1`. This command will also copy all `CustomResourceDefinition`
manifests from `deploy/crds` or the value passed to `--crd-dir` to that CSV's directory.
Note that a valid semantic version is required.

**Note:** `generate csv` will populate many but not all fields in your CSV
automatically. Subsequent calls to `generate csv` will warn you of missing
required fields. See the list of fields [below](#csv-fields) for more information.

#### Deprecated behavior

_This behavior is deprecated. If you have versioned bundle directories, consider
removing them and using default behavior_

Setting `--make-manifests=false` will create a CSV in a versioned directory
`deploy/olm-catalog/<operator-name>/0.0.1/<operator-name>.v0.0.1.clusterserviceversion.yaml`,
If a versioned bundle directory exists on disk, `generate csv` allows you to
upgrade your CSV from that version using the `--from-version` flag. Example: if you
have an existing CSV with version `0.0.1` and want to write a new version `0.0.2`, you can run:

```console
$ operator-sdk generate csv --csv-version 0.0.2 --from-version 0.0.1
```

This will write a new CSV manifest to `deploy/olm-catalog/<operator-name>/0.0.2/<operator-name>.v0.0.2.clusterserviceversion.yaml`
containing user-defined data from `0.0.1` and any modifications you've made to
the configured inputs.

Setting `--update-crds=false` prevents the generator from updating bundled CRD manifests.

## Updating your CSV

Let's say you added a new CRD `deploy/crds/group.domain.com_resource_crd.yaml`
to your Operator project, and added a port to your Deployment manifest `operator.yaml`.
Assuming you're using the same version as above, updating your CSV is as simple
as running `operator-sdk generate csv`. Doing so will append your new CRD to
`spec.customresourcedefinitions.owned`, replace the old data at `spec.install.spec.deployments`
with your updated Deployment, and update your existing CSV manifest.

The SDK will not overwrite [user-defined](#csv-fields) fields like `spec.maintainers`.

## Upgrading your CSV

New versions of your CSV are created by running `operator-sdk generate csv --csv-version <new-version>`.
Doing will persist user-defined fields, updates `spec.version`,
and populates `spec.replaces` with the old CSV versions' name.

## CSV fields

Below are two lists of fields: the first is a list of all fields the SDK and OLM expect in a CSV, and the second are optional.

Several fields require user input (labeled _user_) or a [CSV annotation][csv-annotations] (labeled _annotation_). This list may change as the SDK becomes better at generating CSV's.

Required:

* `metadata.name`: a *unique* name for this CSV. Operator version should be included in the name to ensure uniqueness, ex. `app-operator.v0.1.1`.
* `spec.description` _(user)_ : a thorough description of the Operator's functionality.
* `spec.displayName` _(user)_ : a name to display for the Operator in Operator Hub.
* `spec.keywords` _(user)_ : a list of keywords describing the Operator.
* `spec.maintainers` _(user)_ : a list of human or organizational entities maintaining the Operator, with a `name` and `email`.
* `spec.provider` _(user)_ : the Operator provider, with a `name`; usually an organization.
* `spec.labels` _(user)_ : a list of `key:value` pairs to be used by Operator internals.
* `spec.version`: semantic version of the Operator, ex. `0.1.1`.
* `spec.installModes`: what mode of [installation namespacing][install-modes] OLM should use. Currently all but `MultiNamespace` are supported by SDK Operators.
* `spec.customresourcedefinitions`: any CRDs the Operator uses. Certain fields in elements of `owned` will be filled by the SDK.
    * `owned`: all CRDs the Operator deploys itself from it's bundle.
        * `name`: CRD's `metadata.name`.
        * `kind`: CRD's `metadata.spec.names.kind`.
        * `version`: CRD's `metadata.spec.version`.
        * `description` _(annotation)_ : description of the CRD.
        * `displayName` _(annotation)_ : display name of the CRD.
        * `resources` _(annotation)_ : any Kubernetes resources used by the CRD, ex. `Pod`'s and `ConfigMap`'s.
        * `specDescriptors` _(annotation)_ : UI hints for inputs and outputs of the Operator's spec.
        * `statusDescriptors` _(annotation)_ : UI hints for inputs and outputs of the Operator's status.
        * `actionDescriptors` _(user)_ : UI hints for an Operator's in-cluster actions.
    * `required` _(user)_ : all CRDs the Operator expects to be present in-cluster, if any. All `required` element fields must be populated manually.

Optional:

* `metadata.annotations.alm-examples`: CR examples, in JSON string literal format, for your CRD's. Ideally one per CRD.
* `metadata.annotations.capabilities`: level of Operator capability. See the [Operator maturity model][olm-capabilities] for a list of valid values.
* `spec.replaces`: the name of the CSV being replaced by this CSV.
* `spec.links` _(user)_ : a list of URL's to websites, documentation, etc. pertaining to the Operator or application being managed, each with a `name` and `url`.
* `spec.selector` _(user)_ : selectors by which the Operator can pair resources in a cluster.
* `spec.icon` _(user)_ : a base64-encoded icon unique to the Operator, set in a `base64data` field with a `mediatype`.
* `spec.maturity`: the Operator's maturity, ex. `alpha`.

## Further Reading

* [Information][doc-csv] on what goes in your CSV and CSV semantics.
* The original [design doc][doc-csv-design], which contains a thorough description how CSV's are generated by the SDK.

[doc-csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/4197455/Documentation/design/building-your-csv.md
[olm]:https://github.com/operator-framework/operator-lifecycle-manager
[generate-csv-cli]:../../../cli/operator-sdk_generate_csv
[doc-csv-design]:https://github.com/operator-framework/operator-sdk/blob/master/doc/design/milestone-0.2.0/csv-generation.md
[doc-bundle]:https://github.com/operator-framework/operator-registry/blob/6893d19/README.md#manifest-format
[install-modes]:https://github.com/operator-framework/operator-lifecycle-manager/blob/4197455/Documentation/design/building-your-csv.md#operator-metadata
[olm-capabilities]:../../images/operator-capability-level.png
[csv-annotations]:./csv-annotations.md
