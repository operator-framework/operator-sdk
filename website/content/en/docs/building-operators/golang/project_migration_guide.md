---
title: Migrating Legacy Projects
linkTitle: Migrating Projects to 1.0.0+
weight: 200
description: Instructions for migrating a legacy Go-based project to use the new Kubebuilder-style layout.
---

## Overview

The motivations for the new layout are related to bringing more flexibility to users and 
part of the process to [integrate Kubebuilder and Operator SDK][integration-doc].

### What was changed
 
- The `deploy` directory was replaced with the `config` directory including a new layout of Kubernetes manifests files:
    * CRD manifests in `deploy/crds/` are now in `config/crd/bases`
    * CR manifests in `deploy/crds/` are now in `config/samples`
    * Controller manifest `deploy/operator.yaml` is now in `config/manager/manager.yaml` 
    * RBAC manifests in `deploy` are now in `config/rbac/`
    
- `build/Dockerfile` is moved to `Dockerfile` in the project root directory
- `pkg/apis` and `pkg/controllers` are now in the root directory. 
- `cmd/manager/main.go` is now in the root directory. 

### What is new

Projects are now scaffold using:

- [kustomize][kustomize] to manage Kubernetes resources needed to deploy your operator
- A `Makefile` with helpful targets for build, test, and deployment, and to give you flexibility to tailor things to your project's needs
- Helpers and options to work with webhooks
- Updated metrics configuration using [kube-auth-proxy][kube-auth-proxy], a `--metrics-addr` flag, and [kustomize][kustomize]-based deployment of a Kubernetes `Service` and prometheus operator `ServiceMonitor`
- Scaffolded tests that use the [`envtest`][envtest] test framework

## Migration Steps

The most straightforward migration path is to:
1. Create a new project from scratch to let `operator-sdk` scaffold the new project.
2. Copy your existing code and configuration into the new project structure.

**Note:** It is recommended that you have your project upgraded to the latest SDK release version before following the 
steps of this guide to migrate to new layout.

### Create a new project

In Kubebuilder-style projects, CRD groups are defined using two different flags
(`--group` and `--domain`).

When we initialize a new project, we need to specify the domain that _all_ APIs in
our project will share, so before creating the new project, we need to determine which
domain we're using for the APIs in our existing project.

To determine the domain, look at the `spec.group` field in your CRDs in the
`deploy/crds` directory.

The domain is everything after the first DNS segment. Using `cache.example.com` as an
example, the `--domain` would be `example.com`.

So let's create a new project with the same domain (`example.com`):

```sh
mkdir memcached-operator
cd memcached-operator
operator-sdk init --domain example.com --repo github.com/example-inc/memcached-operator
```

**Note**: `operator-sdk` attempts to automatically discover the Go module path of your project by looking for a `go.mod` file, or if in `$GOPATH`, by using the directory path. Use the `--repo` flag to explicitly set the module path.

## Check if your project is multi-group

Before we start to create the APIs, check if your project has more than one group such as : `foo.example.com/v1` and `crew.example.com/v1`. If you intend to still working on with multiple groups in your project, then add the line 
`multigroup: true` in the `PROJECT` file. The `PROJECT` file for the above example would look like:

```YAML
domain: example.com
repo: github.com/example-inc/memcached-operator
multigroup: true
version: 2
...
```

**Note:** In multi-group projects, APIs are defined in `apis/<group>/<version>` and controllers are defined in `controllers/<group>`.

## Migrate APIs and Controllers

Now that we have our new project initialized, we need to re-create each of our APIs. 
Using our API example from earlier (`cache.example.com`), we'll use `cache` for the
`--group`, `v1alpha1` for the `--version` and `Memcached` for `--kind` flag.

For each API in the existing project, run:

```sh
operator-sdk create api \
    --group=cache \
    --version=<version> \
    --kind=<Kind> \
    --resource \
    --controller
```

### API's

Now let’s copy the API definition from `pkg/apis/<group>/<version>/<kind>_types.go` to `api/<version>/<kind>_types.go`. For our example, it is only required to copy the code from the `Spec` and `Status` fields. 

Observer that this file is quite similar to the old one. You ought to copy all for your new API and it will be pretty much verbatim, however, it requires some attention with the [Markers][markers]:

- The `+k8s:deepcopy-gen:interfaces=...` marker was replaced with `+kubebuilder:object:root=true`.
- If you are not using [openapi-gen][openapi-gen] to generate OpenAPI Go code, then `// +k8s:openapi-gen=true` and other related openapi markers can be removed.

**NOTE** The `operator-sdk generate openapi` command was deprecated in `0.13.0` and was removed from `0.17` SDK release version. So far, it is recommended to use [openapi-gen][openapi-gen] directly for OpenAPI code generation. 

Our Memcached API types will look like:

