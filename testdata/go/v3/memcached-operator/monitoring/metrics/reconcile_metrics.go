package metrics

import (
	"github.com/example/memcached-operator/monitoring/metrics/util"
)

var reconcileMetrics = []util.Metric{
	{
		Name: "memcached_deployment_size_undesired_count_total",
		Help: "Total number of times the deployment size was not as desired",
		Type: util.Counter,
	},
	{
		Name: "memcached_deployment_updates_count_total",
		Help: "Total number of times the deployment resource was updated",
		Type: util.Counter,
	},
}

// IncrementNumberOfUndesiredSize should update both the metric with the total
// count of undesired sizes and the number of updates in the deployment resource
func IncrementNumberOfUndesiredSize() {
	util.GetCounterMetric("memcached_deployment_size_undesired_count_total").Inc()
	util.GetCounterMetric("memcached_deployment_updates_count_total").Inc()
}

func IncrementNumberOfDeploymentUpdates() {
	util.GetCounterMetric("memcached_deployment_updates_count_total").Inc()
}
