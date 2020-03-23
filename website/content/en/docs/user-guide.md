# Operator SDK User Guide

This guide walks through an example of building a simple memcached-operator using the operator-sdk CLI tool and controller-runtime library API. To learn how to use Ansible or Helm to create an operator, see the [Ansible Operator User Guide][ansible_user_guide] or the [Helm Operator User Guide][helm_user_guide]. The rest of this document will show how to program an operator in Go.


## Prerequisites

- [git][git_tool]
- [go][go_tool] version v1.12+.
- [mercurial][mercurial_tool] version 3.9+
- [docker][docker_tool] version 17.03+.
- [kubectl][kubectl_tool] version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

**Note**: This guide uses [minikube][minikube_tool] version v0.25.0+ as the local Kubernetes cluster and [quay.io][quay_link] for the public registry.

## Install the Operator SDK CLI

Follow the steps in the [installation guide][install_guide] to learn how to install the Operator SDK CLI tool.

## Create a new project

Use the CLI to create a new memcached-operator project:

```sh
$ mkdir -p $HOME/projects
$ cd $HOME/projects
$ operator-sdk new memcached-operator --repo=github.com/example-inc/memcached-operator
$ cd memcached-operator
```

To learn about the project directory structure, see [project layout][layout_doc] doc.

#### A note on dependency management

`operator-sdk new` generates a `go.mod` file to be used with [Go modules][go_mod_wiki]. The `--repo=<path>` flag is required when creating a project outside of `$GOPATH/src`, as scaffolded files require a valid module path. Ensure you activate module support before using the SDK. From the [Go modules Wiki][go_mod_wiki]:

> You can activate module support in one of two ways:
> - Invoke the go command in a directory with a valid go.mod file in the current directory or any parent of it and the environment variable GO111MODULE unset (or explicitly set to auto).
> - Invoke the go command with GO111MODULE=on environment variable set.

##### Vendoring

By default `--vendor=false`, so an operator's dependencies are downloaded and cached in the Go modules cache. Calls to `go {build,clean,get,install,list,run,test}` by `operator-sdk` subcommands will use an external modules directory. Execute `go help modules` for more information.

The Operator SDK can create a [`vendor`][go_vendoring] directory for Go dependencies if the project is initialized with `--vendor=true`.

#### Operator scope

Read the [operator scope][operator_scope] documentation on how to run your operator as namespace-scoped vs cluster-scoped.

### Manager
The main program for the operator `cmd/manager/main.go` initializes and runs the [Manager][manager_go_doc].

The Manager will automatically register the scheme for all custom resources defined under `pkg/apis/...` and run all controllers under `pkg/controller/...`.

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
   MapperProvider:     restmapper.NewDynamicRESTMapper,
   MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
})
```

By default the main program will set the manager's namespace using the value of `WATCH_NAMESPACE` env defined in `deploy/operator.yaml`.

## Add a new Custom Resource Definition

Add a new Custom Resource Definition(CRD) API called Memcached, with APIVersion `cache.example.com/v1alpha1` and Kind `Memcached`.

```sh
$ operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=Memcached
```

This will scaffold the Memcached resource API under `pkg/apis/cache/v1alpha1/...`.

### Define the spec and status

Modify the spec and status of the `Memcached` Custom Resource(CR) at `pkg/apis/cache/v1alpha1/memcached_types.go`:

```Go
type MemcachedSpec struct {
	// Size is the size of the memcached deployment
	Size int32 `json:"size"`
}
type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	Nodes []string `json:"nodes"`
}
```

After modifying the `*_types.go` file always run the following command to update the generated code for that resource type:

```sh
$ operator-sdk generate k8s
```

### Updating CRD manifests

Now that `MemcachedSpec` and `MemcachedStatus` have fields and possibly annotations, the CRD corresponding to the API's group and kind must be updated. To do so, run the following command:

```console
$ operator-sdk generate crds
```

**Notes:**
- - Your CRD *must* specify exactly one [storage version][crd-storage-version]. Use the `+kubebuilder:storageversion` [marker][crd-markers] to indicate the GVK that should be used to store data by the API server. This marker should be in a comment above your `Memcached` type.

[crd-storage-version]:https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#writing-reading-and-updating-versioned-customresourcedefinition-objects
[crd-markers]:https://book.kubebuilder.io/reference/markers/crd.html
[api-rules]: https://github.com/kubernetes/kubernetes/tree/36981002246682ed7dc4de54ccc2a96c1a0cbbdb/api/api-rules

#### OpenAPI validation

OpenAPIv3 schemas are added to CRD manifests in the `spec.validation` block when the manifests are generated. This validation block allows Kubernetes to validate the properties in a Memcached Custom Resource when it is created or updated.

Markers (annotations) are available to configure validations for your API. These markers will always have a `+kubebuilder:validation` prefix. For example, adding an enum type specification can be done by adding the following marker:

```go
// +kubebuilder:validation:Enum=Lion;Wolf;Dragon
type Alias string
```

Usage of markers in API code is discussed in the kubebuilder [CRD generation][generating-crd] and [marker][markers] documentation. A full list of OpenAPIv3 validation markers can be found [here][crd-markers].

To update the CRD `deploy/crds/cache.example.com_memcacheds_crd.yaml`, run the following command:

```console
$ operator-sdk generate crds
```

An example of the generated YAML is as follows:

```YAML
spec:
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            size:
              format: int32
              type: integer
