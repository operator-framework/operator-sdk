---
title: Golang Operator Tutorial
linkTitle: Tutorial
weight: 30
description: An in-depth walkthough of building and running a Go-based operator.
---

**NOTE:** If your project was created with an `operator-sdk` version prior to `v1.0.0`
please [migrate][migration-guide], or consult the [legacy docs][legacy-quickstart-doc].

## Prerequisites

- Go through the [installation guide][install-guide].
- Access to a Kubernetes v1.11.3+ cluster (v1.16.0+ if using `apiextensions.k8s.io/v1` CRDs).
- User authorized with `cluster-admin` permissions.

## Overview

We will create a sample project to let you know how it works and this sample will:

- Create a Memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the Memcached CR spec
- Update the Memcached CR status using the status writer with the names of the memcached pods

## Create a new project

Use the CLI to create a new memcached-operator project:

```sh
mkdir -p $HOME/projects/memcached-operator
cd $HOME/projects/memcached-operator
# we'll use a domain of example.com
# so all API groups will be <group>.example.com
operator-sdk init --domain example.com --repo github.com/example/memcached-operator
```

To learn about the project directory structure, see [Kubebuilder project layout][kubebuilder_layout_doc] doc.

#### A note on dependency management

`operator-sdk init` generates a `go.mod` file to be used with [Go modules][go_mod_wiki]. The `--repo=<path>` flag is required when creating a project outside of `$GOPATH/src`, as scaffolded files require a valid module path. Ensure you [activate module support][activate_modules] by running `export GO111MODULE=on` before using the SDK.

### Manager

The main program for the operator `main.go` initializes and runs the [Manager][manager_go_doc].

See the [Kubebuilder entrypoint doc][kubebuilder_entrypoint_doc] for more details on how the manager registers the Scheme for the custom resource API defintions, and sets up and runs controllers and webhooks.


The Manager can restrict the namespace that all controllers will watch for resources:
```Go
mgr, err := ctrl.NewManager(cfg, manager.Options{Namespace: namespace})
```
By default this will be the namespace that the operator is running in. To watch all namespaces leave the namespace option empty:
```Go
mgr, err := ctrl.NewManager(cfg, manager.Options{Namespace: ""})
```

Read the [operator scope][operator_scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.

## Create a new API and Controller

Create a new Custom Resource Definition(CRD) API with group `cache` version `v1alpha1` and Kind Memcached.
When prompted, enter yes `y` for creating both the resource and controller.

```console
$ operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource --controller
Writing scaffold for you to edit...
api/v1alpha1/memcached_types.go
controllers/memcached_controller.go
...
```

This will scaffold the Memcached resource API at `api/v1alpha1/memcached_types.go` and the controller at `controllers/memcached_controller.go`.

**Note:** This guide will cover the default case of a single group API. If you would like to support Multi-Group APIs see the [Single Group to Multi-Group][multigroup-kubebuilder-doc] doc.

#### Understanding Kubernetes APIs

For an in-depth explanation of Kubernetes APIs and the group-version-kind model, check out these [kubebuilder docs][kb-doc-gkvs].

In general, it's recommended to have one controller responsible for manage each API created for the project to 
properly follow the design goals set by [controller-runtime][controller-runtime].

### Define the API

To begin, we will represent our API by defining the `Memcached` type, which will have a `MemcachedSpec.Size` field to set the quantity of memcached instances (CRs) to be deployed, and a `MemcachedStatus.Nodes` field to store a CR's Pod names.

**Note** The Node field is just to illustrate an example of a Status field. In real cases, it would be recommended to use [Conditions][conditionals].

Define the API for the Memcached Custom Resource(CR) by modifying the Go type definitions at `api/v1alpha1/memcached_types.go` to have the following spec and status:

```Go
// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
	// +kubebuilder:validation:Minimum=0
	// Size is the size of the memcached deployment
	Size int32 `json:"size"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	Nodes []string `json:"nodes"`
}
```

Add the `+kubebuilder:subresource:status` [marker][status_marker] to add a [status subresource][status_subresource] to the CRD manifest so that the controller can update the CR status without changing the rest of the CR object:

```Go
// Memcached is the Schema for the memcacheds API
// +kubebuilder:subresource:status
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedSpec   `json:"spec,omitempty"`
	Status MemcachedStatus `json:"status,omitempty"`
}
```

After modifying the `*_types.go` file always run the following command to update the generated code for that resource type:

```sh
make generate
```

The above makefile target will invoke the [controller-gen][controller_tools] utility to update the `api/v1alpha1/zz_generated.deepcopy.go` file to ensure our API's Go type definitons implement the `runtime.Object` interface that all Kind types must implement.

### Generating CRD manifests

Once the API is defined with spec/status fields and CRD validation markers, the CRD manifests can be generated and updated with the following command:

```sh
make manifests
```

This makefile target will invoke [controller-gen][controller_tools] to generate the CRD manifests at `config/crd/bases/cache.example.com_memcacheds.yaml`.

### OpenAPI validation

OpenAPI validation defined in a CRD ensures CRs are validated based on a set of declarative rules. All CRDs should have validation.
See the [OpenAPI valiation][openapi-validation] doc for details.

## Implement the Controller

For this example replace the generated controller file `controllers/memcached_controller.go` with the example [`memcached_controller.go`][memcached_controller] implementation.

**Note**: The next two subsections explain how the controller watches resources and how the reconcile loop is triggered. Skip to the [Build](#build-and-push-the-image) section to see how to build and run the operator.

### Resources watched by the Controller

The `SetupWithManager()` function in `controllers/memcached_controller.go` specifies how the controller is built to watch a CR and other resources that are owned and managed by that controller.

```Go
import (
	...
	appsv1 "k8s.io/api/apps/v1"
	...
)

