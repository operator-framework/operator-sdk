---
title: Golang Based Operator Quickstart
linkTitle: Quickstart
weight: 1
---

This guide walks through an example of building a simple memcached-operator using the operator-sdk CLI tool and controller-runtime library API. 

## Create a new project

Use the CLI to create a new memcached-operator project:

```sh
$ mkdir -p $HOME/projects/memcached-operator
$ cd $HOME/projects/memcached-operator
# we'll use a domain of example.com
# so all API groups will be <group>.tutorial.kubebuilder.io.
$ operator-sdk init memcached-operator --domain=example.com --repo=github.com/example-inc/memcached-operator
```

To learn about the project directory structure, see [Kubebuilder project layout][kubebuilder_layout_doc] doc.

#### A note on dependency management

`operator-sdk init` generates a `go.mod` file to be used with [Go modules][go_mod_wiki]. The `--repo=<path>` flag is required when creating a project outside of `$GOPATH/src`, as scaffolded files require a valid module path. Ensure you activate module support before using the SDK. From the [Go modules Wiki][go_mod_wiki]:

> You can activate module support in one of two ways:
> - Invoke the go command in a directory with a valid go.mod file in the current directory or any parent of it and the environment variable GO111MODULE unset (or explicitly set to auto).
> - Invoke the go command with GO111MODULE=on environment variable set.

### Manager
The main program for the operator `main.go` initializes and runs the [Manager][manager_go_doc].

See the [Kubebuilder entrypoint doc][kubebuilder_entrypoint_doc] for more details on how the manager registers the Scheme for the custom resource API defintions, and sets up and runs controllers and webhooks.


The Manager can restrict the namespace that all controllers will watch for resources:
```Go
mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
```
By default this will be the namespace that the operator is running in. To watch all namespaces leave the namespace option empty:
```Go
mgr, err := manager.New(cfg, manager.Options{Namespace: ""})
```

It is also possible to use the [MultiNamespacedCacheBuilder][multi-namespaced-cache-builder] to watch a specific set of namespaces:
```Go
var namespaces []string // List of Namespaces
// Create a new Cmd to provide shared dependencies and start components
mgr, err := manager.New(cfg, manager.Options{
   NewCache: cache.MultiNamespacedCacheBuilder(namespaces),
})
```

#### Operator scope

> **// TODO:** Update [operator scope doc][operator_scope] and update commands that rely on WATCH_NAMESPACE being defined and sourced from `deploy/operator.yaml`.

