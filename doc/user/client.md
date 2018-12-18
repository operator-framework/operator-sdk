# Operator-SDK: Controller Runtime Client API

## Overview

The [`controller-runtime`][repo-controller-runtime] library provides various abstractions to watch and reconcile resources in a Kubernetes cluster via CRUD (Create, Update, Delete, as well as Get and List in this case) operations. Operators use at least one controller to perform a coherent set of tasks within a cluster, usually through a combination of CRUD operations. The Operator SDK uses controller-runtime's [Client][doc-client-client] interface, which provides the interface for these operations.

controller-runtime defines several interfaces used for cluster interaction:
- `client.Client`: implementers perform CRUD operations on a Kubernetes cluster.
- `manager.Manager`: manages shared dependencies, such as Caches and Clients.
- `reconcile.Reconciler`: compares provided state with actual cluster state and updates the cluster on finding state differences using a Client.

Clients are the focus of this document. A separate document will discuss Managers.

## Client Usage

### Default Client

The SDK relies on a `manager.Manager` to create a `client.Client` interface that performs Create, Update, Delete, Get, and List operations within a `reconcile.Reconciler`'s Reconcile function. The SDK will generate code to create a Manager, which holds a Cache and a Client to be used in CRUD operations and communicate with the API server. By default a Controller's Reconciler will be populated with the Manager's Client which is a [split-client][doc-split-client].

`pkg/controller/<kind>/<kind>_controller.go`:
```Go
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKind{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

type ReconcileKind struct {
	// Populated above from a manager.Manager.
	client client.Client
	scheme *runtime.Scheme
}
```

A split client reads (Get and List) from the Cache and writes (Create, Update, Delete) to the API server. Reading from the Cache significantly reduces request load on the API server; as long as the Cache is updated by the API server, read operations are eventually consistent. 

### Non-default Client

An operator developer may wish to create their own Client that serves read requests(Get List) from the API server instead of the cache, for example. controller-runtime provides a [constructor][doc-client-constr] for Clients:

```Go
// New returns a new Client using the provided config and Options.
func New(config *rest.Config, options client.Options) (client.Client, error)
```

`client.Options` allow the caller to specify how the new Client should communicate with the API server.

```Go
// Options are creation options for a Client
type Options struct {
	// Scheme, if provided, will be used to map go structs to GroupVersionKinds
	Scheme *runtime.Scheme

	// Mapper, if provided, will be used to map GroupVersionKinds to Resources
	Mapper meta.RESTMapper
}
```
Example:
```Go
import (
    "sigs.k8s.io/controller-runtime/pkg/client/config"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

cfg, err := config.GetConfig()
...
c, err := client.New(cfg, client.Options{})
...
```

**Note**: defaults are set by `client.New` when Options are empty. The default [scheme][code-scheme-default] will have the [core][doc-k8s-core] Kubernetes resource types registered. The caller *must* set a scheme that has custom operator types registered for the new Client to recognize these types.

Creating a new Client is not usually necessary nor advised, as the default Client is sufficient for most use cases.

### Reconcile and the Client API

A Reconciler implements the [`reconcile.Reconciler`][doc-reconcile-reconciler] interface, which exposes the Reconcile method. Reconcilers are added to a corresponding Controller for a Kind; Reconcile is called in response to cluster or external Events, with a `reconcile.Request` object argument, to read and write cluster state by the Controller, and returns a `reconcile.Result`. SDK Reconcilers have access to a Client in order to make Kubernetes API calls.

**Note**: For those familiar with the SDK's old project semantics, [Handle][doc-osdk-handle] received resource events and reconciled state for multiple resource types, whereas Reconcile receives resource events and reconciles state for a single resource type.

```Go
// ReconcileKind reconciles a Kind object
type ReconcileKind struct {
	// client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client

	// scheme defines methods for serializing and deserializing API objects,
	// a type registry for converting group, version, and kind information
	// to and from Go schemas, and mappings between Go schemas of different
	// versions. A scheme is the foundation for a versioned API and versioned
	// configuration over time. 
	scheme *runtime.Scheme
}

// Reconcile watches for Events and reconciles cluster state with desired
// state defined in the method body.
// The Controller will requeue the Request to be processed again if an error
// is non-nil or Result.Requeue is true, otherwise upon completion it will
// remove the work from the queue.
func (r *ReconcileKind) Reconcile(request reconcile.Request) (reconcile.Result, error)
```

