# Overview

This guide walks through the steps required to migrate an operator from an Operator SDK project layout to a Kubebuilder project layout.

The document considers [memcached operator][memcached-operator] as an example to describe the process of migrating an operator built using Operator SDK to a runnable Kubebuilder project.

## Prerequisites
- [git][git_tool].
- [go][go_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.3+.
- [kustomize][kustomize_tool] v3.1.0+.
- Access to a Kubernetes v1.11.3+ cluster.

**Note**
It is recommended that you have your project upgraded to the latest SDK release version before following the steps of this guide to migrate to Kubebuilder layout.

## Install the Operator SDK CLI

Follow the steps in the [installation guide][install_guide] to learn how to install the Operator SDK CLI tool.

## Create a new project

Use `operator-sdk init` command to create a new project.

For example, to create a Memcached operator project:

```sh
$ mkdir $HOME/projects/memcached-operator
$ cd $HOME/projects/memcached-operator
$ operator-sdk init --domain example.com --repo github.com/example-inc/memcached-operator
$ cd memcached-operator
```

Make sure to activate the go module support by running `$ export GO111MODULE=on`.

**Note**
If you intend to have multiple groups in your project, then add the line `multigroup: true` in the `PROJECT` file. The `PROJECT` file for the above example would look like:

```YAML
domain: example.com
repo: github.com/example-inc/memcached-operator
multigroup: true
version: 2
...
```
For multi-group project, the API Go type files are created under `apis/<group>/<version>` and the controllers would be created under `controllers/<group>`.

## Create an API

Create a new API and its corresponding controller.

In case of memcached operator, a new API for kind `Memcached` and group/version  `cache/v1aplha1` is created using the following command.

`operator-sdk api --group cache --version v1alpha1 --kind Memcached`

Press `y` when asked for creating resource and controller. This will scaffold the project and create the files `api/<version>/<kind>_types.go` and `controller/<kind>_types.go`.

For memcached operator project, `api/v1alpha1/memcached_types.go` and `controller/memcached_controller.go` are generated.

### Migrate API type definitions

Copy over the API spec and status from the SDK project's `pkg/apis/<group>/<version>/<kind>_types.go` to Kubebuilder's `api/<version>/<kind>_types.go`.

In our example, `MemcachedSpec` and `MemcachedStatus` from [`pkg/apis/cache/v1aplha1/memcached_types.go`][memcached_types] are copied to `api/v1alpha1/memcached_types.go`.

```go
type MemcachedSpec struct {
	// Specify size, the number of pods
	Size int32 `json:"size"`
}

type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	Nodes []string `json:"nodes"`
}
```
**Note**:
If there are any any libraries or pkgs present in `pkg/apis/cache/v1alpha1`, copy them over to `api/v1alpha1`.

### CRD markers

Include any of the CRD generation or validation markers defined for your API type and fields in `api/<version>/<kind>_types.go`.

For `memcached-operator` the following markers present in `pkg/apis/cache/v1aplha1/memcached_types.go` are copied to `api/v1alpha1/memcached_types.go`.

Example:
```Go
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=memcacheds,scope=Namespaced
```

### Migrate controller

Copy over the `Reconcile()` code and the controller logic from your existing controller at `pkg/controller/<kind>/<kind>_controller.go` to the new controller's reconcile function in Kubebuilder project at `controllers/<kind>_controller.go`.

In our example [`pkg/controller/memcached/memcached_controller.go`][memcached_controller] is copied to `controllers/memcached_controller.go`.

Particularly, note the following changes:

1. The naming convention of the reconciler struct may differ. For example, in case of memcached operator the reconciler struct in SDK project is mentioned as [`ReconcileMemcached`][sdk_reconcile_struct] whereas in Kubebuilder project it would be `MemcachedReconciler`.

```go
type MemcachedReconciler struct {
	Log    logr.Logger
	Client client.Client
	Scheme *runtime.Scheme
}
```

2. Update the import aliases in the `Reconcile()` function and copy over any remaining helper functions from `pkg/controller/<kind>/<kind>_controller.go` to `api/<version>/<kind>_controller.go`.

Refer to [`memcached_controller.go`][kb_memcached_controller] for the example implementation of `Reconcile()` logic which would be present in `api/v1alpha1/memcached_controller.go` of Memcached operator.

## Generate CRDs

Run [`make manifests`][generate_crd] to generate CRD manifests. They would be generated inside the `config/crd/bases` folder.

## Operator Manifests

### Operator deployment manifests

The operator deployment manifest [`deploy/operator.yaml`][deployment_yaml] from the old project should be copied over to `config/manager/manager.yaml` in the new project.

**Note**:
The kustomize file requires the operator deployment manifest to have the field `namespace` which is missing in the `deploy/operator.yaml` manifest of SDK project.
> **// TODO:** Explain label propagation for metrics collection.

### RBAC permissions

The RBAC manifests present in `config/rbac/` of a Kubebuilder project are generated through the [RBAC markers][rbac_markers] present in `controller/<kind>_controller.go`. They can be added as comments above the `Reconcile()` method.

In our example of memcached operator, add the following RBAC markers in `memcached_controller.go`:

```Go

...
// +kubebuilder:rbac:groups=apps,resources=deployments;pods;daemonsets;replicasets;statefulsets,verbs=get;update;patch;list;create;delete;watch

func (r *MemcachedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Memcached")

...
```
**Note**:
To update `config/rbac/role.yaml` after changing the markers, run `make manifests`.

The project can now be built and the operator can be deployed on cluster. For further steps regarding the deployment of operator, creation of custom resource and cleaning up of resources, refer to [quickstart guide][kb_quickstart].


[memcached-operator]: /docs/golang/quickstart.md
[git_tool]: https://git-scm.com/downloads
[go_tool]: https://golang.org/dl/
[kubectl_tool]: https://github.com/kubernetes/minikube#installation
[kustomize_tool]: https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md
[kubebuilder_install]: https://book.kubebuilder.io/quick-start.html#installation
[memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/8323f56b91590c3bc8098e0024aa825e95386c8a/example/memcached-operator/memcached_controller.go.tmpl
[sdk_reconcile_struct]: https://github.com/operator-framework/operator-sdk/blob/8323f56b91590c3bc8098e0024aa825e95386c8a/example/memcached-operator/memcached_controller.go.tmpl#L74-L80
[generate_crd]: https://book.kubebuilder.io/reference/generating-crd.html?highlight=make,mani#generating-crds
[deployment_yaml]: https://github.com/operator-framework/operator-sdk-samples/blob/master/go/memcached-operator/deploy/operator.yaml
[rbac_markers]: https://book.kubebuilder.io/reference/markers/rbac.html
[memcached_cr]: https://github.com/operator-framework/operator-sdk-samples/blob/master/go/memcached-operator/deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
[memcached_types]: https://github.com/operator-framework/operator-sdk-samples/blob/master/go/memcached-operator/pkg/apis/cache/v1alpha1/memcached_types.go
[kb_memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/master/example/kb-memcached-operator/memcached_controller.go.tmpl
[kb_quickstart]: /docs/kubebuilder/quickstart.md
[install_guide]: /docs/install-operator-sdk.md