```

To learn more about OpenAPI v3.0 validation schemas in Custom Resource Definitions, refer to the [Kubernetes Documentation][doc-validation-schema].

[doc-validation-schema]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#specifying-a-structural-schema
[generating-crd]: https://book.kubebuilder.io/reference/generating-crd.html
[markers]: https://book.kubebuilder.io/reference/markers.html
[crd-markers]: https://book.kubebuilder.io/reference/markers/crd-validation.html

## Add a new Controller

Add a new [Controller][controller-go-doc] to the project that will watch and reconcile the Memcached resource:

```sh
$ operator-sdk add controller --api-version=cache.example.com/v1alpha1 --kind=Memcached
```

This will scaffold a new Controller implementation under `pkg/controller/memcached/...`.

For this example replace the generated Controller file `pkg/controller/memcached/memcached_controller.go` with the example [`memcached_controller.go`][memcached_controller] implementation.

The example Controller executes the following reconciliation logic for each `Memcached` CR:
- Create a memcached Deployment if it doesn't exist
- Ensure that the Deployment size is the same as specified by the `Memcached` CR spec
- Update the `Memcached` CR status using the status writer with the names of the memcached pods

The next two subsections explain how the Controller watches resources and how the reconcile loop is triggered. Skip to the [Build](#build-and-run-the-operator) section to see how to build and run the operator.

### Resources watched by the Controller

Inspect the Controller implementation at `pkg/controller/memcached/memcached_controller.go` to see how the Controller watches resources.

The first watch is for the Memcached type as the primary resource. For each Add/Update/Delete event the reconcile loop will be sent a reconcile `Request` (a namespace/name key) for that Memcached object:

```Go
err := c.Watch(
  &source.Kind{Type: &cachev1alpha1.Memcached{}}, &handler.EnqueueRequestForObject{})
```

The next watch is for Deployments but the event handler will map each event to a reconcile `Request` for the owner of the Deployment. Which in this case is the Memcached object for which the Deployment was created. This allows the controller to watch Deployments as a secondary resource.

```Go
err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
    IsController: true,
    OwnerType:    &cachev1alpha1.Memcached{},
  })
```

#### Controller configurations

There are a number of useful configurations that can be made when initialzing a controller and declaring the watch parameters. For more details on these configurations consult the upstream [controller godocs][controller_godocs].

- Set the max number of concurrent Reconciles for the controller via the [`MaxConcurrentReconciles`][controller_options]  option. Defaults to 1.
  ```Go
  _, err := controller.New("memcached-controller", mgr, controller.Options{
	  MaxConcurrentReconciles: 2,
	  ...
  })
  ```
- Filter watch events using [predicates][event_filtering]
- Choose the type of [EventHandler][event_handler_godocs] to change how a watch event will translate to reconcile requests for the reconcile loop. For operator relationships that are more complex than primary and secondary resources, the [`EnqueueRequestsFromMapFunc`][enqueue_requests_from_map_func] handler can be used to transform a watch event into an arbitrary set of reconcile requests.


### Reconcile loop

Every Controller has a Reconciler object with a `Reconcile()` method that implements the reconcile loop. The reconcile loop is passed the [`Request`][request-go-doc] argument which is a Namespace/Name key used to lookup the primary resource object, Memcached, from the cache:

```Go
func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error) {
  // Lookup the Memcached instance for this reconcile request
  memcached := &cachev1alpha1.Memcached{}
  err := r.client.Get(context.TODO(), request.NamespacedName, memcached)
  ...
}
```

Based on the return values, [`Result`][result_go_doc] and error, the `Request` may be requeued and the reconcile loop may be triggered again:

```Go
// Reconcile successful - don't requeue
return reconcile.Result{}, nil
// Reconcile failed due to error - requeue
return reconcile.Result{}, err
// Requeue for any reason other than error
return reconcile.Result{Requeue: true}, nil
```

You can set the `Result.RequeueAfter` to requeue the `Request` after a grace period as well:
```Go
import "time"

// Reconcile for any reason than error after 5 seconds
return reconcile.Result{RequeueAfter: time.Second*5}, nil
```

**Note:** Returning `Result` with `RequeueAfter` set is how you can periodically reconcile a CR.

For a guide on Reconcilers, Clients, and interacting with resource Events, see the [Client API doc][doc_client_api].

## Build and run the operator

Before running the operator, the CRD must be registered with the Kubernetes apiserver:

```sh
$ kubectl create -f deploy/crds/cache.example.com_memcacheds_crd.yaml
```

Once this is done, there are two ways to run the operator:

- As a Deployment inside a Kubernetes cluster
- As Go program outside a cluster

### 1. Run as a Deployment inside the cluster

**Note**: `operator-sdk build` invokes `docker build` by default, and optionally `buildah bud`. If using `buildah`, skip to the `operator-sdk build` invocation instructions below. If using `docker`, make sure your docker daemon is running and that you can run the docker client without sudo. You can check if this is the case by running `docker version`, which should complete without errors. Follow instructions for your OS/distribution on how to start the docker daemon and configure your access permissions, if needed.

**Note**: If a `vendor/` directory is present, run

```sh
$ go mod vendor
```

before building the memcached-operator image.

Build the memcached-operator image and push it to a registry.  Make sure to modify
`quay.io/example/` in the example below to reference a container repository that
you have access to. You can obtain an account for storing containers at
repository sites such quay.io or hub.docker.com: 
```sh
$ operator-sdk build quay.io/example/memcached-operator:v0.0.1
$ sed -i 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
$ docker push quay.io/example/memcached-operator:v0.0.1
```

**Note**
If you are performing these steps on OSX, use the following `sed` command instead:
```sh
$ sed -i "" 's|REPLACE_IMAGE|quay.io/example/memcached-operator:v0.0.1|g' deploy/operator.yaml
```

The Deployment manifest is generated at `deploy/operator.yaml`. Be sure to update the deployment image as shown above since the default is just a placeholder.

Setup RBAC and deploy the memcached-operator:

```sh
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
$ kubectl create -f deploy/operator.yaml
```

Verify that the memcached-operator is up and running:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           1m
```

### 2. Run locally outside the cluster

This method is preferred during development cycle to deploy and test faster.

Set the name of the operator in an environment variable:

```sh
export OPERATOR_NAME=memcached-operator
```

Run the operator locally with the default Kubernetes config file present at `$HOME/.kube/config`. And watch the namespace `default`:

```sh
$ operator-sdk run --local --watch-namespace=default
2018/09/30 23:10:11 Go Version: go1.10.2
2018/09/30 23:10:11 Go OS/Arch: darwin/amd64
2018/09/30 23:10:11 operator-sdk Version: 0.0.6+git
2018/09/30 23:10:12 Registering Components.
2018/09/30 23:10:12 Starting the Cmd.
```

You can use a specific kubeconfig via the flag `--kubeconfig=<path/to/kubeconfig>`.

## Create a Memcached CR

Create the example `Memcached` CR that was generated at `deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml`:

```sh
$ cat deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 3

$ kubectl apply -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
```

Ensure that the memcached-operator creates the deployment for the CR:

```sh
$ kubectl get deployment
NAME                     DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
memcached-operator       1         1         1            1           2m
example-memcached        3         3         3            3           1m
```

Check the pods and CR status to confirm the status is updated with the memcached pod names:

```sh
$ kubectl get pods
NAME                                  READY     STATUS    RESTARTS   AGE
example-memcached-6fd7c98d8-7dqdr     1/1       Running   0          1m
example-memcached-6fd7c98d8-g5k7v     1/1       Running   0          1m
example-memcached-6fd7c98d8-m7vn7     1/1       Running   0          1m
memcached-operator-7cc7cfdf86-vvjqk   1/1       Running   0          2m
```

```sh
$ kubectl get memcached/example-memcached -o yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  clusterName: ""
  creationTimestamp: 2018-03-31T22:51:08Z
  generation: 0
  name: example-memcached
  namespace: default
  resourceVersion: "245453"
  selfLink: /apis/cache.example.com/v1alpha1/namespaces/default/memcacheds/example-memcached
  uid: 0026cc97-3536-11e8-bd83-0800274106a1
spec:
  size: 3
status:
  nodes:
  - example-memcached-6fd7c98d8-7dqdr
  - example-memcached-6fd7c98d8-g5k7v
  - example-memcached-6fd7c98d8-m7vn7
```

### Update the size

Change the `spec.size` field in the memcached CR from 3 to 4 and apply the change:

```sh
$ cat deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
apiVersion: "cache.example.com/v1alpha1"
kind: "Memcached"
metadata:
  name: "example-memcached"
spec:
  size: 4

$ kubectl apply -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
```

Confirm that the operator changes the deployment size:

```sh
$ kubectl get deployment
NAME                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-memcached    4         4         4            4           5m
```

### Cleanup

Clean up the resources:

```sh
$ kubectl delete -f deploy/crds/cache.example.com_v1alpha1_memcached_cr.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/role_binding.yaml
$ kubectl delete -f deploy/role.yaml
$ kubectl delete -f deploy/service_account.yaml
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

The operator's Manager supports the Core Kubernetes resource types as found in the client-go [scheme][scheme_package] package and will also register the schemes of all custom resource types defined in your project under `pkg/apis`.

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
  ....

  // Adding the routev1
  if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
    log.Error(err, "")
    os.Exit(1)
  }

  ....

  // Setup all Controllers
  if err := controller.AddToManager(mgr); err != nil {
    log.Error(err, "")
    os.Exit(1)
  }
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
  ....

  log.Info("Registering Components.")

  schemeBuilder := &scheme.Builder{GroupVersion: schema.GroupVersion{Group: "externaldns.k8s.io", Version: "v1alpha1"}}
  schemeBuilder.Register(&externaldns.DNSEndpoint{}, &externaldns.DNSEndpointList{})
  if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
    log.Error(err, "")
    os.Exit(1)
  }

  ....

  // Setup all Controllers
  if err := controller.AddToManager(mgr); err != nil {
    log.Error(err, "")
    os.Exit(1)
  }
}
```



**NOTES:**

* After adding new import paths to your operator project, run `go mod vendor` if a `vendor/` directory is present in the root of your project directory to fulfill these dependencies.
* Your 3rd party resource needs to be added before add the controller in `"Setup all Controllers"`.

#### Default Metrics exported with 3rd party resource 

By default, SDK operator projects are set up to [export metrics][metrics_doc] through `addMetrics` in `cmd/manager/main.go`. See that it will call the `serveCRMetrics`:


```go
func serveCRMetrics(cfg *rest.Config) error {
    ...
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

    ...

	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
``` 

The `kubemetrics.GenerateAndServeCRMetrics` function requires an RBAC rule to list all GroupVersionKinds in the list of watched namespaces, so you might need to [filter](https://github.com/operator-framework/operator-sdk/blob/v0.15.2/pkg/k8sutil/k8sutil.go#L161) the kinds returned by [`k8sutil.GetGVKsFromAddToScheme`](https://godoc.org/github.com/operator-framework/operator-sdk/pkg/k8sutil#GetGVKsFromAddToScheme) more stringently to avoid authorization errors such as `Failed to list *unstructured.Unstructured`.

In this scenario, this error may occur because your Operator RBAC roles do not include permissions to LIST the third party API schemas or the schemas which are required to them and will be added with. See that the default SDK implementation will just add the Kubernetes schemas and they will be ignored in the metrics It means that you might need to do an similar implementation to filter the third party API schemas and their dependencies added in order to provide a filtered a List of GVK(GroupVersionKind) to the  `GenerateAndServeCRMetrics` method. 

### Handle Cleanup on Deletion

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

### Metrics

To learn about how metrics work in the Operator SDK read the [metrics section][metrics_doc] of the user documentation.

## Leader election

During the lifecycle of an operator it's possible that there may be more than 1 instance running at any given time e.g when rolling out an upgrade for the operator.
In such a scenario it is necessary to avoid contention between multiple operator instances via leader election so that only one leader instance handles the reconciliation while the other instances are inactive but ready to take over when the leader steps down.

There are two different leader election implementations to choose from, each with its own tradeoff.

- [Leader-for-life][leader_for_life]: The leader pod only gives up leadership (via garbage collection) when it is deleted. This implementation precludes the possibility of 2 instances mistakenly running as leaders (split brain). However, this method can be subject to a delay in electing a new leader. For instance when the leader pod is on an unresponsive or partitioned node, the [`pod-eviction-timeout`][pod_eviction_timeout] dictates how long it takes for the leader pod to be deleted from the node and step down (default 5m).
- [Leader-with-lease][leader_with_lease]: The leader pod periodically renews the leader lease and gives up leadership when it can't renew the lease. This implementation allows for a faster transition to a new leader when the existing leader is isolated, but there is a possibility of split brain in [certain situations][lease_split_brain].

By default the SDK enables the leader-for-life implementation. However you should consult the docs above for both approaches to consider the tradeoffs that make sense for your use case.

The following examples illustrate how to use the two options:

### Leader for life

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

### Leader with lease

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
[event_filtering]:./user/event-filtering.md
[controller_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/controller#Options
[controller_godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/controller
[operator_scope]:./operator-scope.md
[install_guide]: ./user/install-operator-sdk.md
[pod_eviction_timeout]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/#options
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
[lease_split_brain]: https://github.com/kubernetes/client-go/blob/30b06a83d67458700a5378239df6b96948cb9160/tools/leaderelection/leaderelection.go#L21-L24
[leader_for_life]: https://godoc.org/github.com/operator-framework/operator-sdk/pkg/leader
[leader_with_lease]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/leaderelection
[memcached_handler]: ../example/memcached-operator/handler.go.tmpl
[memcached_controller]: ../example/memcached-operator/memcached_controller.go.tmpl
[layout_doc]:./project_layout.md
[ansible_user_guide]:./ansible/user-guide.md
[helm_user_guide]:./helm/user-guide.md
[homebrew_tool]:https://brew.sh/
[go_mod_wiki]: https://github.com/golang/go/wiki/Modules
[go_vendoring]: https://blog.gopheracademy.com/advent-2015/vendor-folder/
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[mercurial_tool]:https://www.mercurial-scm.org/downloads
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube_tool]:https://github.com/kubernetes/minikube#installation
[scheme_package]:https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
[deployments_register]: https://github.com/kubernetes/api/blob/master/apps/v1/register.go#L41
[doc_client_api]:./user/client.md
[runtime_package]: https://godoc.org/k8s.io/apimachinery/pkg/runtime
[manager_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Manager
[controller-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg#hdr-Controller
[request-go-doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Request
[result_go_doc]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Result
[metrics_doc]: ./user/metrics/README.md
[quay_link]: https://quay.io
[multi-namespaced-cache-builder]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
[scheme_builder]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/scheme#Builder
[typical-status-properties]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[godoc-conditions]: https://godoc.org/github.com/operator-framework/operator-sdk/pkg/status#Conditions