Reconcile is where Controller business logic lives, i.e. where Client API calls are made via `ReconcileKind.client`. A `client.Client` implementer performs the following operations:

#### Get

```Go
// Get retrieves an API object for a given object key from the Kubernetes cluster
// and stores it in obj.
func (c Client) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error
```
**Note**: An `ObjectKey` is simply a `client` package alias for [`types.NamespacedName`][doc-types-nsname].

Example:
```Go
import (
	"context"
	"github.com/example-org/app-operator/pkg/apis/cache/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	app := &v1alpha1.App{}
	ctx := context.TODO()
	err := r.client.Get(ctx, request.NamespacedName, app)

	...
}
```

#### List

```Go
// List retrieves a list of objects for a given namespace and list options
// and stores the list in obj.
func (c Client) List(ctx context.Context, opts *ListOptions, obj runtime.Object) error
```
A `client.ListOptions` sets filters and options for a `List` call:
```Go
type ListOptions struct {
    // LabelSelector filters results by label.  Use SetLabelSelector to
    // set from raw string form.
    LabelSelector labels.Selector

    // FieldSelector filters results by a particular field.  In order
    // to use this with cache-based implementations, restrict usage to
    // a single field-value pair that's been added to the indexers.
    FieldSelector fields.Selector

    // Namespace represents the namespace to list for, or empty for
    // non-namespaced objects, or to list across all namespaces.
    Namespace string

    // Raw represents raw ListOptions, as passed to the API server.  Note
    // that these may not be respected by all implementations of interface,
    // and the LabelSelector and FieldSelector fields are ignored.
    Raw *metav1.ListOptions
}
```
Example:
```Go
import (
	"context"
	"fmt"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	// Return all pods in the request namespace with a label of `app=<name>`
	opts := &client.ListOptions{}
	opts.SetLabelSelector(fmt.Sprintf("app=%s", request.NamespacedName.Name))
	opts.InNamespace(request.NamespacedName.Namespace)

	podList := &v1.PodList{}
	ctx := context.TODO()
	err := r.client.List(ctx, opts, podList)

	...
}
```

#### Create

```Go
// Create saves the object obj in the Kubernetes cluster.
// Returns an error 
func (c Client) Create(ctx context.Context, obj runtime.Object) error
```
Example:
```Go
import (
	"context"
	"k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	app := &v1.Deployment{ // Any cluster object you want to create.
		...
	}
	ctx := context.TODO()
	err := r.client.Create(ctx, app)

	...
}
```

#### Update

```Go
// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned
// by the API server. Update does *not* update the resource's status
// subresource
func (c Client) Update(ctx context.Context, obj runtime.Object) error
```
Example:
```Go
import (
	"context"
	"k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	dep := &v1.Deployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, dep)

	...

	ctx := context.TODO()
	dep.Spec.Selector.MatchLabels["is_running"] = "true"
	err := r.client.Update(ctx, dep)

	...
}
```

##### Updating Status Subresource

When updating the [status subresource][cr-status-subresource] from the client,
the StatusWriter must be used which can be gotten with `Status()`

##### Status

```Go
// Status() returns a StatusWriter object that can be used to update the
// object's status subresource
func (c Client) Status() (client.StatusWriter, error)
```

Example:
```Go
import (
	"context"
	cachev1alpha1 "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	mem := &cachev1alpha.Memcached{}
	err := r.client.Get(context.TODO(), request.NamespacedName, mem)

	...

	ctx := context.TODO()
	mem.Status.Nodes = []string{"pod1", "pod2"}
	err := r.client.Status().Update(ctx, mem)

	...
}
```


#### Delete