func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
```

The `NewControllerManagedBy()` provides a controller builder that allows various controller configurations.

`For(&cachev1alpha1.Memcached{})` specifies the Memcached type as the primary resource to watch. For each Memcached type Add/Update/Delete event the reconcile loop will be sent a reconcile `Request` (a namespace/name key) for that Memcached object.

`Owns(&appsv1.Deployment{})` specifies the Deployments type as the secondary resource to watch. For each Deployment type Add/Update/Delete event, the event handler will map each event to a reconcile `Request` for the owner of the Deployment. Which in this case is the Memcached object for which the Deployment was created.

### Controller Configurations

There are a number of other useful configurations that can be made when initialzing a controller. For more details on these configurations consult the upstream [builder][builder_godocs] and [controller][controller_godocs] godocs.

- Set the max number of concurrent Reconciles for the controller via the [`MaxConcurrentReconciles`][controller_options]  option. Defaults to 1.
  ```Go
  func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
      For(&cachev1alpha1.Memcached{}).
      Owns(&appsv1.Deployment{}).
      WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
      Complete(r)
  }
  ```
- Filter watch events using [predicates][event_filtering]
- Choose the type of [EventHandler][event_handler_godocs] to change how a watch event will translate to reconcile requests for the reconcile loop. For operator relationships that are more complex than primary and secondary resources, the [`EnqueueRequestsFromMapFunc`][enqueue_requests_from_map_func] handler can be used to transform a watch event into an arbitrary set of reconcile requests.

### Reconcile loop

The reconcile function is responsible for enforcing the desired CR state on the actual state of the system. It runs each time an event occurs on a watched CR or resource, and will return some value depending on whether those states match or not.

In this way, every Controller has a Reconciler object with a `Reconcile()` method that implements the reconcile loop. The reconcile loop is passed the [`Request`][request-go-doc] argument which is a Namespace/Name key used to lookup the primary resource object, Memcached, from the cache:

```Go
import (
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/example/memcached-operator/api/v1alpha1"
	...
)

