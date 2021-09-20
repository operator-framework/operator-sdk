---
title: Testing Operator Deployment with OLM
linkTitle: Testing Deployment
weight: 30
---

This document discusses the behavior of `operator-sdk <run|cleanup>` subcommands related to OLM deployment,
and assumes you are familiar with [OLM][olm], related terminology,
and have read the SDK-OLM integration [design proposal][sdk-olm-design].

### Caveats

- `run bundle`, `run bundle-upgrade`, `run packagemanifests`, and `cleanup` are intended to be used for testing purposes only,
since these commands create a transient image registry that should not be used in production.
Typically a registry is deployed separately and a set of catalog manifests are created in the cluster
to inform OLM of that registry and which Operator versions it can deploy and where to deploy the Operator.
- `run bundle` and `run packagemanifests` can only deploy one Operator and one version of that Operator at a time, 
and `run bundle-upgrade` can only upgrade one Operator and one version of that Operator at a time, 
hence their intended purpose being testing only.
- If testing a bundle or catalog whose image will be hosted in a registry that is private and/or
has a custom CA, these [configuration steps][image-reg-config] must be complete.


## `operator-sdk run bundle` command overview

`operator-sdk run bundle` assumes OLM is already installed and running on your
cluster. It also assumes that your Operator has a valid [bundle][bundle-format].
See the [creating a bundle][creating-bundle] guide for more information. See the
[CLI overview][doc-cli-overview] for commands to work with an OLM installation
and generate a bundle.

```
operator-sdk run bundle <bundle-image> [--index-image=] [--kubeconfig=] [--namespace=] [--timeout=] [--install-mode=(AllNamespace|OwnNamespace|SingleNamespace=)]
```

