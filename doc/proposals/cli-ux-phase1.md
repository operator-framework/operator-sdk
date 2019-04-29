# User Experience CLI Improvements | Phase 1

Implementation Owner: @joelanford

Status: Draft

[Background](#Background)

[Goal](#Goal)

[Use cases](#Use_cases)

[Proposed CLI commands](#Proposed_CLI_commands)

[References](#References)

## Background

The SDK CLI is one of the primary tools in the Operator Framework for operator developers and
deployers. However, the current user experience (UX) leaves a lot to be desired. Currently, the
tools and documentation that help a user create, develop, test, package, run, and publish an
operator are spread among many different repositories, making for a steep learning curve for 
newcomers.

## Goal

Describe an initial set of CLI additions to support a simplified workflow for an Operator SDK user to run an Operator SDK-developed project in a cluster using OLM.


## Use cases

1. As an operator developer, I want to deploy OLM to a development cluster.
2. As an operator developer, I want to create the resources necessary for running my operator via OLM.
3. As an operator developer, I want to run my operator via OLM.

## Proposed CLI commands

### `operator-sdk alpha`

Many other Kubernetes CLI projects (e.g. `kubebuilder`, `kubectl`, and `kubeadm`) have an `alpha` subcommand tree that is used to introduce new functionality to the CLI while giving users a cue that the functionality is subject to change. This gives maintainers flexibility while iterating on new CLI features.

The general idea is that new subcommand features should be introduced under the `alpha` subcommand (e.g. `operator-sdk alpha my-command`) and then moved to their own top-level commands once the implementation has matured (e.g. `operator-sdk my-command`).

### `operator-sdk alpha olm init`

Deploys the Operator Lifecycle Manager (OLM). The initial proof-of-concept will initialize OLM using manifests from OLM's repository, using the following URL format, based on provided flags:

`https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/<cluster-type>/manifests/<olm-version>/`

#### Flags for `olm init`

| Flag             | Type   | Description                                                                                     |
|------------------|--------|-------------------------------------------------------------------------------------------------|
| `--cluster-type` | string | The type of cluster to deploy OLM to, one of `upstream`, `ocp`, `okd` (default `upstream`)      |
| `--olm-version`  | string | The version of OLM to deploy (default `latest`)                                                 |
| `--kubeconfig`   | string | Path for custom Kubernetes client config file (overrides the default locations)                 |

### `operator-sdk alpha olm up`

Creates all necessary resources in the cluster to run the operator via OLM, waits for the operator to be deployed by OLM, and tails the operator log, similar to `operator-sdk up local`. When the user terminates the process or if a timeout or error occurs, all of the created resources will be cleaned up.

#### Prerequisites

1. An operator container image built and pushed to a registry accessible to the cluster.
2. An operator bundle on disk (created by `operator-sdk olm-catalog gen-csv`).

#### Flags for `olm up`

| Flag           | Type   | Description                                                                                     |
|----------------|--------|-------------------------------------------------------------------------------------------------|
| `--bundle-dir` | string | The directory containing the operator bundle (default `./deploy/olm-catalog/<operator-name>`)   |
| `--kubeconfig` | string | Path for custom Kubernetes client config file (overrides the default locations)                 |
| `--namespace`  | string | Namespace in which to deploy operator and RBAC rules (overrides namespace from current context) |

#### Resources

OLM uses Kubernetes APIs to learn about the set of operators that are available to be installed and to manage operator lifecycles (i.e. install, upgrade, uninstall). The following resources will be created in the cluster.

| API Kind        | Description  |
|-----------------|--------------|
| `ConfigMap`     | Contains catalog data created from the on-disk operator bundle. |
| `CatalogSource` | Tells OLM where to find operator catalog data. This will refer to the catalog `ConfigMap`. |
| `OperatorGroup` | Tells OLM which namespaces the operator will have RBAC permissions for. We will set this up with the namespace of the current context from the user's `$KUBECONFIG`. |
| `Subscription`  | Tells OLM to manage installation and upgrade of an operator in the namespace in which the `Subscription` is created. We'll create it in the namespace of the current context from the user's `$KUBECONFIG`. |

**Open question:** When the user aborts the process, should we handle cleanup for any of the InstallPlan, CSV, CRD, and CR resources? Which of these will be automatically garbage-collected?

## References

### Operator SDK

* [GitHub][osdk_github]
* [User Guide][osdk_user_guide]
* [CLI Reference][osdk_cli]

### Operator Registry

* [GitHub][registry_github]
* [Manifest format][registry_manifest_format]

### Operator Lifecycle Manager

* [GitHub][olm_github]
* [Architecture][olm_arch]

[osdk_github]: https://github.com/operator-framework/operator-sdk
[osdk_user_guide]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md
[osdk_cli]: https://github.com/operator-framework/operator-sdk/blob/master/doc/sdk-cli-reference.md


[registry_github]: https://github.com/operator-framework/operator-registry
[registry_manifest_format]: https://github.com/operator-framework/operator-registry#manifest-format

[olm_github]: https://github.com/operator-framework/operator-lifecycle-manager
[olm_arch]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/architecture.md

