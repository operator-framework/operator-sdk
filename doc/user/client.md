# Operator-SDK: Controller Runtime Client API

## Overview

The [`controller-runtime`][repo-controller-runtime] library provides various abstractions to watch and reconcile resources in a k8s cluster via CRUD (Create, Update, Delete, as well as Get and List in this case) operations. Operators use at least one controller to perform a coherent set of tasks within a cluster, usually through a combination of CRUD operations. The Operator SDK uses controller-runtime's [Client][doc-client-client] interface, which defines a Reader and Writer to perform these operations.

controller-runtime defines several interfaces used for cluster interaction:
- `client.Client`: implementers perform CRUD operations on a k8s cluster.
- `manager.Manager`: manages shared dependencies, such as Caches and Clients.
- `reconcile.Reconciler`: compares provided state with actual cluster state and updates the cluster on finding state differences using a Client.

Clients are the focus of this document. A separate document will discuss Managers.

## Client Usage

### Default Client

The SDK relies on a `manager.Manager` to create a `client.Client` interface that performs Create, Update, Delete, Get, and List operations within a `reconcile.Reconciler`'s Reconcile function. The SDK will generate code to create a Manager, which holds a Cache and a Client to be used in CRUD operations and communicate with the API server. By default a Controller's Reconciler will be populated with the Manager's Client which is a [split-client][code-split-client].

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

An operator developer may wish to create their own Client that serves read requests(Get List) from the API server instead of the cache, for example. controller-runtime provides a [constructor][code-client-constr] for Clients:

```Go
// New returns a new Client using the provided config and Options.
func New(config *rest.Config, options client.Options) (client.Client, error)
```

Creating a new Client is not usually necessary nor advised, as the default Client is sufficient for most use cases.

### Reconcile and the Client API

A Reconciler implements the [`reconcile.Reconciler`][code-reconcile-reconciler] interface, which exposes the Reconcile method. Reconcilers are added to a corresponding Controller for a Kind; Reconcile is called in response to cluster or external Events, with a `reconcile.Request` object argument, to read and write cluster state by the Controller, and returns a `reconcile.Result`. SDK Reconcilers have access to a Client in order to make k8s API calls.

**Note**: For those familiar with the SDK's old project structure, Reconcile replaces [Handle][doc-osdk-handle].

```Go
// ReconcileMemcached reconciles a Kind object
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

#### Create

```Go
// Create saves the object obj in the k8s cluster.
// Returns an error 
func (c Client) Create(ctx context.Context, obj runtime.Object) error
```

```Go
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	app := &v1alpha1.Deployment{ // Any cluster object you want to create.
		...
	}
	ctx := context.TODO()
	err := r.client.Create(ctx, app)

	...
}
```

#### Update

```Go
// Update updates the given obj in the k8s cluster. obj must be a
// struct pointer so that obj can be updated with the content returned
// by the API server.
func (c Client) Update(ctx context.Context, obj runtime.Object) error
```

```Go
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

#### Delete

```Go
// Delete deletes the given obj from k8s cluster.
func (c Client) Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error
```

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

```Go
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	pod := &v1.Pod{}
	err := r.client.Get(context.TODO(), request.NamespacedName, pod)

	...

	ctx := context.TODO()
	if pod.Status.Phase == v1.PodUnknown {
		df := func(opts *client.DeleteOptions) {
			s := 5
			opts.GracePeriodSeconds = &s // Delete the pod after 5 seconds.
		}
		err := r.client.Delete(ctx, pod, df)

		...
	}

	...
}
```

#### Get

```Go
// Get retrieves an API object for a given object key from the k8s cluster
// and stores it in obj.
func (c Client) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error
```

```Go
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	app := &v1alpha1.App{}
	ctx := context.TODO()
	name := request.NamespacedName
	err := r.client.Get(ctx, name, app)

	...
}
```

#### List

```Go
// List retrieves a list of objects for a given namespace and list options
// and stores the list in obj.
func (c Client) List(ctx context.Context, opts *ListOptions, obj runtime.Object) error
```

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

```Go
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	...
	
	// Return all pods with a label of request.NamespacedName
	opts := &client.ListOptions{}
	opts.SetLabelSelector(request.NamespacedName)

	pod := &v1.Pod{}
	ctx := context.TODO()
	err := r.client.Get(ctx, opts, pod)

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
			dep := r.deploymentForMemcached(app)
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
		if err := r.client.Update(context.TODO(), app); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// deploymentForApp returns a app Deployment object.
func (r *ReconcileMemcached) deploymentForApp(m *appv1alpha1.App) *appsv1.Deployment {
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
[code-split-client]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/split.go#L26-L28
[code-client-constr]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/client.go#L44
[code-reconcile-reconciler]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L35
[doc-osdk-handle]:https://github.com/operator-framework/operator-sdk/blob/master/doc/design/milestone-0.0.2/action-api.md#handler
