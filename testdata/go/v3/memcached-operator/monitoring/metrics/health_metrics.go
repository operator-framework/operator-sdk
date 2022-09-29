/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import "github.com/example/memcached-operator/monitoring/metrics/util"

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
