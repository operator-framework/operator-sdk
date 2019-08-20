## Using the ServiceMonitor prometheus-operator CRD

[prometheus-operator][prom-operator] is an operator that creates, configures, and manages Prometheus clusters atop Kubernetes.

`ServiceMonitor` is a CustomResource of the prometheus-operator, which discovers the `Endpoints` in `Service` objects and configures Prometheus to monitor those pods. See the prometheus-operator [documentation][service-monitor] to learn more about `ServiceMonitor`.

The `CreateServiceMonitors` function takes `Service` objects and generates `ServiceMonitor` resources based on the endpoints. To add `Service` target discovery of your created monitoring `Service` you can use the `metrics.CreateServiceMonitors()` helper function, which accepts the newly created `Service`.

### Prerequisites:

- [prometheus-operator][prom-quickstart] needs to be deployed in the cluster.

### Usage example:

```go
    import(
        "k8s.io/api/core/v1"
        "github.com/operator-framework/operator-sdk/pkg/metrics"
    )

    func main() {

        ...

        // Populate below with the Service(s) for which you want to create ServiceMonitors.
        services := []*v1.Service{}

        // Create one `ServiceMonitor` per application per namespace.
        // Change below value to name of the Namespace you want the `ServiceMonitor` to be created in.
        ns := "default"

        // Pass the Service(s) to the helper function, which in turn returns the array of `ServiceMonitor` objects.
        serviceMonitors, err := metrics.CreateServiceMonitors(restConfig, ns, services)
        if err != nil {
            // handle error here
        }

        ...
    }
```

[prom-operator]: https://github.com/coreos/prometheus-operator
[service-monitor]: https://github.com/coreos/prometheus-operator/blob/7a25bf6b6bb2347dacb235659b73bc210117acc7/Documentation/design.md#servicemonitor
[prom-quickstart]: https://github.com/coreos/prometheus-operator/tree/master/contrib/kube-prometheus#quickstart
