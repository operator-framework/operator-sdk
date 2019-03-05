# Monitoring with Prometheus

[Prometheus][prometheus] is an open-source systems monitoring and alerting toolkit. Below is the overview of the different helpers that exist in operator-sdk to help setup metrics in the generated operator.

## Metrics in operator-sdk

The `func ExposeMetricsPort(ctx context.Context, port int32) (*v1.Service, error)` function exposes general metrics about the running program. These metrics are inherited from controller-runtime. This helper function creates a [Service][service] object with the metrics port exposed, which can then be accessed by Prometheus. The Service object is [garbage collected][gc] when the leader pod's root owner is deleted.

By default, the metrics are served on `0.0.0.0:8383/metrics`. To modify the port the metrics are exposed on, change the `var metricsPort int32 = 8383` variable in the `cmd/manager/main.go` file of the generated operator.

### Usage:

```go
    import(
        "github.com/operator-framework/operator-sdk/pkg/metrics"
        "sigs.k8s.io/controller-runtime/pkg/manager"
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

        // Create Service object to expose the metrics port.
        _, err = metrics.ExposeMetricsPort(ctx, metricsPort)
        if err != nil {
            // handle error
            log.Info(err.Error())
        }

        ...
    }
```

Note: The above example is already present in `cmd/manager/main.go` in all the operators generated with operator-sdk.

[prometheus]: https://prometheus.io/
[service]: https://kubernetes.io/docs/concepts/services-networking/service/
[gc]: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents

