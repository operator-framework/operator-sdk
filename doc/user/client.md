# Operator-SDK: Controller Runtime Client API

## Overview

The [kubernetes-sigs][org-kubernetes-sigs] organization builds tools to call the [Kubernetes][site-kubernetes] API in a clean, abstracted manner, much like the Operator SDK. The [controller-runtime][repo-controller-runtime] project is meant to help users build [controllers][site-kubernetes-controllers] easily by generating a controller project that interacts with a k8s cluster via CRUD operations. User-defined controller can perform specific tasks in a cluster; [operators][site-operators] can use multiple controllers to perform a variety of tasks. As operators use at least one controller, the SDK can rely on controller-runtime's k8s API code rather than develop a parallel set of API calls to execute the same cluster operations, namely CRUD operations.

controller-runtime defines several interfaces used for cluster interaction:
- `client.Client`: implementers perform CRUD operations on a k8s cluster.
- `manager.Manager`: manages shared dependencies, such as Caches and Clients.
- `reconcile.Reconciler`: compares provided state with actual cluster state and updates the cluster on finding state differences using a Client.

The SDK relies on a `manager.Manager` to create a split `client.Client` interface that performs Create, Update, Delete, Get, and List operations within a `reconcile.Reconciler`'s Reconcile function.

## Client Usage

### Default Client

The SDK will generate code creating a Manager. Managers hold a Cache and a Client that are used in CRUD operations and interface with the API server. The particular Client the SDK uses by default is referred to as a [split client][doc-split-client], created when instantiating a Manager. A split client reads (Get and List) from the Cache and writes (Create, Update, Delete) to the API server. Reading from the Cache significantly reduces request load on the API server; as long as the Cache is updated by the API server, read operations are eventually consistent. 

### Non-default Client

An operator developer may wish to create their own Client with different read/write optimzations, for example. controller-runtime provides a [constructor][doc-client-constr] for Clients:

```Go
// New returns a new Client using the provided config and Options.
func New(config *rest.Config, options client.Options) (client.Client, error)
```

The Manager's client field must be set with the new Client:

```Go
import (
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	
	...
)

func main() {
	...

	cfg, _ := config.GetConfig()
	mgr, _ := manager.New(cfg, manager.Options{Namespace: namespace})
	
	// Create a new Client with cfg and set the Manager's corresponding field.
	newClient := client.New(cfg, client.Options{})
	mgr.SetFields(newClient)
	
	...
}
```

### Reconcile and the Client API

A Reconciler implements the [`reconcile.Reconciler`][doc-reconcile-reconciler] interface, which exposes the Reconcile method. Reconcilers are added to a corresponding Controller for a Kind; Reconcile is called in response to cluster or external Events, with a `reconcile.Request` object argument, to read and write cluster state by the Controller, and returns a `reconcile.Result`. SDK Reconcilers have access to a Client in order to make k8s API calls.

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

```Go
// Create saves the object obj in the k8s cluster.
// Returns an error 
func (c *ClientImpl) Create(ctx context.Context, obj runtime.Object) error
```

```Go
// Update updates the given obj in the k8s cluster. obj must be a
// struct pointer so that obj can be updated with the content returned
// by the API server.
func (c *ClientImpl) Update(ctx context.Context, obj runtime.Object) error
```

```Go
// Delete deletes the given obj from k8s cluster.
func (c *ClientImpl) Delete(ctx context.Context, obj runtime.Object, opts ...DeleteOptionFunc) error
```

```Go
// Get retrieves an API object for a given object key from the k8s cluster
// and stores it in obj.
func (c *ClientImpl) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error
```

```Go
// List retrieves a list of objects for a given namespace and list options
// and stores the list in obj.
func (c *ClientImpl) List(ctx context.Context, opts *ListOptions, obj runtime.Object) error
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
[org-kubernetes-sigs]:https://github.com/kubernetes-sigs
[site-kubernetes]:https://kubernetes.io/
[site-kubernetes-controllers]:https://kubernetes.io/docs/concepts/workloads/controllers/
[site-operators]:https://coreos.com/blog/introducing-operator-framework
[doc-split-client]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/split.go#L26-L28
[doc-rest-config]:https://godoc.org/k8s.io/client-go/rest#Config
[doc-client-constr]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/client.go#L44
[doc-client-options]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/client.go#L35
[doc-reconcile-reconciler]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L35
[doc-osdk-handle]:https://github.com/operator-framework/operator-sdk/blob/master/doc/design/milestone-0.0.2/action-api.md#handler
