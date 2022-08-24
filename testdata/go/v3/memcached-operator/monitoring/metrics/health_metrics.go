package metrics

import (
	"github.com/example/memcached-operator/monitoring/metrics/util"
)

var healthMetrics = []util.Metric{
	{
		Name: "memcached_deployment_status",
		Help: "Current status of the memcached deployment",
		Type: util.Gauge,
	},
}

func SetHealthStatus(status float64) {
	util.GetGaugeMetric("memcached_deployment_size_undesired_count_total").Set(status)
}
