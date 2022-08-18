package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// MetricDescription is an exported struct that defines the metric description (Name, Help)
// as a new type named MetricDescription.
type MetricDescription struct {
	Name string
	Help string
}

// metricsDescription is a map of string keys (metrics) to MetricDescription values (Name, Help).
var metricDescription = map[string]MetricDescription{
	"MemcachedDeploymentSizeUndesiredCountTotal": {
		Name: "memcached_deployment_size_undesired_count_total",
		Help: "Total number of times the deployment size was not as desired.",
	},
}

var (
	// MemcachedDeploymentSizeUndesiredCountTotal will count how many times was required
	// to perform the operation to ensure that the number of replicas on the cluster
	// is the same as the quantity desired and specified via the custom resource size spec.
	MemcachedDeploymentSizeUndesiredCountTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricDescription["MemcachedDeploymentSizeUndesiredCountTotal"].Name,
			Help: metricDescription["MemcachedDeploymentSizeUndesiredCountTotal"].Help,
		},
	)
)

// RegisterMetrics will register metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(MemcachedDeploymentSizeUndesiredCountTotal)
}

// GetMetricDesc will create a slice with the metrics available in metricDescription
func GetMetricDesc() []MetricDescription {
	v := make([]MetricDescription, 0, len(metricDescription))
	// Insert value (Name, Help) for each metric
	for _, value := range metricDescription {
		v = append(v, value)
	}

	return v
}