Read the [operator scope][operator_scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.

By default the main program will set the manager's namespace using the value of `WATCH_NAMESPACE` env defined in `deploy/operator.yaml`.

## Create a new API and Controller

Create a new Custom Resource Definition(CRD) API with group `cache` version `v1alpha1` and Kind Memcached.
When prompted, enter yes `y` for creating both the resource and controller.

```console
$ operator-sdk create api --group=cache --version=v1alpha1 --kind=Memcached
Create Resource [y/n]
y
Create Controller [y/n]
y
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

After modifying the `*_types.go` file always run the following command to update the generated code for that resource type:

```sh
$ make generate
```

The above makefile target will invoke the [controller-gen][controller_tools] utility to update the `api/v1alpha1/zz_generated.deepcopy.go` file to ensure our API's Go type definitons implement the `runtime.Object` interface that all Kind types must implement.

### Generating CRD manifests

Once the API is defined with spec/status fields and CRD validation markers, the CRD manifests can be generated and updated with the following command:

```console
$ make manifests
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

The next two subsections explain how the controller watches resources and how the reconcile loop is triggered. Skip to the [Build](#build-and-run-the-operator) section to see how to build and run the operator.

#### Resources watched by the Controller

The `SetupWithManager()` function in `controllers/memcached_controller.go` specifies how the controller is built to watch a CR and other resources that are owned and managed by that controller.

```Go
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
            WithOptions(controller.Options{
                MaxConcurrentReconciles: 2,
            }).
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

	cachev1alpha1 "github.com/example-inc/memcached-operator/api/v1alpha1"
	...
)

func (r *MemcachedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
// Requeue for any reason other than error
return ctrl.Result{Requeue: true}, nil
```

You can set the `Result.RequeueAfter` to requeue the `Request` after a grace period as well:
```Go
import "time"

// Reconcile for any reason than error after 5 seconds
return ctrl.Result{RequeueAfter: time.Second*5}, nil
```

**Note:** Returning `Result` with `RequeueAfter` set is how you can periodically reconcile a CR.

For a guide on Reconcilers, Clients, and interacting with resource Events, see the [Client API doc][doc_client_api].

> **// TODO:** Should we point to the client API doc after the integration?

### Specify permissions and generate RBAC manifests

The controller needs certain RBAC permissions to interact with the resources it manages. These are specified via [RBAC markers][rbac_markers] like the following:

```Go
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

func (r *MemcachedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
```

The `ClusterRole` manifest at `config/rbac/role.yaml` is generated from the above markers via controller-gen with the following command:

```sh
$ make manifests
```

## Build and run the operator

Before running the operator, the CRD must be registered with the Kubernetes apiserver:

```sh
$ make install
```

Once this is done, there are two ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster

### 1. Run locally outside the cluster

To run the operator locally execute the following command:

```sh
$ make run ENABLE_WEBHOOKS=false
```

> **// TODO:** Add an advanced section or separate doc on writing defaulting/validating and conversion webhooks.

### 2. Run as a Deployment inside the cluster

#### Build and push the image

Build the operator image and push it to a registry.

Make sure to modify `quay.io/example/` in the example below to reference a container repository that
you have access to. You can obtain an account for storing containers at
repository sites such quay.io or hub.docker.com

Build the image:

```sh
$ make docker-build IMG=quay.io/example/memcached-operator:v0.0.1
```

Push the image to a repository:

```sh
$ make docker-push IMG=quay.io/example/memcached-operator:v0.0.1
```

> **// TODO:** Mention how to use `buildah build` by just updating the Makefile?

**Note**:
The name and tag of the image (`IMG=<some-registry>/<project-name>:tag`) in both the commands can also be set in the Makefile. Modify the line which has `IMG ?= controller:latest` to set your desired default base image.

#### Deploy the operator

For this example we will run the operator in the `default` namespace which can be specified for all resources in `config/default/kustomization.yaml`:

```YAML
# Adds namespace to all resources.
namespace: default
...
```

Run the following to deploy the operator. This will also install the RBAC manifests from `config/rbac`.

```sh
$ make deploy IMG=quay.io/example/memcached-operator:v0.0.1
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           8m
```

### 3. Deploy your Operator with the Operator Lifecycle Manager (OLM)

OLM will manage creation of most if not all resources required to run your operator, using a bit of setup from other `operator-sdk` commands. Check out the [docs][cli-run-olm] for more information.

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
$ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

Ensure that the memcached operator creates the deployment for the sample CR with the correct size:

```sh
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           8m
memcached-sample                        3/3     3            3           1m
```

Check the pods and CR status to confirm the status is updated with the memcached pod names:

```sh
$ kubectl get pods
NAME                                  READY     STATUS    RESTARTS   AGE
memcached-sample-6fd7c98d8-7dqdr      1/1       Running   0          1m
memcached-sample-6fd7c98d8-g5k7v      1/1       Running   0          1m
memcached-sample-6fd7c98d8-m7vn7      1/1       Running   0          1m
```

```sh
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

Update `config/samples/cache_v1alpha1_memcached.yaml` to change the `spec.size` field in the Memcached CR from 3 to 4:

```YAML
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: memcached-sample
spec:
  size: 3
```

Apply the change:

```
$ kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
memcached-operator-controller-manager   1/1     1            1           10m
memcached-sample                        4/4     4            4           3m
```

### Cleanup

```sh
$ kubectl delete -f config/samples/cache_v1alpha1_memcached.yaml
$ $ kubectl delete deployments,service -l control-plane=controller-manager
$ kubectl delete role,rolebinding --all
```

## Advanced Topics

### Manage CR status conditions

An often-used pattern is to include `Conditions` in the status of custom resources. Conditions represent the latest available observations of an object's state (see the [Kubernetes API conventionsdocumentation][typical-status-properties] for more information).

The `Conditions` field added to the `MemcachedStatus` struct simplifies the management of your CR's conditions. It:
- Enables callers to add and remove conditions.
- Ensures that there are no duplicates.
- Sorts the conditions deterministically to avoid unnecessary repeated reconciliations.
- Automatically handles the each condition's `LastTransitionTime`.
- Provides helper methods to make it easy to determine the state of a condition.

To use conditions in your custom resource, add a Conditions field to the Status struct in `_types.go`:

```Go
import (
    "github.com/operator-framework/operator-sdk/pkg/status"
)

type MyAppStatus struct {
    // Conditions represent the latest available observations of an object's state
    Conditions status.Conditions `json:"conditions"`
}
```

Then, in your controller, you can use [`Conditions`][godoc-conditions] methods to make it easier to set and remove conditions or check their current values.

### Adding 3rd Party Resources To Your Operator

> **// TODO:** Update the `main.go` code in these sections to use the `init()` func to register the scheme instead of doing it in `main()`.

The operator's Manager supports the core Kubernetes resource types as found in the client-go [scheme][scheme_package] package and will also register the schemes of all custom resource types defined in your project.

```Go
import (
  "github.com/example-inc/memcached-operator/pkg/apis"
  ...
)

// Setup Scheme for all resources
if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
  log.Error(err, "")
  os.Exit(1)
}
```

To add a 3rd party resource to an operator, you must add it to the Manager's scheme. By creating an `AddToScheme()` method or reusing one you can easily add a resource to your scheme. An [example][deployments_register] shows that you define a function and then use the [runtime][runtime_package] package to create a `SchemeBuilder`.

#### Register with the Manager's scheme

Call the `AddToScheme()` function for your 3rd party resource and pass it the Manager's scheme via `mgr.GetScheme()`
in `cmd/manager/main.go`.

Example:
```go
import (
  ....

  routev1 "github.com/openshift/api/route/v1"
)

func main() {
  ...

  // Adding the routev1
  if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
    log.Error(err, "")
    os.Exit(1)
  }

  ...
}
```

##### If 3rd party resource does not have `AddToScheme()` function

Use the [SchemeBuilder][scheme_builder] package from controller-runtime to initialize a new scheme builder that can be used to register the 3rd party resource with the manager's scheme.

Example of registering `DNSEndpoints` 3rd party resource from `external-dns`:

```go
import (
  ...
    "k8s.io/apimachinery/pkg/runtime/schema"
    "sigs.k8s.io/controller-runtime/pkg/scheme"
  ...
   // DNSEndoints
   externaldns "github.com/kubernetes-incubator/external-dns/endpoint"
   metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
 )

