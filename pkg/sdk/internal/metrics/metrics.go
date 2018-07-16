// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	eventTypesMetricName       = "operator_event_types_total"
	reconcileResultsMetricName = "operator_reconcile_results_total"
	// EventTypeLabel - metric label for event type
	EventTypeLabel = "type"
	// EventTypeAdd - addition event label
	EventTypeAdd = "add"
	// EventTypeDelete - deletion event label
	EventTypeDelete = "delete"
	// EventTypeUpdate - update event label
	EventTypeUpdate = "update"
	// ReconcileResultLabel - metric label for event result
	ReconcileResultLabel = "result"
	// ReconcileResultSuccess - successful event label
	ReconcileResultSuccess = "success"
	// ReconcileResultFailure - failed event label
	ReconcileResultFailure = "failure"
)

var (
	once sync.Once
)

// Collector - metric collector for all the metrics the sdk will watch
type Collector struct {
	EventType       *prom.CounterVec
	ReconcileResult *prom.CounterVec
}

// New - create a new Collector
func New() *Collector {
	return &Collector{
		EventType: prom.NewCounterVec(prom.CounterOpts{
			Name: eventTypesMetricName,
			Help: "events that the sdk has recieved, segmented by type(add or delete or update)",
		}, []string{EventTypeLabel}),
		ReconcileResult: prom.NewCounterVec(prom.CounterOpts{
			Name: reconcileResultsMetricName,
			Help: "reconcilation events that the sdk has processed segmented by result(success or failure)",
		}, []string{ReconcileResultLabel}),
	}
}

// RegisterCollector - add collector safely to prometheus
func RegisterCollector(c *Collector) {
	once.Do(func() {
		err := prom.Register(c)
		if err != nil {
			logrus.Errorf("unable to register collector with prometheus: %v", err)
		}
	})
}

// Describe returns all the descriptions of the collector
func (c *Collector) Describe(ch chan<- *prom.Desc) {
	c.EventType.Describe(ch)
	c.ReconcileResult.Describe(ch)

}

// Collect returns the current state of the metrics
func (c *Collector) Collect(ch chan<- prom.Metric) {
	c.EventType.Collect(ch)
	c.ReconcileResult.Collect(ch)
}
