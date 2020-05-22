---
title: OLM Integration User Guide
linkTitle: OLM Integration User Guide
weight: 1
---

The [Operator Lifecycle Manager (OLM)][olm] is a set of cluster resources that
manage the lifecycle of an Operator. The Operator SDK supports both readying your
Operator project for OLM deployment, and testing your Operator on an OLM-enabled
Kubernetes cluster.

## Setup

Lets first walk through creating an Operator for `memcached`, a distributed key-value store.

Follow one of the user guides to develop the memcached-operator in either [Go][sdk-user-guide-go],
[Ansible][sdk-user-guide-ansible], or [Helm][sdk-user-guide-helm], depending on which Operator type you are interested in.

### Enabling OLM

Ensure OLM is enabled on your cluster before following this guide. [`operator-sdk olm`][cli-olm]
has several subcommands that can install, uninstall, and check the status of
particular OLM versions in a cluster.

**Note:** Certain cluster types may already have OLM enabled, but under a
non-default (`"olm"`) namespace, which can be configured by setting
`--olm-namespace=[non-default-olm-namespace]` for `operator-sdk olm` subcommands
and `operator-sdk run packagemanifests`.

You can check if OLM is already installed by running the following command,
which will detect the installed OLM version automatically (v0.14.1 in this example):

```console
$ operator-sdk olm status
INFO[0000] Fetching CRDs for version "0.14.1"
INFO[0002] Fetching resources for version "0.14.1"
INFO[0002] Successfully got OLM status for version "0.14.1"

NAME                                            NAMESPACE    KIND                        STATUS
olm                                                          Namespace                   Installed
operatorgroups.operators.coreos.com                          CustomResourceDefinition    Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
...
```

All resources listed should have status `Installed`.

If OLM is not already installed, go ahead and install the latest version:

```console
$ operator-sdk olm install
INFO[0000] Fetching CRDs for version "latest"
INFO[0001] Fetching resources for version "latest"
INFO[0007] Creating CRDs and resources
INFO[0007]   Creating CustomResourceDefinition "clusterserviceversions.operators.coreos.com"
INFO[0007]   Creating CustomResourceDefinition "installplans.operators.coreos.com"
INFO[0007]   Creating CustomResourceDefinition "subscriptions.operators.coreos.com"
...
NAME                                            NAMESPACE    KIND                        STATUS
clusterserviceversions.operators.coreos.com                  CustomResourceDefinition    Installed
installplans.operators.coreos.com                            CustomResourceDefinition    Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
...
```

## Creating a bundle

Now that we have a working, simple memcached-operator, we can generate manifests
and metadata for an Operator [bundle][operator-bundle]. From the bundle docs:

> An Operator Bundle is built as a scratch (non-runnable) container image that
> contains operator manifests and specific metadata in designated directories
> inside the image. Then, it can be pushed and pulled from an OCI-compliant
> container registry. Ultimately, an operator bundle will be used by Operator
> Registry and OLM to install an operator in OLM-enabled clusters.

A bundle consists of on-disk manifests and metadata that define an Operator
at a particular version. At this stage in memcached-operator's development,
we only need to worry about generating bundle files; bundle images becomes important
once we're ready to [publish](#publishing-an-operator) our Operator. For a condensed
overview of all bundle operations supported by the SDK, read [this doc][doc-bundle-cli].

We will now create bundle manifests by running [`operator-sdk generate csv`][cli-generate-csv]
in the root of the memcached-operator project:

**Note:** while `generate csv` only officially supports Go Operators, it will
generate a barebones CSV for Ansible and Helm Operators that _will_ require manual modification.

```console
$ operator-sdk generate csv --csv-version 0.1.0
INFO[0000] Generating CSV manifest version 0.1.0
INFO[0004] Required csv fields not filled in file deploy/olm-catalog/memcached-operator/manifests/memcached-operator.clusterserviceversion.yaml:
	spec.keywords
	spec.provider
INFO[0004] Created deploy/olm-catalog/memcached-operator/manifests/memcached-operator.clusterserviceversion.yaml
```

A bundle manifests directory containing a CSV and all CRDs in `deploy/crds` has
been created at `deploy/olm-catalog/memcached-operator/manifests`:

```console
$ tree deploy/olm-catalog/memcached-operator
deploy/olm-catalog/memcached-operator
└── manifests
    ├── cache.example.com_memcacheds_crd.yaml
    └── memcached-operator.clusterserviceversion.yaml
```

Next we create bundle [metadata][bundle-metadata] and a [Dockerfile][bundle-dockerfile].
Metadata contains information about a particular Operator version available in a registry.
OLM uses this information to install specific Operator versions and resolve dependencies.

Of particular note are channels:

> Channels allow package authors to write different upgrade paths for different users (e.g. beta vs. stable).

Channels become important when publishing, but we should still be aware of them
beforehand as they're required values in our metadata. `bundle create` writes the
channel `stable` by default.

Invoking [`operator-sdk bundle create`][cli-bundle-create] creates annotations metadata
and a `bundle.Dockerfile` for the memcached-operator:

```console
$ operator-sdk bundle create --generate-only
INFO[0000] Building annotations.yaml                    
INFO[0000] Generating output manifests directory        
INFO[0000] Building Dockerfile
```

Your `deploy/olm-catalog/memcached-operator/metadata/annotations.yaml` and `bundle.Dockerfile`
contain the same [annotations][bundle-metadata] in slightly different formats.
In most cases annotations do not need to be modified; if you do decide to modify
them, both sets of annotations _must_ be the same to ensure consistent Operator deployment.

Now that everything need to deploy memcached-operator with OLM has been generated,
we want to ensure our bundled metadata and manifests are [valid][cli-bundle-validate]:

```console
$ operator-sdk bundle validate deploy/olm-catalog/memcached-operator/
INFO[0000] All validation tests have completed successfully  bundle-dir=/home/user/projects/memcached-operator/deploy/olm-catalog/memcached-operator
```

### Publishing an Operator

If you eventually wish to [publish][operatorhub] your Operator, you'll want to
add UI metadata to your CSV. A thorough explanation of which metadata fields
are available and what they render as can be found [here][csv]. Go API [markers][csv-markers]
direct `generate csv` to automatically and reproducibly populate many of these fields.

## Deploying an Operator with OLM

At this point in development we've generated all files necessary to build the
memcached-operator bundle. Now we're ready to test and deploy the Operator
with OLM.

### Testing bundles

Coming soon.

### Testing package manifests

[`operator-sdk run packagemanifests`][cli-run-packagemanifests] will create an Operator [registry][operator-registry]
from manifests and metadata in the memcached-operator project, and inform OLM that
memcached-operator v0.1.0 is ready to be deployed. This process effectively
replicates production deployment in a constrained manner to make sure OLM can
deploy our Operator successfully before attempting real production deployment.

`run packagemanifests` performs some optionally configurable setup [under the hood][doc-run-olm], but for
most use cases the following invocation is all we need:

```console
$ operator-sdk run packagemanifests --operator-version 0.1.0
INFO[0000] loading Bundles                               dir=deploy/olm-catalog/memcached-operator
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=memcached-operator load=bundles
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=manifests load=bundles
INFO[0000] found csv, loading bundle                     dir=deploy/olm-catalog/memcached-operator file=memcached-operator.clusterserviceversion.yaml load=bundles
INFO[0000] loading bundle file                           dir=deploy/olm-catalog/memcached-operator/manifests file=example.com_memcacheds_crd.yaml load=bundle
INFO[0000] loading bundle file                           dir=deploy/olm-catalog/memcached-operator/manifests file=memcached-operator.clusterserviceversion.yaml load=bundle
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=metadata load=bundles
INFO[0000] loading Packages and Entries                  dir=deploy/olm-catalog/memcached-operator
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=memcached-operator load=package
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=manifests load=package
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=metadata load=package
INFO[0000] Creating registry
INFO[0000]   Creating ConfigMap "olm/memcached-operator-registry-bundles"
INFO[0000]   Creating Deployment "olm/memcached-operator-registry-server"
INFO[0000]   Creating Service "olm/memcached-operator-registry-server"
INFO[0000] Waiting for Deployment "olm/memcached-operator-registry-server" rollout to complete
INFO[0000]   Waiting for Deployment "olm/memcached-operator-registry-server" to rollout: 0 out of 1 new replicas have been updated
INFO[0001]   Waiting for Deployment "olm/memcached-operator-registry-server" to rollout: 0 of 1 updated replicas are available
INFO[0002]   Deployment "olm/memcached-operator-registry-server" successfully rolled out
INFO[0002] Creating resources
INFO[0002]   Creating CatalogSource "default/memcached-operator-ocs"
INFO[0002]   Creating Subscription "default/memcached-operator-v0-0-1-sub"
INFO[0002]   Creating OperatorGroup "default/operator-sdk-og"
INFO[0002] Waiting for ClusterServiceVersion "default/memcached-operator.v0.1.0" to reach 'Succeeded' phase
INFO[0002]   Waiting for ClusterServiceVersion "default/memcached-operator.v0.1.0" to appear
INFO[0034]   Found ClusterServiceVersion "default/memcached-operator.v0.1.0" phase: Pending
INFO[0035]   Found ClusterServiceVersion "default/memcached-operator.v0.1.0" phase: InstallReady
INFO[0036]   Found ClusterServiceVersion "default/memcached-operator.v0.1.0" phase: Installing
INFO[0036]   Found ClusterServiceVersion "default/memcached-operator.v0.1.0" phase: Succeeded
INFO[0037] Successfully installed "memcached-operator.v0.1.0" on OLM version "0.14.1"

NAME                            NAMESPACE    KIND                        STATUS
memcacheds.cache.example.com    default      CustomResourceDefinition    Installed
memcached-operator.v0.1.0       default      ClusterServiceVersion       Installed
```

