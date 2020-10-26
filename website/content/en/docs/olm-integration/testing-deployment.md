---
title: Testing Operator Deployment with OLM
linkTitle: Testing Deployment
weight: 30
---

This document discusses the behavior of `operator-sdk <run|cleanup>` subcommands related to OLM deployment,
and assumes you are familiar with [OLM][olm], related terminology,
and have read the SDK-OLM integration [design proposal][sdk-olm-design].

Currently only the package manifests format is supported by `<run|cleanup>` subcommands. Bundle support is coming soon.

**Note:** before continuing, please read the [caveats](#caveats) section below.

## `operator-sdk <run|cleanup> packagemanifests` command overview

`operator-sdk <run|cleanup> packagemanifests` assumes OLM is already installed and running on your cluster,
and that your Operator has a valid [package manifests format][package-manifests].
See the [CLI overview][doc-cli-overview] for commands to work with an OLM installation and generate a package manifests format.

Let's look at the anatomy of the `run packagemanifests` (which is the same for `cleanup`) configuration model:

- **kubeconfig-path**: the local path to a kubeconfig.
  - This uses well-defined default loading rules to load the config if empty.
- **namespace**: the cluster namespace in which Operator resources are created.
  - This namespace must already exist in the cluster.
- **manifests-dir**: a directory containing the Operator's package manifests.
- **version**: the version of the Operator to deploy. It must be a semantic version, ex. 0.0.1.
  - This version must match the version of the CSV manifest found in **manifests-dir**,
    ex. `packagemanifests/0.0.1` in an Operator SDK project.
- **install-mode**: specifies which supported [`installMode`][csv-install-modes] should be used to
  create an `OperatorGroup` by configuring its `spec.targetNamespaces` field.
  - The `InstallModeType` string passed must be marked as "supported" in the CSV being installed.
    The namespaces passed must exist or be created by passing a `Namespace` manifest to IncludePaths.
  - This option understands the following strings (assuming your CSV does as well):
      - `AllNamespaces`: the Operator will watch all namespaces (cluster-scoped Operators). This is the default.
      - `OwnNamespace`: the Operator will watch its own namespace (from **namespace** or the kubeconfig default).
      - `SingleNamespace="my-ns"`: the Operator will watch a namespace, not necessarily its own.
- **timeout**: a time string dictating the maximum time that `run` can run. The command will
  return an error if the timeout is exceeded.

### Caveats

- `<run|cleanup> packagemanifests` are intended to be used for testing purposes only,
since this command creates a transient image registry that should not be used in production.
Typically a registry is deployed separately and a set of catalog manifests are created in the cluster
to inform OLM of that registry and which Operator versions it can deploy and where to deploy the Operator.
- `run packagemanifests` can only deploy one Operator and one version of that Operator at a time,
hence its intended purpose being testing only.


[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
[sdk-olm-design]:https://github.com/operator-framework/operator-sdk/blob/master/proposals/sdk-integration-with-olm.md
[doc-cli-overview]:/docs/olm-integration/cli-overview
[package-manifests]:https://github.com/operator-framework/operator-registry/tree/v1.5.3#manifest-format
[csv-install-modes]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#operator-metadata
[cli-olm-install]:/docs/cli/operator-sdk_olm_install
[cli-olm-status]:/docs/cli/operator-sdk_olm_status