func main() {
  ...

  log.Info("Registering Components.")

  schemeBuilder := &scheme.Builder{GroupVersion: schema.GroupVersion{Group: "externaldns.k8s.io", Version: "v1alpha1"}}
  schemeBuilder.Register(&externaldns.DNSEndpoint{}, &externaldns.DNSEndpointList{})
  if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
    log.Error(err, "")
    os.Exit(1)
  }

  ...
}
```



**NOTES:**

* After adding new import paths to your operator project, run `go mod vendor` if a `vendor/` directory is present in the root of your project directory to fulfill these dependencies.
* Your 3rd party resource needs to be added before add the controller in `"Setup all Controllers"`.

### Metrics

> **// TODO:** Update the [metrics doc](https://github.com/operator-framework/operator-sdk/blob/master/website/content/en/docs/golang/metrics/operator-sdk-monitoring.md) since it doesn't match the default scaffolded by kubebuilder anymore.

To learn about how metrics work in the Operator SDK read the [metrics section][metrics_doc] of the user documentation.

#### Default Metrics exported with 3rd party resource

> **// TODO:** Remove this section since we're no longer scaffolding main.go to use the SDK's `GenerateAndServeCRMetrics()` util in `pkg/kube-metrics`.

### Handle Cleanup on Deletion

> **// TODO:** Update finalizer reconcile code for kubebuilder's default reconciler imports and variable names

To implement complex deletion logic, you can add a finalizer to your Custom Resource. This will prevent your Custom Resource from being
deleted until you remove the finalizer (ie, after your cleanup logic has successfully run). For more information, see the
[official Kubernetes documentation on finalizers](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers).

**Example:**

The following is a snippet from the controller file under `pkg/controller/memcached/memcached_controller.go`

```Go

