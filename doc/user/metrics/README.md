# Monitoring with Prometheus

[Prometheus][prometheus] is an open-source systems monitoring and alerting toolkit. Below is the overview of the different helpers that exist in Operator SDK to help setup metrics in the generated operator.

## Metrics in Operator SDK

### General metrics

The `CreateMetricsService(ctx context.Context, cfg *rest.Config, servicePorts []v1.ServicePort) (*v1.Service, error)` function exposes general metrics about the running program. These metrics are inherited from controller-runtime. To understand which metrics are exposed, read the metrics package doc of [controller-runtime][controller-metrics]. The function creates a [Service][service] object with the metrics port exposed, which can then be accessed by Prometheus. The Service object is [garbage collected][gc] when the leader pod's root owner is deleted.

By default, the metrics are served on `0.0.0.0:8383/metrics`. To modify the port the metrics are exposed on, change the `var metricsPort int32 = 8383` variable in the `cmd/manager/main.go` file of the generated operator.

#### Usage:

```go
    import(
        "context"

        "github.com/operator-framework/operator-sdk/pkg/metrics"
        "sigs.k8s.io/controller-runtime/pkg/manager"
        "k8s.io/api/core/v1"
        "k8s.io/apimachinery/pkg/util/intstr"
    )

    func main() {

        ...

        // Change the below variables to serve metrics on different host or port.
        var metricsHost = "0.0.0.0"
        var metricsPort int32 = 8383

        // Pass metrics address to controller-runtime manager
        mgr, err := manager.New(cfg, manager.Options{
            Namespace:          namespace,
            MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
        })

        ...

        // Add to the below struct any other metrics ports you want to expose.
	    servicePorts := []v1.ServicePort{
		    {Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
	    }

        // Create Service object to expose the metrics port.
        _, err = metrics.CreateMetricsService(context.TODO(), cfg, servicePorts)
        if err != nil {
            // handle error
        }

        ...

    }
```

*Note:* The above example is already present in `cmd/manager/main.go` in all the operators generated with Operator SDK from v0.5.0 onwards.

#### Garbage collection

The metrics Service is [garbage collected][gc] when the resource used to deploy the operator is deleted (e.g. `Deployment`). This resource is determined when the metrics Service is created, at that time the resource owner reference is added to the Service.

In Kubernetes clusters where [OwnerReferencesPermissionEnforcement][ownerref-permission] is enabled (on by default in all OpenShift clusters), the role requires a `<RESOURCE-KIND>/finalizers` rule to be added. By default when creating the operator with the Operator SDK, this is done automatically under the assumption that the `Deployment` object was used to create the operator pods. In case another method of deploying the operator is used, replace the `- deployments/finalizers` in the `deploy/role.yaml` file. Example rule from `deploy/role.yaml` file for deploying operator with a `StatefulSet`:

```yaml
...
- apiGroups:
  - apps
  resourceNames:
  - <STATEFULSET-NAME>
  resources:
  - statefulsets/finalizers
  verbs:
  - update
...
```

### Custom resource specific metrics

By default operator will expose info metrics based on the number of the current instances of an operator's custom resources in the cluster. It leverages [kube-state-metrics][ksm] as a library to generate those metrics. Metrics initialization lives in the `cmd/manager/main.go` file of the operator in the `serveCRMetrics` function. Its arguments are a custom resource's group, version, and kind to generate the metrics. The metrics are served on `0.0.0.0:8686/metrics` by default. To modify the exposed metrics port number, change the `operatorMetricsPort` variable at the top of the `cmd/manager/main.go` file in the generated operator.


[prometheus]: https://prometheus.io/
[service]: https://kubernetes.io/docs/concepts/services-networking/service/
[gc]: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents
[ownerref-permission]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
[ksm]: https://github.com/kubernetes/kube-state-metrics
[controller-metrics]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/internal/controller/metrics
