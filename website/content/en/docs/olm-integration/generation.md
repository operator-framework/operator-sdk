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
in particular [`ClusterServiceVersion`'s (CSVs)][doc-csv], for an Operator: [`generate bundle`][cli-gen-bundle] and [`generate kustomize manifests`][cli-gen-kustomize-manifests].
See this [CLI overview][cli-overview] for details on each command.

**Note:** The packagemanifests format is deprecated and support will be removed in `operator-sdk` v2.0.0.

### Kustomize files

`operator-sdk generate kustomize manifests` generates a CSV kustomize base
`config/manifests/bases/<project-name>.clusterserviceversion.yaml` and a `config/manifests/kustomization.yaml`
by default. These files are required as `kustomize build` input in downstream commands.

By default, the command starts an interactive prompt if a CSV base in `config/manifests/bases` is not present
to collect [UI metadata](#csv-fields). You can disable the interactive prompt by passing `--interactive=false`.

```console
$ operator-sdk generate kustomize manifests
INFO[0000] Generating CSV manifest version 0.1.0

Display name for the operator (required):
> memcached

Comma-separated list of keywords for your operator (required):
> app, operator
...
```

Once this base is written, you may modify any of the fields labeled _user_ in the [fields section](#csv-fields) below.
These values will persist when generating a bundle, so make necessary metadata changes here and not the generated bundle.

**For Go Operators only:** the command parses [CSV markers][csv-markers] from Go API type definitions, located
in `./api` for single group projects and `./apis` for multigroup projects, to populate certain CSV fields.
You can set an alternative path to the API types root directory with `--apis-dir`. These markers are not available
to Ansible or Helm project types. 

The command attempts to process the local types defined in your API.
If you import a package that uses the same name as a local type, running the command causes an infinite loop. For example:
```go
type PodStatus struct {
  SomeField string
  // imported type with the same name will infinitely trigger
  // the parser to process the local PodStatus type
  Status v1.PodStatus 
}
```
To prevent an infinite loop, edit the local type definition to use a different name. For example:
```go
type PodStatusWrapper struct {
  SomeField string
  Status v1.PodStatus 
}
```

### ClusterServiceVersion manifests

CSV's are manifests that define all aspects of an Operator, from what CustomResourceDefinitions (CRDs) it uses to
metadata describing the Operator's maintainers. They are typically versioned by semver, much like Operator projects
themselves; this version is present in both their `metadata.name` and `spec.version` fields. The CSV generator called
by `generate <bundle|packagemanifests>` requires certain input manifests to construct a CSV manifest; all inputs
are read when either command is invoked, along with a CSV's [base](#kustomize-files), to idempotently regenerate a CSV.

The following resource kinds are typically included in a CSV, which are addressed by `config/manifests/kustomization.yaml`:
  - `Role`: define Operator permissions within a namespace.
  - `ClusterRole`: define cluster-wide Operator permissions.
  - `Deployment`: define how the Operator's operand is run in pods.
  - `ValidatingWebhookConfiguration`, `MutatingWebhookConfiguration`: configures webhooks for your manager to handle.
  - `CustomResourceDefinition`: definitions of custom objects your Operator reconciles.
  - Custom resource examples: examples of objects adhering to the spec of a particular CRD.

You can optionally specify an input `ClusterServiceVersion` manifest to the set of manifests passed to
these `generate` subcommands instead of having them read from the [base path](#kustomize-files).
This is advantageous for those who would like to take full advantage of `kustomize` for their base.
All fields unlabeled or labeled with _marker_ [below](#csv-fields) will be overwritten by these command,
so make sure you do not `kustomize build` those fields!

#### Webhooks

A CSV allows you to [define][olm-whs] both [admission][doc-admission-whs] and [conversion][doc-conv-whs] webhooks
at [`spec.webhookdefinitions`][wh-defs]. The `generate <bundle|packagemanifests>` commands, described below,
will automatically add webhooks to your CSV if the following holds true:
1. A webhook configuration must be associated with a `Service` by name and namespace,
whether in a [CRD][crd-wh-serviceref] or in a [`*WebhookConfiguration`][wh-serviceref] file,
1. The associated `Service` must expose one `spec.ports[*].targetPort` that matches both `containerPort`
and `protocol` of one element in the Operator `Deployment`'s `spec.template.spec.containers[*].ports`.

By default, the manager's Deployment is configured to mount a volume containing TLS cert data
created by [cert-manager][cert-manager] into the manager's container.
OLM does [not yet support cert-manager][olm-cert-support], so a [JSON patch][cm-patch] was added
to remove this volume and mount such that OLM can itself create and manage certs for your Operator.

**Note (for Go Operators only):** If targeting OLM < v0.17.0, the manager's default webhook server
is not configured with the correct cert/key paths; the correct path is
`/apiserver.local.config/certificates/apiserver.{cert,key}`.
To cover this case, make the following changes to your `main.go`:

```go
import (
  ...
  ctrl "sigs.k8s.io/controller-runtime"
  "sigs.k8s.io/controller-runtime/pkg/webhook"
)

func main() {
  ...

  // Configure a webhook.Server with the correct path and file names.
  // If webhookServer is nil, which will be the case of OLM >= 0.17 is available,
  // the manager will create a server for you using Host, Port,
  // and the default CertDir, KeyName, and CertName.
  var webhookServer *webhook.Server
  const legacyOLMCertDir = "/apiserver.local.config/certificates"
  if info, err := os.Stat(legacyOLMCertDir); err == nil && info.IsDir() {
    webhookServer = &webhook.Server{
      Host:     <some host>, // Set this only if normally set in ctrl.Options below.
      Port:     <some port>, // Set this only if normally set in ctrl.Options below.
      CertDir:  legacyOLMCertDir,
      CertName: "apiserver.crt",
      KeyName:  "apiserver.key",
    }
  }

  mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Host:          <some host>,
    Port:          <some port>,
    WebhookServer: webhookServer, // Host/Port will not be used if webhookServer is nil.
  })
 
  // Now you can register webhooks.
  ...
}
```

**Note:** The `Service` itself will still be placed into the `manifests/` directory,
in case other Operator resources require routing. Feel free to remove it otherwise.


[olm-whs]:https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks
[doc-admission-whs]:https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
[doc-conv-whs]:https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion
[wh-defs]:https://pkg.go.dev/github.com/operator-framework/api/pkg/operators/v1alpha1#WebhookDefinition
[crd-wh-serviceref]:https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1?utm_source=godoc#ServiceReference
[wh-serviceref]:https://pkg.go.dev/k8s.io/api/admissionregistration/v1?utm_source=godoc#ServiceReference
[olm-cert-support]:https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/#certificate-authority-requirements
[cm-patch]:https://github.com/operator-framework/operator-sdk/blob/163c657/testdata/go/v3/memcached-operator/config/manifests/kustomization.yaml#L12

## Generate your first release

You've recently run `operator-sdk init` and created your APIs with `operator-sdk create api`. Now you'd like to
package your Operator for deployment by OLM. Your Operator is at version `v0.0.1`; the `Makefile` variable `VERSION`
should be set to `0.0.1`. You've also built your operator image, `example.com/memcached-operator:v0.0.1`;
if this image tag does not match yours, swap in the correct one in the docs below.

### Bundle format

A [bundle][bundle] consists of manifests (CSV, CRDs, and other supported kinds) and metadata that define an Operator
at a particular version, and an optional [scorecard][scorecard] configuration file. You may have also heard of a
bundle image. From the bundle docs:

> An Operator Bundle is built as a scratch (non-runnable) container image that
> contains operator manifests and specific metadata in designated directories
> inside the image. Then, it can be pushed and pulled from an OCI-compliant
> container registry. Ultimately, an operator bundle will be used by Operator
> Registry and OLM to install an operator in OLM-enabled clusters.

At this stage in your Operator's development, we only need to worry about generating bundle files;
bundle images become important once you're ready to [publish][operatorhub] your Operator.

SDK projects are scaffolded with a `Makefile` containing the `bundle` recipe by default,
which wraps `generate kustomize manifests`, `generate bundle`, and other related commands.

By default `make bundle` will generate a CSV, copy CRDs and other supported kinds, generate metadata,
and add your scorecard configuration in the bundle format:

```console
$ make bundle
$ tree ./bundle
./bundle
├── manifests
│   ├── cache.example.com_memcacheds.yaml
│   ├── memcached-operator.clusterserviceversion.yaml
│   ├── memcached-operator-controller-manager-metrics-monitor_monitoring.coreos.com_v1_servicemonitor.yaml
│   ├── memcached-operator-controller-manager-metrics-service_v1_service.yaml
│   ├── memcached-operator-metrics-reader_rbac.authorization.k8s.io_v1beta1_clusterrole.yaml
│   └── memcached-operator-webhook-service_v1_service.yaml
├── metadata
│   └── annotations.yaml
└── tests
    └── scorecard
        └── config.yaml
```

**Important:** bundle generation is supposed to be idempotent, so any changes to CSV fields able to be persisted
(marked _(user)_ or _(marker)_ [below](#csv-fields)) must be made to the base set of manifests, typically found in `config/`.

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

#### Validation

The `bundle` recipe includes a call to `operator-sdk bundle validate`, which runs a set of required object
validators on your bundle that ensure both its format and content meet the [bundle specification][bundle].
These will always be run and cannot be disabled.

You may also have added [CSV fields](#csv-fields) containing useful UI metadata for cluster console display,
and want to ensure that metadata matches some hosted catalog's submission requirements.
The `bundle validate` command supports optional validators that can validate these bundle metadata.
These validators are disabled by default, and can be selectively enabled with `--select-optional <label-selector>`.
You can list all available optional validators by setting the `--list-optional` flag:

```console
$ operator-sdk bundle validate --list-optional
NAME           LABELS                                                DESCRIPTION
operatorhub    name=operatorhub                                      OperatorHub.io metadata validation. 
               suite=operatorframework    
community      name=community                                        (stage: alpha) Community Operator bundle validation      
...
```

For example, you want to turn on the `operatorhub` validator shown above so you can publish the `0.0.1` operator
you recently created on [OperatorHub.io][operatorhub]. To do so, you can modify your Makefile's `bundle` recipe
to validate any further changes you make to bundle UI metadata related to OperatorHub requirements:

```make
bundle: ...
  ...
  operator-sdk bundle validate ./bundle --select-optional name=operatorhub
```

Also, see that you can test the bundle against the suite of test to ensure it against all criteria:

```sh 
operator-sdk bundle validate ./bundle --select-optional suite=operatorframework 
```  

**Note**: The `OperatorHub.io` validator in the `operatorframework` optional suite allows you to validate that your manifests can work with a Kubernetes cluster of a particular version using the `k8s-version` optional key value:

```sh 
operator-sdk bundle validate ./bundle --select-optional suite=operatorframework --optional-values=k8s-version=1.22
```

Documentation on optional validators:
- [`operatorhub`][operatorhub_validator]

**Note**: (stage: alpha) The `Community` validator allows you to validate your `bundle.Dockerfile` configuration against its specific criteria using the `image-path` optional key value:

```sh 
operator-sdk bundle validate ./bundle --select-optional name=community --optional-values=image-path=bundle.Dockerfile
```

### Package manifests format

A [package manifests][package-manifests] format consists of on-disk manifests (CSV, CRDs and other supported kinds)
and metadata that define an Operator at all versions of that Operator. Each version is contained in its own directory,
with a parent package manifest YAML file containing channel-to-version mappings, much like a bundle's metadata.

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
PKG_MAN_OPTS ?= $(PKG_FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

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
PKG_MAN_OPTS ?= $(PKG_FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Generate package manifests.
packagemanifests: kustomize
  operator-sdk generate kustomize manifests -q
  cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
  $(KUSTOMIZE) build config/manifests | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
```

By default `make packagemanifests` will generate a CSV, a package manifest file, and copy CRDs in the package manifests format:

```console
$ make packagemanifests IMG=example.com/memcached-operator:v0.0.1
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
$ make bundle IMG=example.com/memcached-operator:v0.0.1
```

If using a package manifests format, run:

```console
$ make packagemanifests IMG=example.com/memcached-operator:v0.0.1
```

Running the command for either format will append your new CRD to `spec.customresourcedefinitions.owned`,
replace the old data at `spec.install.spec.deployments` with your updated Deployment,
and update your existing CSV manifest. The SDK will not overwrite [user-defined](#csv-fields)
fields like `spec.maintainers`.

## Upgrade your Operator

Let's say you're upgrading your Operator to version `v0.0.2`, you've already updated the `VERSION` variable
in your `Makefile` to `0.0.2`, and built a new operator image `example.com/memcached-operator:v0.0.2`.
You also want to add a new channel `beta`, and use it as the default channel.

First, update `spec.replaces` in your [base CSV manifest](#kustomize-files) to the _current_ CSV name.
In this case, the change would look like:

```yaml
spec:
  ...
  replaces: memcached-operator.v0.0.1
```

Next, upgrade your bundle. If using a bundle format, a new version of your CSV can be created by running:

```console
$ make bundle CHANNELS=beta DEFAULT_CHANNEL=beta IMG=example.com/memcached-operator:v0.0.2
```

If using a package manifests format, run:

```console
$ make packagemanifests FROM_VERSION=0.0.1 CHANNEL=beta IS_CHANNEL_DEFAULT=1 IMG=example.com/memcached-operator:v0.0.2
```

Running the command for either format will persist user-defined fields, and updates `spec.version` and `metadata.name`.

**For `packagemanifests` only** The command will also populate `spec.replaces` with the old CSV version's name.

## CSV fields

Below are two lists of fields: the first is a list of all fields the SDK and OLM expect in a CSV, and the second are optional.

**For Go Operators only:** Several fields require user input (labeled _user_) or a [CSV marker][csv-markers]
(labeled _marker_). This list may change as the SDK becomes better at generating CSV's.
These markers are not available to Ansible or Helm project types.

Required:
- `metadata.name` _(user*)_: a *unique* name for this CSV of the format `<project-name>.vX.Y.Z`, ex. `app-operator.v0.0.1`.
- `spec.displayName` _(user)_ : a name to display for the Operator in Operator Hub.
- `spec.version` _(user*)_: semantic version of the Operator, ex. `0.0.1`.
- `spec.installModes` _(user)_: what mode of [installation namespacing][install-modes] OLM should use.
Currently all but `MultiNamespace` are supported by SDK Operators.
- `spec.customresourcedefinitions`: any CRDs the Operator uses. Certain fields in elements of `owned` will be filled by the SDK.
    - `owned`: all CRDs the Operator deploys itself from it's bundle.
        - `name`: CRD's `metadata.name`.
        - `kind`: CRD's `spec.names.kind`.
        - `version`: CRD's `spec.version`.
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
- `spec.keywords` _(user)_ : a list of keywords describing the Operator.
- `spec.maintainers` _(user)_ : a list of human or organizational entities maintaining the Operator, with a `name` and `email`.
- `spec.provider` _(user)_ : the Operator provider, with a `name`; usually an organization.
- `spec.labels` _(user)_ : a list of `key:value` pairs to be used by Operator internals.
- `metadata.annotations.alm-examples`: CR examples, in JSON string literal format, for your CRD's. Ideally one per CRD.
- `metadata.annotations.capabilities`: level of Operator capability. See the [Operator maturity model][olm-capabilities]
for a list of valid values.
- `spec.replaces` _(user)_: the name of the CSV being replaced by this CSV.
- `spec.links` _(user)_ : a list of URL's to websites, documentation, etc. pertaining to the Operator or application
being managed, each with a `name` and `url`.
- `spec.selector` _(user)_ : selectors by which the Operator can pair resources in a cluster.
- `spec.icon` _(user)_ : a base64-encoded icon unique to the Operator, set in a `base64data` field with a `mediatype`.
- `spec.maturity` _(user)_: the Operator's maturity, ex. `alpha`.
- `spec.minKubeVersion` _(user)_: the minimal Kubernetes version supported by the Operator, ex. `1.16.0`.
- `spec.webhookdefinitions`: any webhooks the Operator uses.
- `spec.relatedImages` _(user)_: a list of image tags containing SHA digests [mapped to in-CSV names][relatedimages]
that your Operator might require to perform their functions.
    - To get the correct tag for an image available in some remote registry, run `docker inspect --format='{{range $i, $d := .RepoDigests}}{{$d}}{{"\n"}}{{end}}'`
    and choose the tag for the desired registry.
- `spec.skips` _(user)_: the names of one or more CSVs that should be skipped in a catalog's upgrade graph.

**\*** `metadata.name` and `spec.version` will only be automatically updated from the base CSV
when you set `--version` when running `generate <bundle|packagemanifests>`.

[olm]:https://github.com/operator-framework/operator-lifecycle-manager
[doc-csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/0.15.1/doc/design/building-your-csv.md
[cli-overview]:/docs/olm-integration/cli-overview
[cli-gen-kustomize-manifests]:/docs/cli/operator-sdk_generate_kustomize_manifests
[cli-gen-bundle]:/docs/cli/operator-sdk_generate_bundle
[bundle]: https://github.com/operator-framework/operator-registry/blob/v1.16.1/docs/design/operator-bundle.md
[bundle-metadata]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md#bundle-annotations
[install-modes]:https://github.com/operator-framework/operator-lifecycle-manager/blob/4197455/Documentation/design/building-your-csv.md#operator-metadata
[olm-capabilities]:/docs/overview/operator-capabilities/
[csv-markers]:/docs/building-operators/golang/references/markers
[operatorhub]:https://operatorhub.io/
[scorecard]:/docs/testing-operators/scorecard/
[operatorhub_validator]:https://olm.operatorframework.io/docs/tasks/creating-operator-bundle/#validating-your-bundle
[relatedimages]:https://pkg.go.dev/github.com/operator-framework/api@v0.8.1/pkg/operators/v1alpha1#RelatedImage