```go
// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
	// Size is the size of the memcached deployment
	Size int32 `json:"size"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	Nodes []string `json:"nodes"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Memcached is the Schema for the memcacheds API
type Memcached struct {...}

// +kubebuilder:object:root=true

// MemcachedList contains a list of Memcached
type MemcachedList struct {...}
```

### Controllers

Now let’s migrate the controller code from `pkg/controller/<kind>/<kind>_controller.go` to `controllers/<kind>_controller.go`. Following the steps:

1. Copy over any struct fields from the existing project into the new `<Kind>Reconciler` struct. 
**Note** The `Reconciler` struct has been renamed from `Reconcile<Kind>` to `<Kind>Reconciler`. In our example, we would see `ReconcileMemcached` instead of `MemcachedReconciler`. 
2. Replace the `// your logic here` in the new layout with your reconcile logic. 
3. Copy the code under `func add(mgr manager.Manager, r reconcile.Reconciler)` to `func SetupWithManager`: 
```go
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
```

In our example, the `Watch` implemented for the Deployment will be replaced with `Owns(&appsv1.Deployment{})`. Setting up controller `Watches` is simplified in more recent versions of controller-runtime, which has controller [Builder][builder] helpers to handle more of the details.

### Set the RBAC permissions

The RBAC permissions are now configured via [RBAC markers][rbac_markers], which are used to generate and update the manifest files present in `config/rbac/`. These markers can be found (and should be defined) on the `Reconcile()` method of each controller.

In the Memcached example, they look like the following: 

```go
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list
```

To update `config/rbac/role.yaml` after changing the markers, run `make manifests`.  

By default, new projects are cluster-scoped (i.e. they have cluster-scoped permissions and watch all namespaces). Read the [operator scope documentation][operator-scope] for more information about changing the scope of your operator.

See the complete migrated `memecached_controller.go` code [here][memcached_controller].

## Migrate `main.go`

By checking our new `main.go` we will find that:

