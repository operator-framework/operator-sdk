---
title: Generating Manifests and Metadata
linkTitle: Generating Manifests and Metadata
weight: 20
---

This document describes how to manage packaging and shipping your Operator in the following stages:

* **Generate your first release** - encapsulate the metadata needed to install your Operator with the
[Operator Lifecycle Manager][olm] and configure the permissions it needs from the generated SDK files.
* **Update your Operator** - apply any updates to Operator manifests made during development.
* **Upgrade your Operator** - carry over any customizations you have made and ensure a rolling update to the
next version of your Operator.

## Overview

Several `operator-sdk` subcommands manage operator-framework manifests and metadata,
in particular [`ClusterServiceVersion`'s (CSVs)][doc-csv], for an Operator: [`generate bundle`][cli-gen-bundle],
[`generate packagemanifests`][cli-gen-packagemanifests], and [`generate kustomize manifests`][cli-gen-kustomize-manifests].
See this [CLI overview][cli-overview] for details on each command. These manifests come in two different formats,
[bundle][bundle] and [package manifests][package-manifests], which are described in detail below.
Ideally the bundle format should be used as this is the default packaging format in operator-framework.
However the package manifests format is still supported by operator-framework tooling.

### Kustomize files

`operator-sdk generate kustomize manifests` generates a CSV kustomize base
`config/manifests/bases/<project-name>.clusterserviceversion.yaml` and a `config/manifests/bases/kustomization.yaml`
by default. These files are required as `kustomize build` input in downstream commands.

By default, the command starts an interactive prompt if a CSV base in `config/manifests/bases` is not present
to collect [UI metadata](#csv-fields). You can disable the interactive prompt by passing `--kustomize=false`.

```console
$ operator-sdk generate kustomize manifests
INFO[0000] Generating CSV manifest version 0.1.0

Display name for the operator (required):
> memcached

Comma-separated list of keywords for your operator (required):
> app, operator
...
```

**For Go Operators only:** the command parses [CSV markers][csv-markers] from Go API type definitions, located
in `./api` for single group projects and `./apis` for multigroup projects, to populate certain CSV fields.
You can set an alternative path to the API types root directory with `--apis-dir`. These markers are not available
to Ansible or Helm project types.

### ClusterServiceVersion manifests

CSV's are manifests that define all aspects of an Operator, from what CustomResourceDefinitions (CRDs) it uses to
metadata describing the Operator's maintainers. They are typically versioned by semver, much like Operator projects
themselves; this version is present in both their `metadata.name` and `spec.version` fields. The CSV generator called
by `generate <bundle|packagemanifests>` requires certain input manifests to construct a CSV manifest; all inputs
are read when either command is invoked, along with a CSV's [base](#kustomize-files), to idempotently regenerate a CSV.

The following resource kinds are typically included in a CSV, which are addressed by `config/manifests/bases/kustomization.yaml`:
  - `Role`: define Operator permissions within a namespace.
  - `ClusterRole`: define cluster-wide Operator permissions.
  - `Deployment`: define how the Operator's operand is run in pods.
  - `CustomResourceDefinition`: definitions of custom objects your Operator reconciles.
  - Custom resource examples: examples of objects adhering to the spec of a particular CRD.

## Generate your first release

You've recently run `operator-sdk init` and created your APIs with `operator-sdk create api`. Now you'd like to
package your Operator for deployment by OLM. Your Operator is at version `v0.0.1`; the `Makefile` variable `VERSION`
should be set to `0.0.1`. You've also built your operator image, `quay.io/<user>/memcached-operator:v0.0.1`.

### Bundle format

A [bundle][bundle] consists of manifests (CSV and CRDs) and metadata that define an Operator
at a particular version. You may have also heard of a bundle image. From the bundle docs:

> An Operator Bundle is built as a scratch (non-runnable) container image that
> contains operator manifests and specific metadata in designated directories
> inside the image. Then, it can be pushed and pulled from an OCI-compliant
> container registry. Ultimately, an operator bundle will be used by Operator
> Registry and OLM to install an operator in OLM-enabled clusters.

At this stage in your Operator's development, we only need to worry about generating bundle files;
bundle images become important once you're ready to [publish][operatorhub] your Operator.

SDK projects are scaffolded with a `Makefile` containing the `bundle` recipe by default,
which wraps `generate kustomize manifests`, `generate bundle`, and other related commands.

By default `make bundle` will generate a CSV, copy CRDs, and generate metadata in the bundle format:

```console
$ make bundle
$ tree ./bundle
./bundle
├── manifests
│   ├── cache.my.domain_memcacheds.yaml
│   └── memcached-operator.clusterserviceversion.yaml
└── metadata
    └── annotations.yaml
```

Bundle metadata in `bundle/metadata/annotations.yaml` contains information about a particular Operator version
available in a registry. OLM uses this information to install specific Operator versions and resolve dependencies.
That file and `bundle.Dockerfile` contain the same [annotations][bundle-metadata], the latter as `LABEL`s,
which do not need to be modified in most cases; if you do decide to modify them, both sets of annotations _must_
be the same to ensure consistent Operator deployment.

##### Channels

Metadata for each bundle contains channel information as well:

> Channels allow package authors to write different upgrade paths for different users (e.g. beta vs. stable).

Channels become important when publishing, but we should still be aware of them beforehand as they're required
values in our metadata. `make bundle` writes the channel `alpha` by default.

### Package manifests format

A [package manifests][package-manifests] format consists of on-disk manifests (CSV and CRDs) and metadata that
define an Operator at all versions of that Operator. Each version is contained in its own directory, with a parent
package manifest YAML file containing channel-to-version mappings, much like a bundle's metadata.

If your Operator is already formatted as a package manifests and you do not wish to migrate to the bundle format yet,
you should add the following to your `Makefile` to make development easier:

**For Go-based Operator projects**

```make
# Options for "packagemanifests".
ifneq ($(origin FROM_VERSION), undefined)
PKG_FROM_VERSION := --from-version=$(FROM_VERSION)
endif
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Generate package manifests.
packagemanifests: kustomize manifests
  operator-sdk generate kustomize manifests -q
  cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
  $(KUSTOMIZE) build config/manifests | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
```

**For Helm/Ansible-based Operator projects**

```make
# Options for "packagemanifests".
ifneq ($(origin FROM_VERSION), undefined)
PKG_FROM_VERSION := --from-version=$(FROM_VERSION)
endif
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Generate package manifests.
packagemanifests: kustomize
  operator-sdk generate kustomize manifests -q
  cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
  $(KUSTOMIZE) build config/manifests | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
```

By default `make packagemanifests` will generate a CSV, a package manifest file, and copy CRDs in the package manifests format:

```console
$ make packagemanifests IMG=quay.io/<user>/memcached-operator:v0.0.1
$ tree ./packagemanifests
./packagemanifests
├── 0.0.1
│   ├── cache.my.domain_memcacheds.yaml
│   └── memcached-operator.clusterserviceversion.yaml
└── memcached-operator.package.yaml
```

## Update your Operator

Let's say you added a new API `App` with group `app` and version `v1alpha1` to your Operator project,
and added a port to your manager Deployment in `config/manager/manager.yaml`.

If using a bundle format, the current version of your CSV can be updated by running:

```console
$ make bundle IMG=quay.io/<user>/memcached-operator:v0.0.1
```

If using a package manifests format, run:

```console
$ make packagemanifests IMG=quay.io/<user>/memcached-operator:v0.0.1
```

Running the command for either format will append your new CRD to `spec.customresourcedefinitions.owned`,
replace the old data at `spec.install.spec.deployments` with your updated Deployment,
and update your existing CSV manifest. The SDK will not overwrite [user-defined](#csv-fields)
fields like `spec.maintainers`.

## Upgrade your Operator

Let's say you're upgrading your Operator to version `v0.0.2`, you've already updated the `VERSION` variable
in your `Makefile` to `0.0.2`, and built a new operator image `quay.io/<user>/memcached-operator:v0.0.2`.
You also want to add a new channel `beta`, and use it as the default channel.

If using a bundle format, a new version of your CSV can be created by running:

```console
$ make bundle CHANNELS=beta DEFAULT_CHANNEL=beta IMG=quay.io/<user>/memcached-operator:v0.0.2
```

If using a package manifests format, run:

```console
$ make packagemanifests FROM_VERSION=0.0.1 CHANNEL=beta IS_CHANNEL_DEFAULT=1 IMG=quay.io/<user>/memcached-operator:v0.0.2
```

Running the command for either format will persist user-defined fields, updates `spec.version`,
and populates `spec.replaces` with the old CSV version's name.

## CSV fields

Below are two lists of fields: the first is a list of all fields the SDK and OLM expect in a CSV, and the second are optional.

**For Go Operators only:** Several fields require user input (labeled _user_) or a [CSV marker][csv-markers]
(labeled _marker_). This list may change as the SDK becomes better at generating CSV's.
These markers are not available to Ansible or Helm project types.

Required:
- `metadata.name`: a *unique* name for this CSV of the format `<project-name>.vX.Y.Z`, ex. `app-operator.v0.0.1`.
- `spec.version`: semantic version of the Operator, ex. `0.0.1`.
- `spec.installModes`: what mode of [installation namespacing][install-modes] OLM should use.
Currently all but `MultiNamespace` are supported by SDK Operators.
- `spec.customresourcedefinitions`: any CRDs the Operator uses. Certain fields in elements of `owned` will be filled by the SDK.
    - `owned`: all CRDs the Operator deploys itself from it's bundle.
        - `name`: CRD's `metadata.name`.
        - `kind`: CRD's `metadata.spec.names.kind`.
        - `version`: CRD's `metadata.spec.version`.
        - `description` _(marker)_ : description of the CRD.
        - `displayName` _(marker)_ : display name of the CRD.
        - `resources` _(marker)_ : any Kubernetes resources used by the CRD, ex. `Pod`'s and `ConfigMap`'s.
        - `specDescriptors` _(marker)_ : UI hints for inputs and outputs of the Operator's spec.
        - `statusDescriptors` _(marker)_ : UI hints for inputs and outputs of the Operator's status.
        - `actionDescriptors` _(user)_ : UI hints for an Operator's in-cluster actions.
    - `required` _(user)_ : all CRDs the Operator expects to be present in-cluster, if any.
    All `required` element fields must be populated manually.

Optional:
- `spec.description` _(user)_ : a thorough description of the Operator's functionality.
- `spec.displayName` _(user)_ : a name to display for the Operator in Operator Hub.
- `spec.keywords` _(user)_ : a list of keywords describing the Operator.
- `spec.maintainers` _(user)_ : a list of human or organizational entities maintaining the Operator, with a `name` and `email`.
- `spec.provider` _(user)_ : the Operator provider, with a `name`; usually an organization.
- `spec.labels` _(user)_ : a list of `key:value` pairs to be used by Operator internals.
- `metadata.annotations.alm-examples`: CR examples, in JSON string literal format, for your CRD's. Ideally one per CRD.
- `metadata.annotations.capabilities`: level of Operator capability. See the [Operator maturity model][olm-capabilities]
for a list of valid values.
- `spec.replaces`: the name of the CSV being replaced by this CSV.
- `spec.links` _(user)_ : a list of URL's to websites, documentation, etc. pertaining to the Operator or application
being managed, each with a `name` and `url`.
- `spec.selector` _(user)_ : selectors by which the Operator can pair resources in a cluster.
- `spec.icon` _(user)_ : a base64-encoded icon unique to the Operator, set in a `base64data` field with a `mediatype`.
- `spec.maturity`: the Operator's maturity, ex. `alpha`.


[olm]:https://github.com/operator-framework/operator-lifecycle-manager
[doc-csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/0.15.1/doc/design/building-your-csv.md
[cli-overview]:/docs/olm-integration/cli-overview
[cli-gen-kustomize-manifests]:/docs/cli/operator-sdk_generate_kustomize_manifests
[cli-gen-bundle]:/docs/cli/operator-sdk_generate_bundle
[cli-gen-packagemanifests]:/docs/cli/operator-sdk_generate_packagemanifests
[bundle]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md
[bundle-metadata]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md#bundle-annotations
[package-manifests]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[install-modes]:https://github.com/operator-framework/operator-lifecycle-manager/blob/4197455/Documentation/design/building-your-csv.md#operator-metadata
[olm-capabilities]:/docs/advanced-topics/operator-capabilities/operator-capabilities
[csv-markers]:/docs/building-operators/golang/references/markers
[operatorhub]:https://operatorhub.io/
