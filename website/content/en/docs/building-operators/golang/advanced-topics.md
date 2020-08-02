---
title: Advanced Topics
linkTitle: Advanced Topics
weight: 70
---

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


The operator's Manager supports the core Kubernetes resource types as found in the client-go [scheme][scheme_package] package and will also register the schemes of all custom resource types defined in your project.

```Go
import (
    cachev1alpha1 "github.com/example-inc/memcached-operator/api/v1alpha1
    ...
)

func init() {

    // Setup Scheme for all resources
    utilruntime.Must(cachev1alpha1.AddToScheme(mgr.GetScheme()))
    // +kubebuilder:scaffold:scheme
}
```

To add a 3rd party resource to an operator, you must add it to the Manager's scheme. By creating an `AddToScheme()` method or reusing one you can easily add a resource to your scheme. An [example][deployments_register] shows that you define a function and then use the [runtime][runtime_package] package to create a `SchemeBuilder`.

#### Register with the Manager's scheme

Call the `AddToScheme()` function for your 3rd party resource and pass it the Manager's scheme via `mgr.GetScheme()` in `main.go`.
Example:
```go
import (
    routev1 "github.com/openshift/api/route/v1"
)

func init() {
    ...

    // Adding the routev1
    utilruntime.Must(clientgoscheme.AddToScheme(mgr.GetScheme()))

    utilruntime.Must(routev1.AddToScheme(mgr.GetScheme()))
    // +kubebuilder:scaffold:scheme

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

func init() {
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

To learn about how metrics work in the Operator SDK read the [metrics section][metrics_doc] of the Kubebuilder documentation.


### Handle Cleanup on Deletion

To implement complex deletion logic, you can add a finalizer to your Custom Resource. This will prevent your Custom Resource from being
deleted until you remove the finalizer (ie, after your cleanup logic has successfully run). For more information, see the
[official Kubernetes documentation on finalizers](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#finalizers).

**Example:**

The following is a snippet from the controller file under `pkg/controller/memcached/memcached_controller.go`

```Go
import (
    ...
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const memcachedFinalizer = "finalizer.cache.example.com"

func (r *MemcachedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
    ctx := context.Background()
    reqLogger := r.log.WithValues("memcached", req.NamespacedName)
    reqLogger.Info("Reconciling Memcached")

    // Fetch the Memcached instance
    memcached := &cachev1alpha1.Memcached{}
    err := r.Get(ctx, req.NamespacedName, memcached)
    if err != nil {
        if errors.IsNotFound(err) {
            // Request object not found, could have been deleted after reconcile request.
            // Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
            // Return and don't requeue
            reqLogger.Info("Memcached resource not found. Ignoring since object must be deleted.")
            return ctrl.Result{}, nil
        }
        // Error reading the object - requeue the request.
        reqLogger.Error(err, "Failed to get Memcached.")
        return ctrl.Result{}, err
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
                return ctrl.Result{}, err
            }

            // Remove memcachedFinalizer. Once all finalizers have been
            // removed, the object will be deleted.
            controllerutil.RemoveFinalizer(memcached, memcachedFinalizer)
            err := r.Update(ctx, memcached)
            if err != nil {
                return ctrl.Result{}, err
            }
        }
        return ctrl.Result{}, nil
    }

    // Add finalizer for this CR
    if !contains(memcached.GetFinalizers(), memcachedFinalizer) {
        if err := r.addFinalizer(reqLogger, memcached); err != nil {
            return ctrl.Result{}, err
        }
    }

    ...

    return ctrl.Result{}, nil
}

func (r *MemcachedReconciler) finalizeMemcached(reqLogger logr.Logger, m *cachev1alpha1.Memcached) error {
    // TODO(user): Add the cleanup steps that the operator
    // needs to do before the CR can be deleted. Examples
    // of finalizers include performing backups and deleting
    // resources that are not owned by this CR, like a PVC.
    reqLogger.Info("Successfully finalized memcached")
    return nil
}

func (r *MemcachedReconciler) addFinalizer(reqLogger logr.Logger, m *cachev1alpha1.Memcached) error {
    reqLogger.Info("Adding Finalizer for the Memcached")
    controllerutil.AddFinalizer(m, memcachedFinalizer)

    // Update CR
    err := r.Update(context.TODO(), m)
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
```

### Leader election

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

[typical-status-properties]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[godoc-conditions]: https://godoc.org/github.com/operator-framework/operator-lib/status#Conditions
[scheme_package]:https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
[deployments_register]: https://github.com/kubernetes/api/blob/master/apps/v1/register.go#L41
[runtime_package]: https://godoc.org/k8s.io/apimachinery/pkg/runtime
[scheme_builder]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/scheme#Builder
[metrics_doc]: https://book.kubebuilder.io/reference/metrics.html
[lease_split_brain]: https://github.com/kubernetes/client-go/blob/30b06a83d67458700a5378239df6b96948cb9160/tools/leaderelection/leaderelection.go#L21-L24
[leader_for_life]: https://godoc.org/github.com/operator-framework/operator-lib/leader
[leader_with_lease]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/leaderelection
[pod_eviction_timeout]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/#options
[manager_options]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/manager#Options