- The SDK [leader.Become][leader-lib-doc] was replaced by the [controller-runtime's leader][controller-runtime-leader] with lease mechanism. However, you still able to stick with the [leader.Become][leader-lib-doc] for life if you wish:
 
```go
func main() {
...
	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "memcached-operator-lock")
	if err != nil {
    	log.Error(err, "")
    	os.Exit(1)
	}
..
```
 
In order to use the previous one ensure that you have the [operator-lib][operator-lib] as a dependency of your project.

- The default port used by the metric endpoint binds to from `:8383` to `:8080`. To continue using port `8383`, specify `--metrics-addr=:8383` when you start the operator. 

- `OPERATOR_NAME` and `POD_NAME` environment variable are no longer used. `OPERATOR_NAME` was used to define the name for a leader election config map. Operator authors should use the `LeaderElectionID` attribute from the [Manager Options][ctrl-options] which is set hardcoded into the `main.go`:
 
```go
func main() {
...
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "f1c5ece8.example.com",
	})
...
```

- Ensure that you copy all customizations made in `cmd/manager/main.go` to `main.go`. You’ll also need to ensure that all needed schemes have been registered, if you have been using third-party API's (i.e Route Api from OpenShift).

### Configuring your test environment

[Setup the `envtest` binaries and environment][envtest-setup] for your project.
Update your `test` Makefile target to the following:

```sh
# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out
```

## Migrate your tests

For the new layout, you will see that `controllers/suite_test.go` is created. This file contains boilerplate for executing integration tests using [envtest][envtest] with [ginkgo](https://onsi.github.io/ginkgo/) and [gomega](https://onsi.github.io/gomega/).

Operator SDK 1.0.0+ removes support for the legacy test framework and no longer supports the `operator-sdk test` subcommand. All affected tests should be migrated to use `envtest`.

The Operator SDK project recommends controller-runtime's [envtest][envtest] because it has a more active contributor community, it has become more mature than Operator SDK's test framework, and it does not require an actual cluster to run tests, which can be a huge benefit in CI scenarios. 

To learn more about how you can test your controllers, see the documentation about [writing controller tests][writing-controller-tests].

## Migrate your Custom Resources

Custom resource samples are stored in `./config/samples` in the new project structure. Copy the examples from your existing project into this directory. In existing projects, CR files have the format `./deploy/crds/<group>.<domain>_<version>_<kind>_cr.yaml`.

In our example, we'll copy the specs from `deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml` 
to `config/samples/cache_v1alpha1_memcached.yaml`

## Configure your Operator

In case your project has customizations in the `deploy/operator.yaml` then, it needs to be port to 
`config/manager/manager.yaml`. Note that, `OPERATOR_NAME` and `POD_NAME` env vars are no longer used. For further information came back to the section [Migrate `main.go` ][migration-guide-main-section].

## Export Metrics 

If you are using metrics and would like to keep them exported, see that the `func addMetrics()` is no longer generated in the `main.go` and it is now configurable via [kustomize][kustomize]. Following the steps.

### Configure Prometheus metrics

- Ensure that you have Prometheus installed in the cluster:
To check if you have the required API resource to create the `ServiceMonitor` run: 
```sh
kubectl api-resources | grep servicemonitors
```
If not, you can install Prometheus via [kube-prometheus](https://github.com/coreos/kube-prometheus#installing): 
```sh
kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/release-0.33/bundle.yaml
```
- Now uncomment the line `- ../prometheus` in the `config/default/kustomization.yaml` file. It creates the `ServiceMonitor` resource which enables exporting the metrics:
```sh
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
- ../prometheus
```

### Use Handler from `operator-lib` 

By using the [InstrumentedEnqueueRequestForObject](https://pkg.go.dev/github.com/operator-framework/operator-lib@v0.1.0/handler?tab=doc#InstrumentedEnqueueRequestForObject) you will able to export metrics from your Custom Resources.  In our example, it would like:  

```go
import (
    ...
	"github.com/operator-framework/operator-lib/handler"
    ...
)

func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a new controller
	c, err := controller.New("memcached-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
    ...
	err = c.Watch(&source.Kind{Type: &cachev1alpha1.Memcached{}}, &handler.InstrumentedEnqueueRequestForObject{})
	if err != nil {
		return err
	}
	...
	return nil
}
```

**Note** Ensure that you have the [operator-lib][operator-lib] added to your `go.mod`.

In this way, the following metric with the resource info will be exported:

```shell             
resource_created_at_seconds{"name", "namespace", "group", "version", "kind"}
```

**Note:** To check it you can create a pod to curl the `metrics/` endpoint but note that it is now protected by the [kube-auth-proxy][kube-auth-proxy] which means that you will need to create a `ClusterRoleBinding` and obtained the token from the ServiceAccount's secret which will be used in the requests. Otherwise, to test you can disable the [kube-auth-proxy][kube-auth-proxy] as well.

For more info see the [metrics][metrics].
 
## Operator image 

The Dockerfile image also changes and now it is a `multi-stage`, `distroless` and still been `rootless`, however, users can change it to work as however they want.
 
 See that, you might need to port some customizations made in your old Dockerfile as well. Also, if you wish to still using the previous UBI image replace:

```sh
# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
```

With:

```sh
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
```

## Generate Manifests and Build the operator

Note that:

- `operator-sdk generate crds` is replaced with `make manifests`, which generates CRDs and RBAC rules.
- `operator-sdk build` is replaced with `make docker-build IMG=<some-registry>/<project-name>:tag`.

In this way, run:

```sh
make manifests
make docker-build IMG=<some-registry>/<project-name>:<tag>
```

## Verify the migration

The project can now be built, and the operator can be deployed on-cluster. For further steps regarding the deployment of the operator, creation of custom resources, and cleaning up of resources, see the [quickstart guide][quickstart].

Note that, you also can troubleshooting by checking the container logs. 
E.g `kubectl logs deployment.apps/memcached-operator-controller-manager -n memcached-operator-system -c manager`  

[quickstart-legacy]: https://v0-19-x.sdk.operatorframework.io/docs/golang/legacy/quickstart/
[integration-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
[quickstart]: /docs/building-operators/golang/quickstart/
[metrics]: https://book.kubebuilder.io/reference/metrics.html?highlight=metr#metrics
[memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/master/example/memcached-operator/memcached_controller.go.tmpl
[rbac_markers]: https://book.kubebuilder.io/reference/markers/rbac.html
[envtest-setup]: /docs/building-operators/golang/references/envtest-setup
[kube-auth-proxy]: https://github.com/brancz/kube-rbac-proxy
[markers]: https://book.kubebuilder.io/reference/markers.html?highlight=markers#marker-syntax
[operator-scope]: /docs/building-operators/golang/operator-scope
[leaderElection]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/leaderelection?tab=doc
[ginkgo]: https://onsi.github.io/ginkgo/
[gomega]: https://onsi.github.io/gomega/
[builder]: https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.6.1/pkg/builder?tab=doc
[writing-controller-tests]: https://book.kubebuilder.io/reference/writing-tests.html?highlight=controller,test#writing-controller-tests
[openapi-gen]: https://github.com/kubernetes/kube-openapi/tree/master/cmd/openapi-gen 
[controller-runtime-leader]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#LeaderElectionRunnable
[operator-lib]: https://github.com/operator-framework/operator-lib/
[leader-lib-doc]: https://pkg.go.dev/github.com/operator-framework/operator-lib@v0.1.0/leader?tab=doc
[migration-guide-main-section]: /docs/building-operators/golang/project_migration_guide/#migrate-maingo
[kustomize]: https://github.com/kubernetes-sigs/kustomize 
[ctrl-options]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/manager#Options 
[envtest]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest