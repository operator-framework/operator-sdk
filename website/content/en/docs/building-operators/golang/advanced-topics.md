---
title: Advanced Topics
linkTitle: Advanced Topics
weight: 80
---

### Manage CR status conditions

An often-used pattern is to include `Conditions` in the status of custom resources. A [`Condition`][apimachinery_condition] represents the latest available observations of an object's state (see the [Kubernetes API conventions documentation][typical-status-properties] for more information).

The `Conditions` field added to the `MemcachedStatus` struct simplifies the management of your CR's conditions. It:
- Enables callers to add and remove conditions.
- Ensures that there are no duplicates.
- Sorts the conditions deterministically to avoid unnecessary repeated reconciliations.
- Automatically handles the each condition's `LastTransitionTime`.
- Provides helper methods to make it easy to determine the state of a condition.

To use conditions in your custom resource, add a Conditions field to the Status struct in `_types.go`:

```Go
import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MyAppStatus struct {
    // Conditions represent the latest available observations of an object's state
    Conditions []metav1.Condition `json:"conditions"`
}
```

Then, in your controller, you can use [`Conditions`][helpers-conditions] methods to make it easier to set and remove conditions or check their current values.

### Adding 3rd Party Resources To Your Operator


The operator's Manager supports the core Kubernetes resource types as found in the client-go [scheme][scheme_package] package and will also register the schemes of all custom resource types defined in your project.

```Go
import (
    cachev1alpha1 "github.com/example/memcached-operator/api/v1alpha1
    ...
)

func init() {

    // Setup Scheme for all resources
    utilruntime.Must(cachev1alpha1.AddToScheme(scheme))
    //+kubebuilder:scaffold:scheme
}
```

To add a 3rd party resource to an operator, you must add it to the Manager's scheme. By creating an `AddToScheme()` method or reusing one you can easily add a resource to your scheme. An [example][deployments_register] shows that you define a function and then use the [runtime][runtime_package] package to create a `SchemeBuilder`.

#### Register with the Manager's scheme

Call the `AddToScheme()` function for your 3rd party resource and pass it the Manager's scheme via `mgr.GetScheme()` or `scheme` in `main.go`.
Example:
```go
import (
    routev1 "github.com/openshift/api/route/v1"
)

func init() {
    ...

    // Adding the routev1
    utilruntime.Must(clientgoscheme.AddToScheme(scheme))

    utilruntime.Must(routev1.AddToScheme(scheme))
    //+kubebuilder:scaffold:scheme

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

### Monitoring and Observability
This section covers how to create custom metrics, [alerts] and [recording rules] for your operator. It focuses on the technical aspects, and demonstrates the implementation by updating the sample [memcached-operator].

For more information regarding monitoring best practices, take a look at our docs on [observability-best-practices].  

#### Prerequisites
The following steps are required in order to inspect the operator's custom metrics, alerts and recording rules:
- Install Prometheus and Prometheus Operator. We recommend using [kube-prometheus] in production if you don’t have your own monitoring system. If you are just experimenting, you can only install Prometheus and Prometheus Operator.
- Make sure Prometheus has access to the operator's namespace, by setting the corresponding RBAC rules.
  
  Example: [prometheus_role.yaml] and [prometheus_role_binding.yaml]

#### Publishing Custom Metrics
If you wish to publish custom metrics for your operator, this can be easily achieved by using the global registry from `controller-runtime/pkg/metrics`.
One way to achieve this is to declare your collectors as global variables, register them using `RegisterMetrics()` and call it in the controller's `init()` function.

Example custom metric: [MemcachedDeploymentSizeUndesiredCountTotal]

```go
package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	MemcachedDeploymentSizeUndesiredCountTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "memcached_deployment_size_undesired_count_total",
			Help: "Total number of times the deployment size was not as desired.",
		},
	)
)

// RegisterMetrics will register metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(MemcachedDeploymentSizeUndesiredCountTotal)
}
```

- The above example creates a new `Counter` metric. For other metrics' types, see [Prometheus Documentation].
- For more information regarding operators metrics best-practices, please follow [observability-best-practices].

[init() function example]:

```go
package main


import (
   ...
   "github.com/example/memcached-operator/monitoring"
)

func init() {
   ...
   monitoring.RegisterMetrics()
   ...
}
```

The next step would be to set the controller's logic according to which we update the metric's value. In this case, the new metric type is `Counter`, thus a valid update operation would be to increment its value. 

[Metric update example]:

```go
...
size := memcached.Spec.Size
if *found.Spec.Replicas != size {
    // Increment MemcachedDeploymentSizeUndesiredCountTotal metric by 1
    monitoring.MemcachedDeploymentSizeUndesiredCountTotal.Inc()
}
...
```
Different metrics types have different valid operations. For more information, please follow [Prometheus Golang client].

#### Publishing Alerts and Recording Rules
In order to add alerts and recording rules, which are unique to the operator's needs, we'll create a dedicated PrometheusRule object, by using [prometheus-operator API].

[PrometheusRule example]:

```go
package monitoring

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NewPrometheusRule creates new PrometheusRule(CR) for the operator to have alerts and recording rules
func NewPrometheusRule(namespace string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "memcached-operator-rules",
			Namespace: "memcached-operator-system",
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alerts and recording rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	return &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: "memcached.rules",
			Rules: []monitoringv1.Rule{
				createOperatorUpTotalRecordingRule(), 
				createOperatorDownAlertRule()
			},
		}},
	}
}

