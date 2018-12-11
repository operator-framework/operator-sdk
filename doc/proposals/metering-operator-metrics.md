## Auto register operator specific metrics as part of operator-metering

### Motivation and goal

We want to be able to generate the metering reports based on the operator specific Prometheus metrics. In order to be able to do that, operators must be instrumented to expose those metrics, and the operator-sdk should make this as easy as possible. The goal is to have the metering happen based on the usage of each individual operator. Metrics will be based on objects managed by the particular operator.

### Overview of the metrics

To follow both the Prometheus instrumentation [best practices](https://prometheus.io/docs/practices/naming/) as well as the official Kubernetes instrumentation [guide](https://github.com/kubernetes/community/blob/master/contributors/devel/instrumentation.md), the metrics will have the following format:

```
crd_kind_info{namespace="namespace",crdkind="instance-name"} 1
```

example metric for the memchached-operator would look like this:

```
memcached_info{namespace="default",memcached="example-memcached"} 1
```

### kube-state-metrics based solution

The solution makes use of Kubernetes list/watch to populate a Prometheus metrics registry, kube-state-metrics implements its own registry for performance reasons. [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics#overview) is used because it solves exactly the same problem we are facing but it does it for upstream known resources. The operator-sdk can re-use its functionality to perform the same thing but with custom resources. The kube-state-metrics library can only be used for constant (/static) metrics, metrics that are immutable and thereby entirely regenerated on change. This is perfect for our above mentioned use-case. It is not meant to do e.g. counting in performance critical code paths. Thereby an operator would need kube-state-metrics library for exposing the amount of custom resources that it manages and its details and Prometheus client_golang to expose metrics of its own internals e.g. count of reconciliation loops.


```go
// NewCollectors returns a collection of metrics in the namespaces provided, per the api/kind resource.
// The metrics are registered in the custom generateStore function that needs to be defined.
func NewCollectors(
    client *Client,
    namespaces []string,
    api string,
    kind string,
    metricsGenerator func(obj interface{}) []*metrics.Metric) (collectors []*kcoll.Collector)
```

```go
// ServeMetrics takes in the collectors that were created and port number on which the metrics will be served.
func ServeMetrics(collectors []*kcoll.Collector, portNumber int) {

```

Note: Due to taking advantage of kube-state-metrics functions and interfaces we cannot use the prometheus/client_golang and we need to register it in the same way as kube-state-metrics does, and expose the `/metrics` and serve it on a port (port `:8389/metrics` for example). For that we will need to also create a [Service](https://kubernetes.io/docs/concepts/services-networking/service/) object or rather update the current Service object.

### User facing architecture

Below is how roughly an example for kube-state-metrics implementation will look like.

User will have all the below code already generated and included as part of the `main.go` file:

```go
	c := metrics.NewCollectors(client, []string{"default"}, resource, kind, MetricsGenerator)
	metrics.ServeMetrics(c)
```

with the `MetricsGenerator` function living in the users `pkg/metrics` package:

```go
var (
	descMemInfo = ksmetrics.NewMetricFamilyDef(
		"memcached_info",
		"The information of the resource instance.",
		[]string{"namespace", "memcached"},
		nil,
	)
)

 func MetricsGenerator(obj interface{}) []*ksmetrics.Metric {
	ms := []*ksmetrics.Metric{}
	crdp := obj.(*unstructured.Unstructured)
 	crd := *crdp

	lv := []string{crd.GetNamespace(), crd.GetName()}
	m, err := ksmetrics.NewMetric(descMemInfo.Name, descMemInfo.LabelKeys, lv,  float64(1))
	if err != nil {
		fmt.Println(err)
		return ms
	}
	ms = append(ms, m)
	return ms
}
```

### Related work

In the future if the agreed on kube-state-metrics restructure happens (see https://github.com/kubernetes/kube-state-metrics/issues/579) we can get rid of some of the duplicated functions. But that will probably take a few months and our user facing interface should not change as a result.
