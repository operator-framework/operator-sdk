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
