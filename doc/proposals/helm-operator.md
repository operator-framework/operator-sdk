## Helm Operator Proposal

### Background

As was mentioned in the [Ansible Operator Proposal](./ansible-operator.md), not everyone is a golang developer, so the SDK needs to support other types of operators to gain adoption across a wider community of users.

[Helm](https://helm.sh/) is one of the most widely-used tools for Kubernetes application management, and it bills itself as the "package manager for Kubernetes." Operators serve a nearly identical function, but they improve on Helm's concepts by incorporating an always-on reconciliation loop rather than relying on an imperative user-driven command line tool. By integrating Helm's templating engine and release management into an operator, the SDK will further increase the number of potential users by adding the ability to deploy Charts (e.g. from Helm's [large catalog of existing Charts](https://github.com/helm/charts)) as operators with very little extra effort.

### Goals

The goal of the Helm Operator will be to create a fully functional framework for Helm Chart developers to create operators. It will also expose a library for golang users to use Helm in their operator if they so choose. These two goals in conjunction will allow users to select the best technology for their project or skillset.

### New Operator Type

This proposal creates a new type of operator called `helm`. The new type is used to tell the tooling to act on that type of operator.

### Package Structure

Packages will be added to the operator-sdk. These packages are designed to be usable by the end user if they choose to and should have a well documented public API. The proposed packages are:

* /operator-sdk/pkg/helm/client
  * Will contain a helper function to create a Helm client from `controller-runtime` manager.

* /operator-sdk/pkg/helm/controller
  * Will contain an exported `HelmOperatorReconciler` that implements the `controller-runtime` `reconcile.Reconciler` interface.
  * Will contain an exported `Add` function that creates a controller using the `HelmOperatorReconciler` and adds watches based on a set of watch options passed to the `Add` function.

* /operator-sdk/pkg/helm/engine
  * Will contain a Helm Engine implementation that adds owner references to generated Kubernetes resource assets, which is necessary for garbage collection of Helm chart resources.

* /operator-sdk/pkg/helm/internal
  * Will contain types and utilities used by other Helm packages in the SDK.

* /operator-sdk/pkg/helm/release
  * Will contain the `Manager` types and interfaces. A `Manager` is responsible for:
    * Implementing Helm's Tiller functions that are necessary to install, update, and uninstall releases.
    * Reconciling an existing release's resources.
  * A default `Manager` implementation is provided in this package but is not exported.
  * Package functions:
    * `NewManager` - function that returns a new Manager for a provided helm chart.
    * `NewManagersFromEnv` - function that returns a map of GVK to Manager types based on environment variables.
    * `NewManagersFromFile` - function that returns a map of GVK to Manager types based on a provided config file.

### Commands

We are adding and updating existing commands to accommodate the Helm operator.  Changes to the `cmd` package as well as changes to the generator are needed.

#### New

New functionality will be updates to allow Helm operator developers to create a new boilerplate operator structure with everything necessary to get started developing and deploying a Helm operator with the SDK.

```
operator-sdk new <project-name> --type=helm --kind=<kind> --api-version=<group/version>
```

Flags:
* `--type=helm` is required to create Helm operator project.
* **Required:** --kind - the kind for the CRD.
* **Required:** --api-version - the group/version for the CRD.

This will be new scaffolding for the above command under the hood. We will:
* Create a `./<project-name>` directory.
* Create a `./<project-name>/helm-charts` directory.
* Generate a simple default chart at `./<project-name>/helm-charts/<kind>`.
* Create a new watches file at `./<project-name>/watches.yaml`. The chart and GVK will be defaulted based on input to the `new` command.
* Create a `./<project-name>/deploy` with the Kubernetes resource files necessary to run the operator.
* Create a `./build/Dockerfile` that uses the watches file and the helm chart. It will use the Helm operator as its base image.

The resulting structure will be:

```
<project-name>
|   watches.yaml
|
|-- build
|   |   Dockerfile
|
|-- helm-charts
|   |-- <kind>
|       |   Chart.yaml
|       |   ...
|
|-- deploy
|   |   operator.yaml
|   |   role_binding.yaml
|   |   role.yaml
|   |   service_account.yaml
|   |
|   |-- crds
|       |   <gvk>_crd.yaml
|       |   <gvk>_cr.yaml
```

The SDK CLI will use the presence of the `helm-charts` directory to detect a `helm` type project.

#### Add

Add functionality will be updated to allow Helm operator developers to add new CRDs/CRs and to update the watches.yaml file for additional Helm charts. The command helps when a user wants to watch more than one CRD for their operator.

```
operator-sdk add crd --api-version=<group>/<version> --kind=<kind> --update-watches=<true|false>
```

Flags:
* **Required:** --kind - the kind for the CRD.
* **Required:** --api-version - the group/version for the CRD.
* **Optional:** --update-watches - whether or not to update watches.yaml file (default: false).

**NOTE:** `operator-sdk add` subcommands `api` and `controller` will not be supported, since they are only valid for Go operators. Running these subcommands in a Helm operator project will result in an error.

#### Up

Up functionality will be updated to allow Helm operator developers to run their operator locally, using the `operator-sdk` binary's built-in helm operator implementation.

```
operator-sdk up local
```

This should use the known structure and the helm operator code to run the operator from this location. The existing code will need to be updated with a new operator type check for `helm` (in addition to existing `go` and `ansible` types). The command works by running the operator-sdk binary, which includes the Helm operator code, as the operator process.

#### Build

Build functionality will be updated to support building a docker image from the Helm operator directory structure.

```
operator-sdk build <image-name>
```

#### Test

The SDK `test` command currently only supports Go projects, so there will be no support for the `operator-sdk test` subcommand in the initial integration of the Helm operator.

### Base Image

The SDK team will maintain a build job for the `helm-operator` base image with the following tagging methodology:
* Builds on the master branch that pass nightly CI tests will be tagged with `:master`
* Builds for tags that pass CI will be tagged with `:<tag>`. If the tag is also the greatest semantic version for the repository, the image will also be tagged with `:latest`.

The go binary included in the base image will be built with `GOOS=linux` and `GOARCH=amd64`.

The base image repository will be `quay.io/water-hole/helm-operator`.

### Observations and open questions

* There will be a large amount of overlap in the `operator-sdk` commands for the Ansible and Helm operators. We should take care to extract the resusable features of the Ansible operator commands into a shared library, usable by both Helm and Ansible commands.

* There is a moderate amount of complexity already related to how operator types are handled between the `go` and `ansible` types. With the addition of a third type, there may need to be a larger design proposal for operator types. For example, do we need to define an `Operator` interface that each of the operator types can implement for flag verification, scaffolding, project detection, etc.?
