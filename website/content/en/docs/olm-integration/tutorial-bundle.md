---
title: OLM Integration Bundle Tutorial
linkTitle: Bundle Tutorial
weight: 1
---

The [Operator Lifecycle Manager (OLM)][olm] is a set of cluster resources that manage the lifecycle of an Operator.
The Operator SDK supports both creating manifests for OLM deployment, and testing your Operator on an OLM-enabled
Kubernetes cluster.

This document succinctly walks through getting an Operator OLM-ready with [bundles][bundle], and glosses over
explanations of certain steps for brevity. The following documents contain more detail on these steps:
- All operator-framework manifest commands supported by the SDK: [CLI overview][doc-cli-overview].
- Generating operator-framework manifests: [generation overview][doc-olm-generate].

If you are working with package manifests, see the [package manifests tutorial][tutorial-package-manifests]
once you have completed the *Setup* section below.

**Important:** this guide assumes your project was scaffolded with `operator-sdk init --project-version=3`.
These features are unavailable to projects of version `2` or less; this information can be found by inspecting
your `PROJECT` file's `version` value.

## Setup

Let's first walk through creating an Operator for `memcached`, a distributed key-value store.

Follow one of the user guides to develop the memcached-operator in either [Go][sdk-user-guide-go],
[Ansible][sdk-user-guide-ansible], or [Helm][sdk-user-guide-helm], depending on which Operator type you are interested in.
This guide assumes memcached-operator is on version `0.0.1`, which is set in the `Makefile` variable `VERSION`.

### Enabling OLM

Ensure OLM is enabled on your cluster before following this guide. [`operator-sdk olm`][cli-olm]
has several subcommands that can install, uninstall, and check the status of particular OLM versions in a cluster.

**Note:** Certain cluster types may already have OLM enabled, but under a non-default (`"olm"`) namespace,
which can be configured by setting `--olm-namespace=[non-default-olm-namespace]` for `operator-sdk olm status|uninstall` subcommands.

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


**Note:** The `operator-sdk olm status` command is geared to detect the status of OLM that was installed by installation methods like `operator-sdk olm install` or by applying OLM [manifests][olm-manifests] directly on the cluster. This command retrieves the resources that were compiled into SDK at the time of installation from the OLM [manifests][olm-manifests]. However, if OLM was installed in a cluster in a custom fashion (such as in OpenShift clusters), it is possible that some resources will show a `Not Found` status when the `operator-sdk olm status` command is issued.

To check the true status of such resources in OCP clusters, run:

```
oc get <resource-name> -n <resource-namespace>
```

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

_If working with package manifests, see the [package manifests tutorial][tutorial-package-manifests]._

We will now create bundle manifests by running `make bundle` in the root of the memcached-operator project.

```console
$ make bundle
/home/user/go/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
operator-sdk generate kustomize manifests -q
kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version 0.0.1
INFO[0000] Building annotations.yaml
INFO[0000] Writing annotations.yaml in /home/user/go/src/github.com/test-org/memcached-operator/bundle/metadata
INFO[0000] Building Dockerfile
INFO[0000] Writing bundle.Dockerfile in /home/user/go/src/github.com/test-org/memcached-operator
operator-sdk bundle validate ./bundle
INFO[0000] Found annotations file                        bundle-dir=bundle container-tool=docker
INFO[0000] Could not find optional dependencies file     bundle-dir=bundle container-tool=docker
INFO[0000] All validation tests have completed successfully
```

The above command will have created the following bundle artifacts: a manifests directory
(`bundle/manifests`) containing a CSV and all CRDs from `config/crds`, [metadata][bundle-metadata]
directory (`bundle/metadata`), and [`bundle.Dockerfile`][bundle-dockerfile] have been created in
the Operator project. These files have been statically validated by `operator-sdk bundle validate`
to ensure the on-disk bundle representation is correct.

## Deploying an Operator with OLM

At this point in development we've generated all files necessary to build the memcached-operator bundle.
Now we're ready to test and deploy the Operator with OLM.

**Note:** If testing a bundle whose image will be hosted in a registry that is private and/or
has a custom CA, these [configuration steps][image-reg-config] must be complete.

### Testing bundles

Before proceeding, make sure you've [Installed OLM](#enabling-olm) onto your
cluster.

First, we need to build our bundle. To build a memcached-operator bundle, run:

```console
$ make bundle-build bundle-push BUNDLE_IMG=<some-registry>/memcached-operator-bundle:v0.0.1
```

Now that the bundle image is present in a registry, [`operator-sdk run bundle`][cli-run-bundle]
can create a pod to serve that bundle to OLM via a [`Subscription`][install-your-operator],
along with other OLM objects, ephemerally.