func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  // Lookup the Memcached instance for this reconcile request
  memcached := &cachev1alpha1.Memcached{}
  err := r.Get(ctx, req.NamespacedName, memcached)
  ...
}
```

For a guide on Reconcilers, Clients, and interacting with resource Events, see the [Client API doc][doc_client_api].

The following are a few possible return options for a Reconciler:

- With the error:
  ```go
  return ctrl.Result{}, err
  ```
- Without an error:
  ```go
  return ctrl.Result{Requeue: true}, nil
  ```
- Therefore, to stop the Reconcile, use:
  ```go
  return ctrl.Result{}, nil
  ```
- Reconcile again after X time:
  ```go
   return ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}, nil
   ```

For more details, check the Reconcile and its [Reconcile godoc][reconcile-godoc].

### Specify permissions and generate RBAC manifests

The controller needs certain [RBAC][rbac-k8s-doc] permissions to interact with the resources it manages. These are specified via [RBAC markers][rbac_markers] like the following:

```Go
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  ...
}
```

The `ClusterRole` manifest at `config/rbac/role.yaml` is generated from the above markers via controller-gen with the following command:

```sh
make manifests
```

## Run the Operator

There are three ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by the [Operator Lifecycle Manager (OLM)][doc-olm] in [bundle][quickstart-bundle] format

### 1. Run locally outside the cluster

The following steps will show how to deploy the operator on the Cluster. However, to run locally for development purposes and outside of a Cluster use the target `make install run`.

### 2. Run as a Deployment inside the cluster

### Build and push the image

Before building the operator image, ensure the generated Dockerfile references
the base image you want. You can change the default "runner" image `gcr.io/distroless/static:nonroot`
by replacing its tag with another, for example `alpine:latest`, and removing
the `USER: nonroot:nonroot` directive.

To build and push the operator image, use the following `make` commands.
Make sure to modify the `IMG` arg in the example below to reference a container repository that
you have access to. You can obtain an account for storing containers at
repository sites such quay.io or hub.docker.com. This example uses quay.

Build and push the image:

```sh
export USERNAME=<quay-namespace>
make docker-build docker-push IMG=quay.io/$USERNAME/memcached-operator:v0.0.1
```

**Note**: The name and tag of the image (`IMG=<some-registry>/<project-name>:tag`) in both the commands can also be set in the Makefile.
Modify the line which has `IMG ?= controller:latest` to set your desired default image name.

**Note**: If using an OS which does not point `sh` to the `bash` shell (Ubuntu for example) then you should add the following line to the `Makefile`:

`SHELL := /bin/bash`

This will fix potential issues when the `docker-build` target runs the controller test suite. Issues maybe similar to following error:
`failed to start the controlplane. retried 5 times: fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory occurred`

#### Deploy the operator

By default, a new namespace is created with name `<project-name>-system`, i.e. memcached-operator-system and will be used for the deployment.

Run the following to deploy the operator. This will also install the RBAC manifests from `config/rbac`.

```sh
make deploy IMG=quay.io/$USERNAME/memcached-operator:v0.0.1
```

Verify that the memcached-operator is up and running:

```console
$ kubectl get deployment -n memcached-operator-system
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           8m
```

### 3. Deploy your Operator with OLM

First, install [OLM][doc-olm]:

```sh
operator-sdk olm install
```

Then bundle your operator and push the bundle image:

```sh
make bundle IMG=$OPERATOR_IMG
# Note the "-bundle" component in the image name below.
export BUNDLE_IMG="quay.io/$USERNAME/memcached-operator-bundle:v0.0.1"
make bundle-build BUNDLE_IMG=$BUNDLE_IMG
make docker-push IMG=$BUNDLE_IMG
```

Finally, run your bundle:

```sh
operator-sdk run bundle $BUNDLE_IMG
```

Check out the [docs][quickstart-bundle] for a deep dive into `operator-sdk`'s OLM integration.

## Create a Memcached CR

Update the sample Memcached CR manifest at `config/samples/cache_v1alpha1_memcached.yaml` and define the `spec` as the following:

```YAML
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: memcached-sample
spec:
  size: 3
```

Create the CR:

```sh
kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

Ensure that the memcached operator creates the deployment for the sample CR with the correct size:

```console
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-sample                        3/3     3            3           1m
```

Check the pods and CR status to confirm the status is updated with the memcached pod names:

```console
$ kubectl get pods
NAME                                  READY     STATUS    RESTARTS   AGE
memcached-sample-6fd7c98d8-7dqdr      1/1       Running   0          1m
memcached-sample-6fd7c98d8-g5k7v      1/1       Running   0          1m
memcached-sample-6fd7c98d8-m7vn7      1/1       Running   0          1m
```