Let's look at the configuration shared between `run bundle`, `run
packagemanifests`, `run bundle-upgrade` and `cleanup`:

- **kubeconfig**: the local path to a kubeconfig. This uses well-defined default
  loading rules to load the config if empty.
- **namespace**: the cluster namespace in which Operator resources are created.
  This namespace must already exist in the cluster. This is an optional field
  which will default to the kubeconfig context if not provided.
- **timeout**: a time string dictating the maximum time that `run` can run. The
  command will return an error if the timeout is exceeded.

Let's look at the anatomy of the `run bundle` configuration model:

- **bundle-image**: specifies the Operator bundle image, this is a
  required parameter. The bundle image must be pullable.
- **index-image**: specifies an index image in which to inject the given bundle.
  This is an optional field which will default to
  `quay.io/operator-framework/opm:latest`
- **install-mode**: specifies which supported [`installMode`][csv-install-modes]
  should be used to create an `OperatorGroup` by configuring its
  `spec.targetNamespaces` field. The `InstallModeType` string passed must be
  marked as "supported" in the CSV being installed.
  - This option understands the following strings (assuming your CSV does as
    well):
    - `AllNamespaces`: the Operator will watch all namespaces (cluster-scoped
      Operators). This is the default.
    - `OwnNamespace`: the Operator will watch its own namespace (from
      **namespace** or the kubeconfig default).
    - `SingleNamespace="my-ns"`: the Operator will watch a namespace, not
      necessarily its own.
  - This is an optional parameter, but if the CSV does not support
    `AllNamespaces` then this parameter becomes **required** to instruct
    `run bundle` with the appropriate `InstallModeType`.

## `operator-sdk run packagemanifests` command overview

`operator-sdk run packagemanifests` assumes OLM is already installed and
running on your cluster, and that your Operator has a valid
[package manifests format][package-manifests]. See the
[CLI overview][doc-cli-overview] for commands to work with an OLM installation
and generate a package manifests format.

```
operator-sdk run packagemanifests <packagemanifests-root-dir> [--version=] [--kubeconfig=] [--namespace=] [--timeout=] [--install-mode=(AllNamespace|OwnNamespace|SingleNamespace=)]
```

Let's look at the configuration shared between `run bundle`, `run
packagemanifests` and `cleanup`:

- **kubeconfig**: the local path to a kubeconfig. This uses well-defined default
  loading rules to load the config if empty.
- **namespace**: the cluster namespace in which Operator resources are created.
  This namespace must already exist in the cluster. This is an optional field
  which will default to the kubeconfig context if not provided.
- **timeout**: a time string dictating the maximum time that `run` can run. The
  command will return an error if the timeout is exceeded.

Let's look at the anatomy of the `run packagemanifests` configuration model:

- **packagemanifests-root-dir**: a directory containing the Operator's package
  manifests, this is a required parameter.
- **install-mode**: specifies which supported [`installMode`][csv-install-modes]
  should be used to create an `OperatorGroup` by configuring its
  `spec.targetNamespaces` field. The `InstallModeType` string passed must be
  marked as "supported" in the CSV being installed.
  - This option understands the following strings (assuming your CSV does as
    well):
    - `AllNamespaces`: the Operator will watch all namespaces (cluster-scoped
      Operators). This is the default.
    - `OwnNamespace`: the Operator will watch its own namespace (from
      **namespace** or the kubeconfig default).
    - `SingleNamespace="my-ns"`: the Operator will watch a namespace, not
      necessarily its own.
  - This is an optional parameter, but if the CSV does not support
    `AllNamespaces` then this parameter becomes **required** to instruct
    `run packagemanifests` with the appropriate `InstallModeType`.
- **version**: the version of the Operator to deploy. It must be a semantic
  version, ex. 0.0.1. This version must match the version of the CSV manifest
  found in **manifests-dir**, e.g. `packagemanifests/0.0.1` in an Operator
  SDK project.

## `operator-sdk run bundle-upgrade` command overview

`operator-sdk run bundle-upgrade` assumes OLM is already installed and running on your 
cluster and that the Operator has a valid [bundle][bundle-format]. It also assumes that 
the previous version of the Operator was either deployed on the cluster using `run bundle` 
command or traditionally via OLM. Another assumption of this command is that the newer operator bundle
should not exist in the index image, if the previous version of the operator bundle was installed 
traditionally using OLM. This will cause the registry pod to fail as the bundle is already added to the 
index that provides package and csv. See the [CLI overview][doc-cli-overview] for commands to work with 
an OLM installation and generate a bundle.

```
operator-sdk run bundle-upgrade <bundle-image> [--kubeconfig=] [--namespace=] [--timeout=] 
```
Let's look at the anatomy of the `run bundle-upgrade` configuration model:

- **bundle-image**: specifies the Operator bundle image, this is a
  required parameter. The bundle image must be pullable.

## `operator-sdk cleanup` command overview

`operator-sdk cleanup` assumes an Operator was deployed using `run bundle` or
`run packagemanifests`.

```
operator-sdk cleanup <operatorPackageName> [--delete-all=] [--delete-crds=] [--delete-operator-groups=] [--kubeconfig=] [--namespace=] [--timeout=]
```

Let's look at the configuration shared between `run bundle`, `run
packagemanifests` and `cleanup`:

- **kubeconfig**: the local path to a kubeconfig. This uses well-defined default
  loading rules to load the config if empty.
- **namespace**: the cluster namespace in which Operator resources are created.
  This namespace must already exist in the cluster. This is an optional field
  which will default to the kubeconfig context if not provided.
- **timeout**: a time string dictating the maximum time that `run` can run. The
  command will return an error if the timeout is exceeded.

Let's look at the anatomy of the `cleanup` configuration model:

- **operatorPackageName**: the Operator's package name which you want to remove
  from the cluster, e.g. memcached-operator. This is a required parameter.
- **delete-all**: a boolean indicating to enable all the delete flags that are present. This is an optional
  field which will default to true if not provided. If set to true, it will enable all the delete flags to be true. If set to false, it will enable specific delete flags.
- **delete-crds**: a boolean indicating to delete all owned CRDs and CRs. This is an optional field
  which will default to false if not provided. If set to true, owned CRDs and CRs
  will be deleted.
- **delete-operator-groups**: a boolean indicating to delete all operator groups. This is an optional field
  which will default to false if not provided. If set to true, operator groups will be deleted.


[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
[sdk-olm-design]:https://github.com/operator-framework/operator-sdk/blob/master/proposals/sdk-integration-with-olm.md
[doc-cli-overview]:/docs/olm-integration/cli-overview
[bundle-format]:https://github.com/operator-framework/operator-registry/tree/v1.15.3#manifest-format
[package-manifests]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[csv-install-modes]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#operator-metadata
[cli-olm-install]:/docs/cli/operator-sdk_olm_install
[cli-olm-status]:/docs/cli/operator-sdk_olm_status
[creating-bundles]:/docs/olm-integration/tutorial-bundle/#creating-a-bundle
[add-sa-secret]:https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account
[image-reg-config]:/docs/olm-integration/cli-overview#private-bundle-and-catalog-image-registries