```console
$ operator-sdk run bundle <some-registry>/memcached-operator-bundle:v0.0.1
INFO[0008] Successfully created registry pod: <some-registry>-memcached-operator-bundle-0-0-1
INFO[0008] Created CatalogSource: memcached-operator-catalog
INFO[0008] OperatorGroup "operator-sdk-og" created
INFO[0008] Created Subscription: memcached-operator-v0-0-1-sub
INFO[0019] Approved InstallPlan install-krv7q for the Subscription: memcached-operator-v0-0-1-sub
INFO[0019] Waiting for ClusterServiceVersion "default/memcached-operator.v0.0.1" to reach 'Succeeded' phase
INFO[0019]   Waiting for ClusterServiceVersion "default/memcached-operator.v0.0.1" to appear
INFO[0031]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: Pending
INFO[0032]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: Installing
INFO[0040]   Found ClusterServiceVersion "default/memcached-operator.v0.0.1" phase: Succeeded
INFO[0040] OLM has successfully installed "memcached-operator.v0.0.1"
```

**Note:** If the bundle that is being installed has dependencies, the `--index-image` flag allows adding a bundle to a catalog that contains that bundle's dependencies.

**Note:** Version `v1.22.0` and later of the `operator-sdk` use the new file-based catalog (FBC) bundle format by default. Earlier releases use the deprecated SQLite bundle format. If you use an earlier version of the Operator SDK, you must update to a newer version or specify the index image by adding the `--index-image=quay.io/operator-framework/opm:v1.23.0` flag. For more information about this known issue, see the [FAQ][run-bundle-fbc-sqlite-faq].
<!-- TODO(jmccormick2001): add `scorecard` usage here -->

### Upgrading a bundle to a newer version

We can use the `operator-sdk run bundle-upgrade` command with a newer version of bundle image to upgrade
an existing operator bundle deployed on cluster. The command automates the manual orchestration typically required to upgrade an operator
from one version to another. It extracts the package name from bundle, finds the existing subscription, updates the catalog
source, deletes the existing registry pod and creates a new registry pod with the version of bundle image provided in the command.

Let's upgrade the previously deployed memcached-operator bundle from version `0.0.1` to `0.0.2`.

```console
$ operator-sdk run bundle-upgrade <some-registry>/memcached-operator-bundle:v0.0.2
INFO[0002] Found existing subscription with name memcached-operator-bundle-0-0-1-sub and namespace default
INFO[0002] Found existing catalog source with name memcached-operator-catalog and namespace default
INFO[0007] Successfully created registry pod: <some-registry>-memcached-operator-bundle-0-0-2
INFO[0007] Updated catalog source memcached-operator-catalog with address and annotations
INFO[0008] Deleted previous registry pod with name "<some-registry>-memcached-operator-bundle-0-0-1"
INFO[0050] Approved InstallPlan install-c8fkh for the Subscription: memcached-operator-bundle-0-0-1-sub
INFO[0050] Waiting for ClusterServiceVersion "default/memcached-operator.v0.0.2" to reach 'Succeeded' phase
INFO[0050]   Waiting for ClusterServiceVersion "default/memcached-operator.v0.0.2" to appear
INFO[0052]   Found ClusterServiceVersion "default/memcached-operator.v0.0.2" phase: Pending
INFO[0057]   Found ClusterServiceVersion "default/memcached-operator.v0.0.2" phase: InstallReady
INFO[0058]   Found ClusterServiceVersion "default/memcached-operator.v0.0.2" phase: Installing
INFO[0095]   Found ClusterServiceVersion "default/memcached-operator.v0.0.2" phase: Succeeded
INFO[0095] Successfully upgraded to "memcached-operator.v0.0.2"
```

**Note:** If a bundle was installed using [`operator-sdk run bundle`][run-bundle] with a SQLite index image, the `replaces` field *must* be present and populated in the upgraded CSV's spec. 

#### Upgrading a bundle that was installed traditionally using OLM

An operator bundle can be upgraded even if it was originally deployed using OLM without using the `run bundle` command.

Let's see how to deploy an operator bundle traditionally using OLM and then upgrade the operator bundle to a newer version.

First, create a CatalogSource by building the CatalogSource from a catalog.

```console
$ oc create -f catalogsource.yaml
```

```yaml
# catalogsource.yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: etcdoperator
  namespace: default
spec:
  displayName: Etcd Operators
  image: <some-registry>/etcd-catalog:latest
  sourceType: grpc
```

Next, install the operator bundle by creating a subscription.

```console
$ oc create -f subscription.yaml
```