// createOperatorUpTotalRecordingRule creates memcached_operator_up_total recording rule
func createOperatorUpTotalRecordingRule() monitoringv1.Rule {
   return monitoringv1.Rule{
      Record: "memcached_operator_up_total",
      Expr:   intstr.FromString("sum(up{pod=~'memcached-operator-controller-manager-.*'} or vector(0))"),
   }
}

// createOperatorDownAlertRule creates MemcachedOperatorDown alert rule
func createOperatorDownAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: "MemcachedOperatorDown",
		Expr:  intstr.FromString("memcached_operator_up_total == 0"),
		Annotations: map[string]string{
			"description": "No running memcached-operator pods were detected in the last 5 min.",
		},
		For: "5m",
		Labels: map[string]string{
			"severity":    "critical",
		},
	}
}
```

Then, we may want to ensure that the new PrometheusRule is being created and reconciled. One way to achieve this is by expanding the existing `Reconcile()` function logic.

[PrometheusRule reconciliation example]:

```go
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    ...
    ...
    // Check if prometheus rule already exists, if not create a new one
    foundRule := &monitoringv1.PrometheusRule{}
    err := r.Get(ctx, types.NamespacedName{Name: ruleName, Namespace: namespace}, foundRule)
    if err != nil && apierrors.IsNotFound(err) {
        // Define a new prometheus rule
        prometheusRule := monitoring.NewPrometheusRule(namespace)
        if err := r.Create(ctx, prometheusRule); err != nil {
            log.Error(err, "Failed to create prometheus rule")
            return ctrl.Result{}, nil
        }
    }
    
    if err == nil {
    // Check if prometheus rule spec was changed, if so set as desired
    desiredRuleSpec := monitoring.NewPrometheusRuleSpec()
    if !reflect.DeepEqual(foundRule.Spec.DeepCopy(), desiredRuleSpec) {
        desiredRuleSpec.DeepCopyInto(&foundRule.Spec)
        if r.Update(ctx, foundRule); err != nil {
            log.Error(err, "Failed to update prometheus rule")
            return ctrl.Result{}, nil
        }
    }
    ...
    ...
}
```

- Please review the [observability-best-practices] for additional important information regarding alerts and recording rules.


#### Alerts Unit Testing
It is highly recommended implementing unit tests for prometheus rules. For more information, please follow the Prometheus [unit testing documentation]. For examples of unit testing in a Golang operator, see the sample memcached-operator [alerts unit tests].

#### Inspecting the metrics, alerts and recording rules with Prometheus UI
Finally, in order to inspect the exposed metrics and alerts, we need to forward the corresponding port where metrics are published by Prometheus (usually `9090`, which is the default value). This can be done with the following command:
```bash
$ kubectl -n monitoring port-forward svc/prometheus-k8s 9090
```


where we assume that the prometheus service is available in the `monitoring` namespace.

Now you can access Prometheus UI using `http://localhost:9090`. For more details on exposing prometheus metrics, please refer [kube-prometheus docs].

### Handle Cleanup on Deletion

Operators may create objects as part of their operational duty. Object accumulation can consume unnecessary resources, slow down the API and clutter the user interface. As such it is important for operators to keep good hygiene and to clean up resources when they are not needed. Here are a few common scenarios.
 
#### Internal Resources

A typical example of correct resource cleanup is the [Jobs][jobs] implementation. When a Job is created, one or multiple Pods are created as child resources. When a Job is deleted, the associated Pods are deleted as well. This is a very common pattern easily achieved by setting an owner reference from the parent (Job) to the child (Pod) object. Here is a code snippet for doing so, where "r" is the reconcilier and "ctrl" the controller-runtime library:

```go
ctrl.SetControllerReference(job, pod, r.Scheme)
```

Note that the default behavior for cascading deletion is background propagation, meaning deletion requests for child objects occur after the request to delete the parent object. [This Kubernetes doc][garbage_collection] provides alternative deletion types.

#### External Resources

Sometimes external resources or resources that are not owned by a custom resource, those across namespaces for example, need to be cleaned up when the parent resource is deleted. In that case [Finalizers][finalizers] can be leveraged. A deletion request for an object with a finalizer becomes an update during which a deletion timestamp is set; the object is not deleted while the finalizer is present. The reconciliation loop of the custom resource's controller will then need to check whether a the deletion timestamp is set, perform the external cleanup operation(s), then remove the finalizer to allow garbage collection of the object. Multiple finalizers may be present on an object, each with a key that should indicate what external resources require deletion by the controller.

The following is a snippet from a theoretical controller file `controllers/memcached_controller.go` that implements a finalizer handler:


```go
import (
    ...
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const memcachedFinalizer = "cache.example.com/finalizer"

func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
        if controllerutil.ContainsFinalizer(memcached, memcachedFinalizer) {
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
    if !controllerutil.ContainsFinalizer(memcached, memcachedFinalizer) {
        controllerutil.AddFinalizer(memcached, memcachedFinalizer)
        err = r.Update(ctx, memcached)
        if err != nil {
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
```

#### Complex cleanup logic

Similar to the previous scenario, finalizers can be used for implementing complex cleanup logic. Take [CronJobs][cronjobs] as an example: the controller maintains limited-size lists of jobs that have been created by the CronJob controller to check for deletion. These list sizes are configured by the CronJob fields [`.spec.successfulJobsHistoryLimit` and `.spec.failedJobsHistoryLimit`][cronjob_fields], which specify how many completed and failed jobs should be kept. Check out the [Kubebuilder CronJob tutorial][cronjob_tutorial] for full implementation details.

#### Sensitive resources

Sensitive resources need to be protected against unintended deletion. An intuitive example of protecting resources is the [PersistentVolume (PV) / PersistentVolumeClaim (PVC)][pv] relationship. A PV is first created, after which users can request access to that PV's storage by creating a PVC, which gets bound to the PV. If a user tries to delete a PV currently bound by a PVC, the PV is not removed immediately. Instead, PV removal is postponed until the PV is not bound to any PVC. Finalizers again can be leveraged to achieve a similar behaviour for your own PV-like custom resources: by setting a finalizer on an object, your controller can make sure there are no remaining objects bound to it before removing the finalizer and deleting the object.
Additionally, the user who created the PVC can specify what happens to the underlying storage allocated in a PV when the PVC is deleted through the [reclaim policy][reclaiming]. There are several options available, each of which defines a behavior that is achieved again through the use of finalizers. The key concept to take away is that your operator can give a user the power to decide how their resources are cleaned up via finalizers, which may be dangerous yet useful depending on your workloads.

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
    "github.com/operator-framework/operator-lib/leader"
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

### Multiple architectures

Authors may decide to distribute their bundles for various architectures: x86_64, aarch64, ppc64le, s390x, etc, to accommodate the diversity of Kubernetes clusters and reach a larger number of potential users. Each architecture requires however compatible binaries. Considerations on the topic are available in the [Multiple Architectures page][multi_arch].

[typical-status-properties]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[scheme_package]:https://github.com/kubernetes/client-go/blob/master/kubernetes/scheme/register.go
[deployments_register]: https://github.com/kubernetes/api/blob/master/apps/v1/register.go#L41
[runtime_package]: https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime
[scheme_builder]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/scheme#Builder
[metrics_doc]: https://book.kubebuilder.io/reference/metrics.html
[jobs]: https://kubernetes.io/docs/concepts/workloads/controllers/job/
[garbage_collection]: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
[finalizers]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#finalizers
[cronjobs]: https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/
[cronjob_fields]: https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#jobs-history-limits
[cronjob_tutorial]: https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html#3-clean-up-old-jobs-according-to-the-history-limit
[pv]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[reclaiming]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming
[lease_split_brain]: https://github.com/kubernetes/client-go/blob/30b06a83d67458700a5378239df6b96948cb9160/tools/leaderelection/leaderelection.go#L21-L24
[leader_for_life]: https://pkg.go.dev/github.com/operator-framework/operator-lib/leader
[leader_with_lease]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/leaderelection
[pod_eviction_timeout]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/#options
[manager_options]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Options
[apimachinery_condition]: https://github.com/kubernetes/apimachinery/blob/d4f471b82f0a17cda946aeba446770563f92114d/pkg/apis/meta/v1/types.go#L1368
[helpers-conditions]: https://github.com/kubernetes/apimachinery/blob/master/pkg/api/meta/conditions.go
[multi_arch]:/docs/advanced-topics/multi-arch
[observability-best-practices]:https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/
[alerts]:https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
[recording rules]:https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
[prometheus_role.yaml]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/config/rbac/prometheus_role.yaml
[prometheus_role_binding.yaml]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/config/rbac/prometheus_role_binding.yaml
[MemcachedDeploymentSizeUndesiredCountTotal]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/monitoring/metrics.go
[init() function example]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/cmd/main.go
[Metric update example]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/internal/controller/memcached_controller.go
[Prometheus Documentation]:https://prometheus.io/docs/concepts/metric_types/
[Prometheus Golang client]:https://pkg.go.dev/github.com/prometheus/client_golang/prometheus
[kube-prometheus]:https://github.com/prometheus-operator/kube-prometheus
[memcached-operator]:https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator
[prometheus-operator API]:https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/api-reference/api.md
[PrometheusRule example]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/monitoring/alerts.go
[PrometheusRule reconciliation example]:https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/monitoring/memcached-operator/internal/controller/memcached_controller.go
[unit testing documentation]:https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/
[alerts unit tests]:https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/monitoring/prom-rule-ci
[kube-prometheus docs]:https://github.com/prometheus-operator/kube-prometheus/blob/main/docs/access-ui.md#prometheus