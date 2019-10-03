## Using the ServiceMonitor prometheus-operator CRD

- [Overview](#overview)
- [Prerequisites:](#prerequisites)
- [Usage example:](#usage-example)

## Overview 

[prometheus-operator][prom-operator] is an operator that creates, configures, and manages Prometheus clusters atop Kubernetes.

`ServiceMonitor` is a CustomResource of the prometheus-operator, which discovers the `Endpoints` in `Service` objects and configures Prometheus to monitor those pods. See the prometheus-operator [documentation][service-monitor] to learn more about `ServiceMonitor`.

The `CreateServiceMonitors` function takes `Service` objects and generates `ServiceMonitor` resources based on the endpoints. To add `Service` target discovery of your created monitoring `Service` you can use the `metrics.CreateServiceMonitors()` helper function, which accepts the newly created `Service`.

## Prerequisites:

- [prometheus-operator][prom-quickstart] needs to be deployed in the cluster.

## Usage example:

```go
    import(
        ... 
        "k8s.io/api/core/v1"
        "github.com/operator-framework/operator-sdk/pkg/metrics"
        ...
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

    // serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
    // It serves those metrics on "http://metricsHost:operatorMetricsPort".
    func serveCRMetrics(cfg *rest.Config) error {
        // Below function returns filtered operator/CustomResource specific GVKs.
        // For more control override the below GVK list with your own custom logic.
        filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
        if err != nil {
            return err
        }
        // Get the namespace the operator is currently deployed in.
        operatorNs, err := k8sutil.GetOperatorNamespace()
        if err != nil {
            return err
        }
        // To generate metrics in other namespaces, add the values below.
        ns := []string{operatorNs}
        // Generate and serve custom resource specific metrics.
        err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
        if err != nil {
            return err
        }
        return nil
    }
```

[prom-operator]: https://github.com/coreos/prometheus-operator
[service-monitor]: https://github.com/coreos/prometheus-operator/blob/7a25bf6b6bb2347dacb235659b73bc210117acc7/Documentation/design.md#servicemonitor
[prom-quickstart]: https://github.com/coreos/prometheus-operator/tree/master/contrib/kube-prometheus#quickstart
