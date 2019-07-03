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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	subsystem = "ansible_operator"
)

var (
	reconcileResults = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "reconcile_result",
			Help:      "Gauge of reconciles and their results.",
		},
		[]string{
			"GVK",
			"result",
		})

	reconciles = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "reconciles",
			Help:      "How long in seconds a reconcile takes.",
		},
		[]string{
			"GVK",
		})
)

func init() {
	metrics.Registry.MustRegister(reconcileResults)
	metrics.Registry.MustRegister(reconciles)
}

// We will never want to panic our app because of metric saving.
// Therefore, we will recover our panics here and error log them
// for later diagnosis but will never fail the app.
func recoverMetricPanic() {
	if r := recover(); r != nil {
		logf.Log.WithName("metrics").Error(fmt.Errorf("%v", r),
			"Recovering from metric function")
	}
}

func ReconcileSucceeded(gvk string) {
	defer recoverMetricPanic()
	reconcileResults.WithLabelValues(gvk, "succeeded").Inc()
}

func ReconcileFailed(gvk string) {
	// TODO: consider taking in a failure reason
	defer recoverMetricPanic()
	reconcileResults.WithLabelValues(gvk, "failed").Inc()
}

func ReconcileTimer(gvk string) *prometheus.Timer {
	defer recoverMetricPanic()
	return prometheus.NewTimer(prometheus.ObserverFunc(func(duration float64) {
		reconciles.WithLabelValues(gvk).Observe(duration)
	}))
}
