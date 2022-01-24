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
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	sdkVersion "github.com/operator-framework/operator-sdk/internal/version"
)

const (
	subsystem = "ansible_operator"
)

var (
	buildInfo = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "build_info",
			Help:      "Build information for the ansible-operator binary",
			ConstLabels: map[string]string{
				"commit":  sdkVersion.GitCommit,
				"version": sdkVersion.Version,
			},
		})

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

	userMetrics = map[string]prometheus.Collector{}
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

func RegisterBuildInfo(r prometheus.Registerer) {
	buildInfo.Set(1)
	r.MustRegister(buildInfo)
}

type UserMetric struct {
	Name      string               `json:"name" yaml:"name"`
	Help      string               `json:"description" yaml:"description"`
	Counter   *UserMetricCounter   `json:"counter,omitempty" yaml:"counter,omitempty"`
	Gauge     *UserMetricGauge     `json:"gauge,omitempty" yaml:"gauge,omitempty"`
	Histogram *UserMetricHistogram `json:"histogram,omitempty" yaml:"histogram,omitempty"`
	Summary   *UserMetricSummary   `json:"summary,omitempty" yaml:"summary,omitempty"`
}

type UserMetricCounter struct {
	Inc bool    `json:"increment,omitempty" yaml:"increment,omitempty"`
	Add float64 `json:"add,omitempty" yaml:"add,omitempty"`
}

type UserMetricGauge struct {
	Set              float64 `json:"set,omitempty" yaml:"set,omitempty"`
	Inc              bool    `json:"increment,omitempty" yaml:"increment,omitempty"`
	Dec              bool    `json:"decrement,omitempty" yaml:"decrement,omitempty"`
	Add              float64 `json:"add,omitempty" yaml:"add,omitempty"`
	Sub              float64 `json:"subtract,omitempty" yaml:"subtract,omitempty"`
	SetToCurrentTime bool    `json:"set_to_current_time,omitempty" yaml:"set_to_current_time,omitempty"`
}

type UserMetricHistogram struct {
	Observe float64 `json:"observe,omitempty" yaml:"observe,omitempty"`
}

type UserMetricSummary struct {
	Observe float64 `json:"observe,omitempty" yaml:"observe,omitempty"`
}

func HandleUserMetric(r prometheus.Registerer, metricSpec UserMetric) error {
	defer recoverMetricPanic()
	logf.Log.WithName("metrics").Info("Registering", "metric", metricSpec.Name)
	_, ok := userMetrics[metricSpec.Name]
	if ok != true {
		// This is the first time we've seen this metric
		if metricSpec.Counter != nil {
			userMetrics[metricSpec.Name] = prometheus.NewCounter(prometheus.CounterOpts{
				Name: metricSpec.Name,
				Help: metricSpec.Help,
			})
		}
		if metricSpec.Gauge != nil {
			userMetrics[metricSpec.Name] = prometheus.NewGauge(prometheus.GaugeOpts{
				Name: metricSpec.Name,
				Help: metricSpec.Help,
			})
		}
		if metricSpec.Histogram != nil {
			userMetrics[metricSpec.Name] = prometheus.NewHistogram(prometheus.HistogramOpts{
				Name: metricSpec.Name,
				Help: metricSpec.Help,
			})
		}
		if metricSpec.Summary != nil {
			userMetrics[metricSpec.Name] = prometheus.NewSummary(prometheus.SummaryOpts{
				Name: metricSpec.Name,
				Help: metricSpec.Help,
			})
		}
		r.MustRegister(userMetrics[metricSpec.Name])
	}

	collector := userMetrics[metricSpec.Name]
	switch v := collector.(type) {
	case prometheus.Gauge:
		if metricSpec.Gauge == nil {
			return errors.New("Cannot change metric type of metricSpec.Name, which is a gauge metric")
		}
		if metricSpec.Gauge.Inc == true {
			v.Inc()
		} else if metricSpec.Gauge.Dec == true {
			v.Dec()
		} else if metricSpec.Gauge.Add != 0.0 {
			v.Add(metricSpec.Gauge.Add)
		} else if metricSpec.Gauge.Sub != 0.0 {
			v.Sub(metricSpec.Gauge.Sub)
		} else if metricSpec.Gauge.Set != 0.0 {
			v.Set(metricSpec.Gauge.Set)
		} else if metricSpec.Gauge.SetToCurrentTime == true {
			v.SetToCurrentTime()
		}
	// Counter must be first, because otherwise it can be confused with a gauge.
	case prometheus.Counter:
		if metricSpec.Counter == nil {
			return errors.New("Cannot change metric type of metricSpec.Name, which is a counter metric")
		}
		if metricSpec.Counter.Inc == true {
			v.Inc()
		} else if metricSpec.Counter.Add != 0.0 {
			v.Add(metricSpec.Counter.Add)
		}
	case prometheus.Histogram:
		if metricSpec.Histogram == nil {
			return errors.New("Cannot change metric type of metricSpec.Name, which is a histogram metric")
		}
		if metricSpec.Histogram.Observe != 0.0 {
			v.Observe(metricSpec.Histogram.Observe)
		}
	case prometheus.Summary:
		if metricSpec.Summary == nil {
			return errors.New("Cannot change metric type of metricSpec.Name, which is a summary metric")
		}
		if metricSpec.Summary.Observe != 0.0 {
			v.Observe(metricSpec.Summary.Observe)
		}
	}
	return nil
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