const memcachedFinalizer = "finalizer.cache.example.com"

func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Memcached")

	// Fetch the Memcached instance
	memcached := &cachev1alpha1.Memcached{}
	err := r.client.Get(context.TODO(), request.NamespacedName, memcached)
	if err != nil {
		// If the resource is not found, that means all of
		// the finalizers have been removed, and the memcached
		// resource has been deleted, so there is nothing left
		// to do.
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("could not fetch memcached instance: %s", err)
	}

	...

	// Check if the Memcached instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMemcachedMarkedToBeDeleted := memcached.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if contains(memcached.GetFinalizers(), memcachedFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeMemcached(reqLogger, memcached); err != nil {
				return reconcile.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			memcached.SetFinalizers(remove(memcached.GetFinalizers(), memcachedFinalizer))
			err := r.client.Update(context.TODO(), memcached)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(memcached.GetFinalizers(), memcachedFinalizer) {
		if err := r.addFinalizer(reqLogger, memcached); err != nil {
			return reconcile.Result{}, err
		}
	}

	...

	return reconcile.Result{}, nil
}

func (r *ReconcileMemcached) finalizeMemcached(reqLogger logr.Logger, m *cachev1alpha1.Memcached) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	reqLogger.Info("Successfully finalized memcached")
	return nil
}