As long as both the `ClusterServiceVersion` and all `CustomResourceDefinition`'s
return an `Installed` status, the memcached-operator has been deployed successfully.

Now that we're done testing the memcached-operator, we should probably clean up
the Operator's resources. The [`operator-sdk cleanup packagemanifests`][cli-cleanup-packagemanifests]
command will do this for you:

```console
$ operator-sdk cleanup packagemanifests --operator-version 0.1.0
INFO[0000] loading Bundles                               dir=deploy/olm-catalog/memcached-operator
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=memcached-operator load=bundles
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=manifests load=bundles
INFO[0000] found csv, loading bundle                     dir=deploy/olm-catalog/memcached-operator file=memcached-operator.clusterserviceversion.yaml load=bundles
INFO[0000] loading bundle file                           dir=deploy/olm-catalog/memcached-operator/manifests file=example.com_memcacheds_crd.yaml load=bundle
INFO[0000] loading bundle file                           dir=deploy/olm-catalog/memcached-operator/manifests file=memcached-operator.clusterserviceversion.yaml load=bundle
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=metadata load=bundles
INFO[0000] loading Packages and Entries                  dir=deploy/olm-catalog/memcached-operator
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=memcached-operator load=package
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=manifests load=package
INFO[0000] directory                                     dir=deploy/olm-catalog/memcached-operator file=metadata load=package
INFO[0000] Deleting resources
INFO[0000]   Deleting CatalogSource "default/memcached-operator-ocs"
INFO[0000]   Deleting Subscription "default/memcached-operator-v0-0-1-sub"
INFO[0000]   Deleting OperatorGroup "default/operator-sdk-og"
INFO[0000]   Deleting CustomResourceDefinition "default/memcacheds.example.com"
INFO[0000]   Deleting ClusterServiceVersion "default/memcached-operator.v0.1.0"
INFO[0000]   Waiting for deleted resources to disappear
INFO[0001] Successfully uninstalled "memcached-operator.v0.1.0" on OLM version "0.14.1"
```

### Production bundle deployment

OLM and Operator Registry consumes Operator bundles via an [index image][index-image],
which are composed of one or more bundles. To build a memcached-operator bundle, run:

```console
$ docker build -f bundle.Dockerfile -t quay.io/<username>/memcached-operator:v0.1.0 .
```

Although we've validated on-disk manifests and metadata, we also must make sure
the bundle itself is valid:

```console
$ docker push quay.io/<username>/memcached-operator:v0.1.0
$ operator-sdk bundle validate quay.io/<username>/memcached-operator:v0.1.0
INFO[0000] Unpacked image layers                         bundle-dir=/tmp/bundle-716785960 container-tool=docker
INFO[0000] running docker pull                           bundle-dir=/tmp/bundle-716785960 container-tool=docker
INFO[0002] running docker save                           bundle-dir=/tmp/bundle-716785960 container-tool=docker
INFO[0002] All validation tests have completed successfully  bundle-dir=/tmp/bundle-716785960 container-tool=docker
```

Currently the SDK does not build index images; instead, use the Operator package
manager tool [`opm`][opm] to manage index images. Once one has been built, follow
the OLM [docs][doc-olm-index] on adding an index to a cluster registry.


[sdk-user-guide-go]:/docs/golang/quickstart
[sdk-user-guide-ansible]:/docs/ansible/quickstart
[sdk-user-guide-helm]:/docs/helm/quickstart
[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
[csv]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md
[operator-registry]:https://github.com/operator-framework/operator-registry
[operator-bundle]:https://github.com/operator-framework/operator-registry/tree/master#manifest-format
[bundle-metadata]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md#bundle-annotations
[bundle-dockerfile]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md#bundle-dockerfile
[opm]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md
[olm-prod-install]:https://github.com/operator-framework/operator-lifecycle-manager/#discovery-catalogs-and-automated-upgrades
[operatorhub]:https://operatorhub.io/
[cli-olm]:/docs/cli/operator-sdk_olm
[cli-olm-install]:/docs/cli/operator-sdk_olm_install
[cli-olm-status]:/docs/cli/operator-sdk_olm_status
[cli-run-packagemanifests]:/docs/cli/operator-sdk_run_packagemanifests
[cli-cleanup-packagemanifests]:/docs/cli/operator-sdk_cleanup_packagemanifests
[cli-bundle-create]:/docs/cli/operator-sdk_bundle_create
[cli-bundle-validate]:/docs/cli/operator-sdk_bundle_validate
[doc-bundle-cli]:/docs/olm-integration/cli-overview
[cli-generate-csv]:/docs/cli/operator-sdk_generate_csv
[csv-markers]:/docs/golang/references/markers
[doc-run-olm]:/docs/olm-integration/olm-deployment/#operator-sdk-run-packagemanifests-command-overview
[doc-olm-index]:https://github.com/operator-framework/operator-registry#using-the-index-with-operator-lifecycle-manager
[index-image]:https://github.com/operator-framework/operator-registry#building-an-index-of-operators-using-opm