```yaml
# subscription.yaml
apiVersion: v1
items:
- apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: etcd
    namespace: default
  spec:
    channel: "stable"
    installPlanApproval: Manual
    name: etcd
    source: etcdoperator
    sourceNamespace: default
    startingCSV: etcdoperator.v0.0.1
```

Once the Operator bundle is deployed, you can use the `run bundle-upgrade` command by specifying the new bundle image that you want to upgrade to.

```console
$ operator-sdk run bundle-upgrade <some-registry>/etcd-bundle:v0.0.2
INFO[0000] Found existing subscription with name etcd and namespace default
INFO[0000] Found existing catalog source with name etcdoperator and namespace default
INFO[0005] Successfully created registry pod: <some-registry>-etcd-bundle-0-0-2
INFO[0005] Updated catalog source etcdoperator with address and annotations
INFO[0005] Deleted previous registry pod with name "<some-registry>-etcd-bundle-0-0-1"
INFO[0005] Approved InstallPlan install-6vrzh for the Subscription: etcd
INFO[0005] Waiting for ClusterServiceVersion "default/etcdoperator.v0.0.2" to reach 'Succeeded' phase
INFO[0005]   Waiting for ClusterServiceVersion "default/etcdoperator.v0.0.2" to appear
INFO[0007]   Found ClusterServiceVersion "default/etcdoperator.v0.0.2" phase: Pending
INFO[0008]   Found ClusterServiceVersion "default/etcdoperator.v0.0.2" phase: Installing
INFO[0018]   Found ClusterServiceVersion "default/etcdoperator.v0.0.2" phase: Succeeded
INFO[0018] Successfully upgraded to "etcdoperator.v0.0.2"
```

### Deploying bundles in production

OLM and Operator Registry consumes Operator bundles via a catalog of Operators, implemented as an
[index image][index-image], which are composed of one or more bundles. To build and push a
memcached-operator bundle image for version v0.0.1, run:

```console
$ make bundle-build bundle-push BUNDLE_IMG=<some-registry>/memcached-operator-bundle:v0.0.1
```

Now you can build and push the catalog by running `catalog-*` Makfile targets, which use
the Operator package manager tool [`opm`][opm] to [build][doc-index-build] the catalog:

```console
$ make catalog-build catalog-push CATALOG_IMG=<some-registry>/memcached-operator-catalog:v0.0.1
```

Assuming `IMAGE_TAG_BASE = <some-registry>/memcached-operator` has the desired tag base, you can inline
the above two commands to:

```console
$ make bundle-build bundle-push catalog-build catalog-push
```

Which will build and push both `<some-registry>/memcached-operator-bundle:v0.0.1`
and `<some-registry>/memcached-operator-catalog:v0.0.1`.

## Further reading

In-depth discussions of OLM concepts mentioned here:
- [CatalogSource][catalogsource]
- [Subscription][subscription]
- [Install an Operator from a catalog][olm-install]


[sdk-user-guide-go]:/docs/building-operators/golang/quickstart
[sdk-user-guide-ansible]:/docs/building-operators/ansible/quickstart
[sdk-user-guide-helm]:/docs/building-operators/helm/quickstart
[tutorial-package-manifests]:/docs/olm-integration/tutorial-package-manifests
[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
[bundle]:https://github.com/operator-framework/operator-registry/blob/v1.16.1/docs/design/operator-bundle.md
[bundle-metadata]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md#bundle-annotations
[bundle-dockerfile]:https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md#bundle-dockerfile
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
[cli-olm]:/docs/cli/operator-sdk_olm
[cli-run-bundle]:/docs/cli/operator-sdk_run_bundle
[doc-cli-overview]:/docs/olm-integration/cli-overview
[doc-olm-generate]:/docs/olm-integration/generation
[opm]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md
[index-image]:https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md#index
[doc-index-build]:https://github.com/operator-framework/operator-registry#building-an-index-of-operators-using-opm
[install-your-operator]:https://olm.operatorframework.io/docs/tasks/install-operator-with-olm/#install-your-operator
[catalogsource]:https://olm.operatorframework.io/docs/concepts/crds/catalogsource/
[subscription]:https://olm.operatorframework.io/docs/concepts/crds/subscription/
[olm-install]:https://olm.operatorframework.io/docs/tasks/install-operator-with-olm/
[olm-manifests]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/deploy/upstream/quickstart/olm.yaml
[run-bundle]: https://sdk.operatorframework.io/docs/cli/operator-sdk_run_bundle/
[run-bundle-fbc-sqlite-faq]: https://sdk.operatorframework.io/docs/faqs/#operator-sdk-run-bundle-command-fails-and-the-registry-pod-has-an-error-of-mkdir-cant-create-directory-database-permission-denied
