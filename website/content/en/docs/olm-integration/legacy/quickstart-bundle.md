---
title: OLM Integration Bundle Quickstart
linkTitle: Bundle Quickstart
weight: 1
---

The [Operator Lifecycle Manager (OLM)][olm] is a set of cluster resources that manage the lifecycle of an Operator.
The Operator SDK supports both creating manifests for OLM deployment, and testing your Operator on an OLM-enabled
Kubernetes cluster.

This document succinctly walks through getting an Operator OLM-ready with [bundles][bundle], and glosses over
explanations of certains steps for brevity. The following documents contain more detail on these steps:
- All operator-framework manifest commands supported by the SDK: [CLI overview][doc-cli-overview].
- Generating operator-framework manifests: [generation overview][doc-olm-generate].

If you are working with package manifests, see the [package manifests quickstart][quickstart-package-manifests]
once you have completed the *Setup* section below.

## Setup

Let's first walk through creating an Operator for `memcached`, a distributed key-value store.

Follow one of the user guides to develop the memcached-operator in either [Go][sdk-user-guide-go],
[Ansible][sdk-user-guide-ansible], or [Helm][sdk-user-guide-helm], depending on which Operator type you are interested in.
This guide assumes memcached-operator is on version `0.0.1`.

### Enabling OLM

Ensure OLM is enabled on your cluster before following this guide. [`operator-sdk olm`][cli-olm]
has several subcommands that can install, uninstall, and check the status of particular OLM versions in a cluster.

**Note:** Certain cluster types may already have OLM enabled, but under a non-default (`"olm"`) namespace,
which can be configured by setting `--olm-namespace=[non-default-olm-namespace]` for `operator-sdk olm` subcommands
and `operator-sdk run packagemanifests`.

You can check if OLM is already installed by running the following command,
which will detect the installed OLM version automatically (0.15.1 in this example):

```console
$ operator-sdk olm status
INFO[0000] Fetching CRDs for version "0.15.1"
INFO[0002] Fetching resources for version "0.15.1"
INFO[0002] Successfully got OLM status for version "0.15.1"

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

**Note:** By default, `olm status` and `olm uninstall` auto-detect the OLM version installed in your cluster.
This can fail if the installation is broken in some way, so the version of OLM can be overridden using the
`--version` flag provided with these commands.

## Creating a bundle

_If working with package manifests, see the [package manifests quickstart][quickstart-package-manifests]._

We will now create bundle manifests and metadata by running `generate bundle` in the root of the memcached-operator project.

```sh
$ operator-sdk generate bundle --version 0.0.1
```

A bundle manifests directory `deploy/olm-catalog/memcached-operator/manifests` containing a CSV and all CRDs
in `deploy/crds`, a bundle [metadata][bundle-metadata] directory `deploy/olm-catalog/memcached-operator/metadata`,
and a [Dockerfile][bundle-dockerfile] `bundle.Dockerfile` have been created in the Operator project.

These files can be statically validated by `bundle validate` to ensure the on-disk bundle representation is correct:

```console
$ operator-sdk bundle validate ./deploy/olm-catalog/memcached-operator
INFO[0000] Found annotations file                        bundle-dir=deploy/olm-catalog/memcached-operator container-tool=docker
INFO[0000] Could not find optional dependencies file     bundle-dir=deploy/olm-catalog/memcached-operator container-tool=docker
INFO[0000] All validation tests have completed successfully
```

## Deploying an Operator with OLM

At this point in development we've generated all files necessary to build the memcached-operator bundle.
Now we're ready to test and deploy the Operator with OLM.

### Deploying bundles in production

OLM and Operator Registry consumes Operator bundles via an [index image][index-image],
which are composed of one or more bundles. To build a memcached-operator bundle, run:

```console
$ docker build -f bundle.Dockerfile -t quay.io/<username>/memcached-operator:v0.1.0 .
```

Although we've validated on-disk manifests and metadata, we also must make sure the bundle itself is valid:

```console
$ docker push quay.io/<username>/memcached-operator:v0.1.0
$ operator-sdk bundle validate quay.io/<username>/memcached-operator:v0.1.0
INFO[0000] Unpacked image layers                         bundle-dir=/tmp/bundle-716785960 container-tool=docker
INFO[0000] running docker pull                           bundle-dir=/tmp/bundle-716785960 container-tool=docker
INFO[0002] running docker save                           bundle-dir=/tmp/bundle-716785960 container-tool=docker
INFO[0002] All validation tests have completed successfully  bundle-dir=/tmp/bundle-716785960 container-tool=docker
```

The SDK does not build index images; instead, use the Operator package manager tool [`opm`][opm] to
[build][doc-index-build] one. Once one has been built, follow the index image [usage docs][doc-olm-index]
to add an index to a cluster catalog, and the catalog [discovery docs][doc-olm-discovery] to tell OLM
about your cataloged Operator.


[sdk-user-guide-go]:/docs/golang/legacy/quickstart
[sdk-user-guide-ansible]:/docs/ansible/quickstart
[sdk-user-guide-helm]:/docs/helm/quickstart
[quickstart-package-manifests]:/docs/olm-integration/legacy/quickstart-package-manifests
[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
[bundle]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md
[bundle-metadata]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md#bundle-annotations
[bundle-dockerfile]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md#bundle-dockerfile
[cli-olm]:/docs/cli/operator-sdk_olm
[doc-cli-overview]:/docs/olm-integration/legacy/cli-overview
[doc-olm-generate]:/docs/olm-integration/legacy/generating-a-csv
[opm]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md
[index-image]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md#index
[doc-index-build]:https://github.com/operator-framework/operator-registry#building-an-index-of-operators-using-opm
[doc-olm-index]:https://github.com/operator-framework/operator-registry#using-the-index-with-operator-lifecycle-manager
[doc-olm-discovery]:https://github.com/operator-framework/operator-lifecycle-manager/#discovery-catalogs-and-automated-upgrades
