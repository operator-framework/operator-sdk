package metrics

import "github.com/example/memcached-operator/monitoring/metrics/util"

var metrics = [][]util.Metric{
	healthMetrics,
	reconcileMetrics,
}

func RegisterMetrics() {
	util.RegisterMetrics(metrics)
}

func ListMetrics() []util.Metric {
	return util.ListMetrics(metrics)
}