func (r *ReconcileMemcached) addFinalizer(reqLogger logr.Logger, m *cachev1alpha1.Memcached) error {
	reqLogger.Info("Adding Finalizer for the Memcached")
	m.SetFinalizers(append(m.GetFinalizers(), memcachedFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Memcached with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
```

### Leader election

> **// TODO:** Update this section to remove `leader-for-life` option? Since it's no longer the default.

During the lifecycle of an operator it's possible that there may be more than 1 instance running at any given time e.g when rolling out an upgrade for the operator.
In such a scenario it is necessary to avoid contention between multiple operator instances via leader election so that only one leader instance handles the reconciliation while the other instances are inactive but ready to take over when the leader steps down.

There are two different leader election implementations to choose from, each with its own tradeoff.

- [Leader-with-lease][leader_with_lease]: The leader pod periodically renews the leader lease and gives up leadership when it can't renew the lease. This implementation allows for a faster transition to a new leader when the existing leader is isolated, but there is a possibility of split brain in [certain situations][lease_split_brain].
- [Leader-for-life][leader_for_life]: The leader pod only gives up leadership (via garbage collection) when it is deleted. This implementation precludes the possibility of 2 instances mistakenly running as leaders (split brain). However, this method can be subject to a delay in electing a new leader. For instance when the leader pod is on an unresponsive or partitioned node, the [`pod-eviction-timeout`][pod_eviction_timeout] dictates how long it takes for the leader pod to be deleted from the node and step down (default 5m).

By default the SDK enables the leader-with-lease implementation. However you should consult the docs above for both approaches to consider the tradeoffs that make sense for your use case.

The following examples illustrate how to use the two options:

#### Leader for life

A call to `leader.Become()` will block the operator as it retries until it can become the leader by creating the configmap named `memcached-operator-lock`.

```Go
import (
  ...
  "github.com/operator-framework/operator-sdk/pkg/leader"
)

func main() {
  ...
  err = leader.Become(context.TODO(), "memcached-operator-lock")
  if err != nil {
    log.Error(err, "Failed to retry for leader lock")
    os.Exit(1)
  }
  ...
}
```
If the operator is not running inside a cluster `leader.Become()` will simply return without error to skip the leader election since it can't detect the operator's namespace.

#### Leader with lease

The leader-with-lease approach can be enabled via the [Manager Options][manager_options] for leader election.

```Go
import (
  ...
  "sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
  ...
  opts := manager.Options{
    ...
    LeaderElection: true,
    LeaderElectionID: "memcached-operator-lock"
  }
  mgr, err := manager.New(cfg, opts)
  ...
}
```

When the operator is not running in a cluster, the Manager will return an error on starting since it can't detect the operator's namespace in order to create the configmap for leader election. You can override this namespace by setting the Manager's `LeaderElectionNamespace` option.

[enqueue_requests_from_map_func]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/handler#EnqueueRequestsFromMapFunc
[event_handler_godocs]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/handler#hdr-EventHandlers
[event_filtering]:/docs/golang/references/event-filtering/
[controller_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/controller#Options
[controller_godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/controller
[operator_scope]:/docs/operator-scope/
[pod_eviction_timeout]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/#options
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
[lease_split_brain]: https://github.com/kubernetes/client-go/blob/30b06a83d67458700a5378239df6b96948cb9160/tools/leaderelection/leaderelection.go#L21-L24
[leader_for_life]: https://godoc.org/github.com/operator-framework/operator-sdk/pkg/leader
[leader_with_lease]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/leaderelection
[memcached_handler]: ../example/memcached-operator/handler.go.tmpl

[kubebuilder_layout_doc]:https://book.kubebuilder.io/cronjob-tutorial/basic-project.html
[homebrew_tool]:https://brew.sh/
[go_mod_wiki]: https://github.com/golang/go/wiki/Modules
[go_vendoring]: https://blog.gopheracademy.com/advent-2015/vendor-folder/
[scheme_package]:https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
[deployments_register]: https://github.com/kubernetes/api/blob/master/apps/v1/register.go#L41
[doc_client_api]:/docs/golang/references/client/
[runtime_package]: https://godoc.org/k8s.io/apimachinery/pkg/runtime
[manager_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Manager
[controller-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg#hdr-Controller
[request-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Request
[result_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Result
[metrics_doc]: /docs/golang/metrics/operator-sdk-monitoring/
[multi-namespaced-cache-builder]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
[scheme_builder]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/scheme#Builder
[typical-status-properties]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[godoc-conditions]: https://godoc.org/github.com/operator-framework/operator-sdk/pkg/status#Conditions
[cli-run-olm]: /docs/golang/olm-integration/olm-cli/
[kubebuilder_entrypoint_doc]: https://book.kubebuilder.io/cronjob-tutorial/empty-main.html

[api_terms_doc]: https://book.kubebuilder.io/cronjob-tutorial/gvks.html
[kb_controller_doc]: https://book.kubebuilder.io/cronjob-tutorial/controller-overview.html
[kb_api_doc]: https://book.kubebuilder.io/cronjob-tutorial/new-api.html
[controller_tools]: https://sigs.k8s.io/controller-tools
[doc-validation-schema]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#specifying-a-structural-schema
[generating-crd]: https://book.kubebuilder.io/reference/generating-crd.html
[markers]: https://book.kubebuilder.io/reference/markers.html
[crd-markers]: https://book.kubebuilder.io/reference/markers/crd-validation.html
[rbac-markers]: https://book.kubebuilder.io/reference/markers/rbac.html
[memcached_controller]: https://github.com/operator-framework/operator-sdk/blob/master/example/kb-memcached-operator/memcached_controller.go.tmpl
[builder_godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/builder#example-Builder