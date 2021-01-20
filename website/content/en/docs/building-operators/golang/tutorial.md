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

It is also possible to use the [MultiNamespacedCacheBuilder][multi-namespaced-cache-builder] to watch a specific set of namespaces:
```Go
var namespaces []string // List of Namespaces
// Create a new Cmd to provide shared dependencies and start components
mgr, err := ctrl.NewManager(cfg, manager.Options{
   NewCache: cache.MultiNamespacedCacheBuilder(namespaces),
})
```

#### Operator scope

Read the [operator scope][operator_scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.

### Multi-Group APIs

Before creating an API and controller, consider if your operator requires multiple API [groups][api-groups]. Then to change the layout of your project to support multi-group run the command `operator-sdk edit --multigroup`. It will update the `PROJECT` file which should look like the following:

```YAML
domain: example.com
layout: go.kubebuilder.io/v3
multigroup: true
...
```
For multi-group projects, the API Go type files are created under `apis/<group>/<version>/` and the controllers under `controllers/<group>/` and then, the Dockerfile will be updated accordingly. For further information see the [multi-group migration doc][multigroup-kubebuilder-doc]

This guide will cover the default case of a single group API.

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

See the [API terminology doc][api_terms_doc] for details on the CRD API conventions.

To understand the API Go types and controller scaffolding see the Kubebuilder [api doc][kb_api_doc] and [controller doc][kb_controller_doc].

### Define the API

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

This makefile target will invoke controller-gen to generate the CRD manifests at `config/crd/bases/cache.example.com_memcacheds.yaml`.

#### OpenAPI validation

OpenAPIv3 schemas are added to CRD manifests in the `spec.validation` block when the manifests are generated. This validation block allows Kubernetes to validate the properties in a Memcached Custom Resource when it is created or updated.

Markers (annotations) are available to configure validations for your API. These markers will always have a `+kubebuilder:validation` prefix.

Usage of markers in API code is discussed in the kubebuilder [CRD generation][generating-crd] and [marker][markers] documentation. A full list of OpenAPIv3 validation markers can be found [here][crd-markers].

To learn more about OpenAPI v3.0 validation schemas in CRDs, refer to the [Kubernetes Documentation][doc-validation-schema].

### Implement the Controller

For this example replace the generated controller file `controllers/memcached_controller.go` with the example [`memcached_controller.go`][memcached_controller] implementation.

The example controller executes the following reconciliation logic for each Memcached CR:
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the Memcached CR spec
- Update the Memcached CR status using the status writer with the names of the memcached pods

The next two subsections explain how the controller watches resources and how the reconcile loop is triggered. Skip to the ["run the Operator"](#run-the-operator) section to see how to build and run the operator.

#### Resources watched by the Controller

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

#### Controller Configurations

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


#### Reconcile loop

Every Controller has a Reconciler object with a `Reconcile()` method that implements the reconcile loop. The reconcile loop is passed the [`Request`][request-go-doc] argument which is a Namespace/Name key used to lookup the primary resource object, Memcached, from the cache:

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

Based on the return values, [`Result`][result_go_doc] and error, the `Request` may be requeued and the reconcile loop may be triggered again:

```Go
// Reconcile successful - don't requeue
return ctrl.Result{}, nil
// Reconcile failed due to error - requeue
return ctrl.Result{}, err
// Requeue for any reason other than an error
return ctrl.Result{Requeue: true}, nil
```

You can set the `Result.RequeueAfter` to requeue the `Request` after a grace period as well:
```Go
import "time"

// Reconcile for any reason other than an error after 5 seconds
return ctrl.Result{RequeueAfter: time.Second*5}, nil
```

**Note:** Returning `Result` with `RequeueAfter` set is how you can periodically reconcile a CR.

For a guide on Reconcilers, Clients, and interacting with resource Events, see the [Client API doc][doc_client_api].

### Specify permissions and generate RBAC manifests

The controller needs certain RBAC permissions to interact with the resources it manages. These are specified via [RBAC markers][rbac_markers] like the following:

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

Execute the following command, which install your CRDs and run the manager locally:

```sh
make install run ENABLE_WEBHOOKS=false
```

### 2. Run as a Deployment inside the cluster

#### Build and push the image

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

*NOTE* If you have enabled webhooks in your deployments, you will need to have cert-manager already installed
in the cluster or `make deploy` will fail when creating the cert-manager resources.

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

The following guides build off the operator created in this example, adding advanced features:

- [Create a validating or mutating Admission Webhook][create_a_webhook]

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
[memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/v1.2.0/testdata/go/memcached-operator/controllers/memcached_controller.go
[builder_godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/builder#example-Builder
[activate_modules]: https://github.com/golang/go/wiki/Modules#how-to-install-and-activate-module-support
[advanced_topics]: /docs/building-operators/golang/advanced-topics/
[create_a_webhook]: https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html
[status_marker]: https://book.kubebuilder.io/reference/generating-crd.html#status
[status_subresource]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource
[env-test-setup]: /docs/building-operators/golang/references/envtest-setup
[role-based-access-control]: https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#iam-rolebinding-bootstrap
[multigroup-kubebuilder-doc]: https://book.kubebuilder.io/migration/multi-group.html
[quickstart-bundle]:/docs/olm-integration/quickstart-bundle
[doc-olm]:/docs/olm-integration/quickstart-bundle/#enabling-olm