```console
$ kubectl get memcached/memcached-sample -o yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  clusterName: ""
  creationTimestamp: 2018-03-31T22:51:08Z
  generation: 0
  name: memcached-sample
  namespace: default
  resourceVersion: "245453"
  selfLink: /apis/cache.example.com/v1alpha1/namespaces/default/memcacheds/memcached-sample
  uid: 0026cc97-3536-11e8-bd83-0800274106a1
spec:
  size: 3
status:
  nodes:
  - memcached-sample-6fd7c98d8-7dqdr
  - memcached-sample-6fd7c98d8-g5k7v
  - memcached-sample-6fd7c98d8-m7vn7
```

### Update the size

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from 3 to 5:

```sh
kubectl patch memcached memcached-sample -p '{"spec":{"size": 5}}' --type=merge
```

Confirm that the operator changes the deployment size:

```console
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-sample                        5/5     5            5           3m
```

### Cleanup

Call the following to delete all deployed resources:

```sh
make undeploy
```

## Further steps

Next, try adding the following to your project:
1. Validating and mutating [admission webhooks][create_a_webhook].
2. Operator packaging and distribution with [OLM][olm-integration].

Also see the [advanced topics][advanced_topics] doc for more use cases and under the hood details. 

[legacy-quickstart-doc]:https://v0-19-x.sdk.operatorframework.io/docs/golang/legacy/quickstart/
[migration-guide]:/docs/building-operators/golang/migration
[install-guide]:/docs/building-operators/golang/installation
[enqueue_requests_from_map_func]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/handler#EnqueueRequestsFromMapFunc
[event_handler_godocs]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/handler#hdr-EventHandlers
[event_filtering]:/docs/building-operators/golang/references/event-filtering/
[controller_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/controller#Options
[controller_godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/controller
[operator_scope]:/docs/building-operators/golang/operator-scope/
[kubebuilder_layout_doc]:https://book.kubebuilder.io/cronjob-tutorial/basic-project.html
[go_mod_wiki]: https://github.com/golang/go/wiki/Modules
[doc_client_api]:/docs/building-operators/golang/references/client/
[manager_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Manager
[request-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Request
[result_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Result
[multi-namespaced-cache-builder]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
[kubebuilder_entrypoint_doc]: https://book.kubebuilder.io/cronjob-tutorial/empty-main.html
[api_terms_doc]: https://book.kubebuilder.io/cronjob-tutorial/gvks.html
[kb_controller_doc]: https://book.kubebuilder.io/cronjob-tutorial/controller-overview.html
[kb_api_doc]: https://book.kubebuilder.io/cronjob-tutorial/new-api.html
[controller_tools]: https://sigs.k8s.io/controller-tools
[doc-validation-schema]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema
[generating-crd]: https://book.kubebuilder.io/reference/generating-crd.html
[markers]: https://book.kubebuilder.io/reference/markers.html
[crd-markers]: https://book.kubebuilder.io/reference/markers/crd-validation.html
[memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/v1.3.0/testdata/go/v3/memcached-operator/controllers/memcached_controller.go
[builder_godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/builder#example-Builder
[activate_modules]: https://github.com/golang/go/wiki/Modules#how-to-install-and-activate-module-support
[advanced_topics]: /docs/building-operators/golang/advanced-topics/
[create_a_webhook]: https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html
[status_marker]: https://book.kubebuilder.io/reference/generating-crd.html#status
[status_subresource]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource
[API-groups]:https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-groups
[legacy_CLI]:https://v0-19-x.sdk.operatorframework.io/docs/cli/
[role-based-access-control]: https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#iam-rolebinding-bootstrap
[multigroup-kubebuilder-doc]: https://book.kubebuilder.io/migration/multi-group.html
[quickstart-bundle]:/docs/olm-integration/quickstart-bundle
[doc-olm]:/docs/olm-integration/quickstart-bundle/#enabling-olm
[conditionals]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[kubernetes-extend-api]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
[reconcile-godoc]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
[rbac-k8s-doc]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[olm-integration]: /docs/olm-integration
[openapi-validation]: /docs/building-operators/golang/references/openapi-validation
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[kb-doc-gkvs]: https://book.kubebuilder.io/cronjob-tutorial/gvks.html