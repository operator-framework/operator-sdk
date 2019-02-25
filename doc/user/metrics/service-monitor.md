## Using the ServiceMonitor prometheus-operator CRD

[prometheus-operator][prom-operator] an operator that creates, configures and manages Prometheus clusters atop Kubernetes.

`ServiceMonitor` is a [CR][cr] of the prometheus-operator, which discovers the `Endpoints` in `Service` objects and configures Prometheus to monitor those `Pod`s. See the prometheus-operator [documention][sm] to learn more about `ServiceMonitor`s.

The `GenerateServiceMonitor` takes a `Service` object and generates a `ServiceMonitor` resource based on it. To add `Service` target discovery of your created monitoring `Service`s you can use the `metrics.CreateServiceMonitor()` helper function, which accepts the newly created `Service`.

### Prerequisites:

- [prometheus-operator][prom-quickstart] needs to be deployed already in the cluster.

### Usage example:

```go
    import(
        v1 "k8s.io/api/core/v1"
        "github.com/operator-framework/operator-sdk/pkg/metrics"
    )

    func main() {    
        
        ...

        // Populate bellow with Service(s) for which you want to create `ServiceMonitor` for.
        services := []*v1.Service{}

        // Create one `ServiceMonitor` per application per namespace.
        // Change below value to name of the Namespace you want the `ServiceMonitor` to be created in.
        ns := "default"
        
        // Pass the Service(s) to the helper function, which in turn returns the `ServiceMonitor` object.
        serviceMonitors, err := metrics.CreateServiceMonitors(restConfig, ns, services)
        if err != nil {
            // handle error here
        }

        ...
    }
```

[prom-operator]: https://github.com/coreos/prometheus-operator
[sm]: https://github.com/coreos/prometheus-operator/blob/7a25bf6b6bb2347dacb235659b73bc210117acc7/Documentation/design.md#servicemonitor
[cr]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[prom-quickstart]: https://github.com/coreos/prometheus-operator/tree/master/contrib/kube-prometheus#quickstart