```Go
// Delete deletes the given obj from Kubernetes cluster.
func (c Client) Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error
```
A `client.DeleteOptionFunc` sets fields of `client.DeleteOptions` to configure a `Delete` call:
```Go
// DeleteOptionFunc is a function that mutates a DeleteOptions struct.
type DeleteOptionFunc func(*DeleteOptions)

type DeleteOptions struct {
    // GracePeriodSeconds is the duration in seconds before the object should be
    // deleted. Value must be non-negative integer. The value zero indicates
    // delete immediately. If this value is nil, the default grace period for the
    // specified type will be used.
    GracePeriodSeconds *int64

    // Preconditions must be fulfilled before a deletion is carried out. If not
    // possible, a 409 Conflict status will be returned.
    Preconditions *metav1.Preconditions

    // PropagationPolicy determined whether and how garbage collection will be
    // performed. Either this field or OrphanDependents may be set, but not both.
    // The default policy is decided by the existing finalizer set in the
    // metadata.finalizers and the resource-specific default policy.
    // Acceptable values are: 'Orphan' - orphan the dependents; 'Background' -
    // allow the garbage collector to delete the dependents in the background;
    // 'Foreground' - a cascading policy that deletes all dependents in the
    // foreground.
    PropagationPolicy *metav1.DeletionPropagation

    // Raw represents raw DeleteOptions, as passed to the API server.
    Raw *metav1.DeleteOptions
}
```
Example:
```Go
import (
	"context"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	pod := &v1.Pod{}
	err := r.client.Get(context.TODO(), request.NamespacedName, pod)

	...

	ctx := context.TODO()
	if pod.Status.Phase == v1.PodUnknown {
		// Delete the pod after 5 seconds.
		err := r.client.Delete(ctx, pod, client.GracePeriodSeconds(5))
		...
	}

	...
}
```

### Example usage

```Go
import (
	"context"
	"reflect"

	appv1alpha1 "github.com/example-org/app-operator/pkg/apis/app/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcileApp struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Fetch the App instance.
	app := &appv1alpha1.App{}
	err := r.client.Get(context.TODO(), request.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Check if the deployment already exists, if not create a new deployment.
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)
	if err != nil {
	 	if errors.IsNotFound(err) {
			// Define and create a new deployment.
			dep := r.deploymentForApp(app)
			if err = r.client.Create(context.TODO(), dep); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		} else {
			return reconcile.Result{}, err			
		}
	}

	// Ensure the deployment size is the same as the spec.
	size := app.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.client.Update(context.TODO(), found); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the App status with the pod names.
	// List the pods for this app's deployment.
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForApp(app.Name))
	listOps := &client.ListOptions{Namespace: app.Namespace, LabelSelector: labelSelector}
	if err = r.client.List(context.TODO(), listOps, podList); err != nil {
		return reconcile.Result{}, err
	}

	// Update status.Nodes if needed.
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, app.Status.Nodes) {
		app.Status.Nodes = podNames
		if err := r.client.Status().Update(context.TODO(), app); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// deploymentForApp returns a app Deployment object.
func (r *ReconcileKind) deploymentForApp(m *appv1alpha1.App) *appsv1.Deployment {
	lbls := labelsForApp(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: lbls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "app:alpine",
						Name:    "app",
						Command: []string{"app", "-a=64", "-b"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 10000,
							Name:          "app",
						}},
					}},
				},
			},
		},
	}

	// Set App instance as the owner and controller.
	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}

// labelsForApp creates a simple set of labels for App.
func labelsForApp(name string) map[string]string {
	return map[string]string{"app_name": "app", "app_cr": name}
}
```

[repo-controller-runtime]:https://github.com/kubernetes-sigs/controller-runtime
[doc-client-client]:https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/client#Client
[doc-split-client]:https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/client#DelegatingClient
[doc-client-constr]:https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/client#New
[code-scheme-default]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/client.go#L51
[doc-k8s-core]:https://godoc.org/k8s.io/api/core/v1
[doc-reconcile-reconciler]:https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/reconcile#Reconciler
[doc-osdk-handle]:https://github.com/operator-framework/operator-sdk/blob/master/doc/design/milestone-0.0.2/action-api.md#handler
[doc-types-nsname]:https://godoc.org/k8s.io/apimachinery/pkg/types#NamespacedName
[cr-status-subresource]:https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#status-subresource
