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

Implement a proof-of-concept with an initial set of CLI additions to support a simplified workflow for an Operator SDK user to run an Operator SDK-developed project in a cluster using the Operator Lifecyle Manager (OLM).

## Use cases

1. As an operator developer, I want to deploy OLM to a development cluster.
2. As an operator developer, I want to create the resources necessary for running my operator via OLM.
3. As an operator developer, I want to run my operator via OLM.

## Proposed CLI commands

### `operator-sdk alpha`

Many other Kubernetes CLI projects (e.g. `kubebuilder`, `kubectl`, and `kubeadm`) have an `alpha` subcommand tree that is used to introduce new functionality to the CLI while giving users a cue that the functionality is subject to change. This gives maintainers flexibility while iterating on new CLI features.

The general idea is that new subcommand features should be introduced under the `alpha` subcommand (e.g. `operator-sdk alpha my-command`) and then moved to their own top-level commands once the implementation has matured (e.g. `operator-sdk my-command`).

### `operator-sdk alpha olm init`

Checks cluster facts based on the current kubeconfig context (e.g. cluster type, version, OLM status), and ensures OLM is running. Using this command with unsupported clusters results in a failure.

#### OCP/OKD

For clusters where OLM is expected to be installed by default, this command will print OLM's current status and exit.

#### Upstream Kubernetes

For other clusters that meet OLM's prerequisites, the command will check to see if OLM is already deployed. If so, it will print OLM's current status and exit. If not, it will deploy OLM using the `upstream` manifests from OLM's repository at the version specified by the `--olm-version` flag, based on the following URL pattern:

`https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/manifests/<olm-version>/`

Each resource that is applied will be logged so that users can manually backout the OLM deployment if desired. Once the manifests have been applied, the command will wait until OLM is running (or until a specified timeout) and then print its status.

#### Flags for `olm init`

| Flag             | Type   | Description                                                                                     |
|------------------|--------|-------------------------------------------------------------------------------------------------|
| `--olm-version`  | string | The version of OLM to deploy (default `latest`)                                                 |
| `--timeout`      | string | Timeout duration to wait for OLM to become ready before outputting status (default 60s)         |
| `--kubeconfig`   | string | Path for custom Kubernetes client config file (overrides the default locations)                 |

### `operator-sdk alpha olm up`

Creates all necessary resources in the cluster to run the operator via OLM, waits for the operator to be deployed by OLM, and tails the operator log, similar to `operator-sdk up local`. When the user terminates the process or if a timeout or error occurs, all of the created resources will be cleaned up.

Different clusters install OLM in different namespaces. Since `olm up` may need to create resources in these namespaces, the cluster fact collection used in `operator-sdk alpha olm init` should be used here as well to determine which OLM namespaces are in use. This cluster fact collection package should be maintained in a separate shared package within the SDK repo, such that it is decoupled and could be extracted into a separate `operator-framework-tools` (or similar) repo in the future.

#### Prerequisites

1. The operator container image referenced by the CSV is available to the cluster.
2. An operator bundle on disk (created by `operator-sdk olm-catalog gen-csv`).

#### Flags for `olm up`

| Flag             | Type   | Description                                                                                     |
|------------------|--------|-------------------------------------------------------------------------------------------------|
| `--bundle-dir`   | string | The directory containing the operator bundle (default `./deploy/olm-catalog/<operator-name>`)   |
| `--install-mode` | string | The [`InstallMode`][olm_install_modes] to use when running the operator, one of `OwnNamespace`, `SingleNamespace`, `MultiNamespace`, or `AllNamespaces` (default `OwnNamespace`). If using `MultiNamespace`, users can define the `OperatorGroup` target namespaces with `--install-mode=MultiNamespace=ns1,ns2,nsN`  |
| `--kubeconfig`   | string | Path for custom Kubernetes client config file (overrides the default locations)                 |
| `--namespace`    | string | Namespace in which to deploy operator and RBAC rules (overrides namespace from current context). We'll verify this is compatible with the defined install mode. |

#### Resources

OLM uses Kubernetes APIs to learn about the set of operators that are available to be installed and to manage operator lifecycles (i.e. install, upgrade, uninstall). The following resources will be created in the cluster.

| API Kind        | Description  |
|-----------------|--------------|
| `ConfigMap`     | Contains catalog data created from the on-disk operator bundle. |
| `CatalogSource` | Tells OLM where to find operator catalog data. This will refer to the catalog `ConfigMap`. |
| `OperatorGroup` | Tells OLM which namespaces the operator will have RBAC permissions for. We'll configure it based on the `--install-mode` and `--namespace` flags. |
| `Subscription`  | Tells OLM to manage installation and upgrade of an operator in the namespace in which the `Subscription` is created. We'll create it based on the value of the `--namespace` flag. |

**Open questions:** 
1. When the user aborts the process, should we handle cleanup for any of the InstallPlan, CSV, CRD, and CR resources? Which of these will be automatically garbage-collected?

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
[olm_install_modes]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/operatorgroups.md#installmodes-and-supported-operatorgroups

