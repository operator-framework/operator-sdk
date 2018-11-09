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
	"net/http"
	"os"
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "k8s.io/api/core/v1"
)

// Setup function registers go and process metrics, serves metrics and exposes Prometheus metrics port.
// It also generates and returns a Kubernetes Service to expose the metrics port.
func Setup() (*v1.Service, error) {
	reg := NewPrometheusRegistery()
	mux := http.NewServeMux()
	HandlerFuncs(mux, reg)
	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(OperatorSDKPrometheusMetricsPort), mux)
		if err != nil {
			fmt.Printf("Serving metrics failed: %v", err)
		}
	}()

	service, err := k8sutil.InitOperatorService(int32(OperatorSDKPrometheusMetricsPort), OperatorSDKPrometheusMetricsPortName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service object for operator metrics: %v", err)
	}
	return service, nil
}

// NewPrometheusRegistery returns a newly created prometheus registry.
// It also registers go collector and process collector metrics.
func NewPrometheusRegistery() *prometheus.Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
			PidFn:        func() (int, error) { return os.Getpid(), nil },
			Namespace:    "",
			ReportErrors: false,
		}),
	)
	return r
}

// HandlerFuncs registers the handles the /healthz and /metrics, to be able to serve the prometheus registry.
func HandlerFuncs(mux *http.ServeMux, reg *prometheus.Registry) {
	mux.HandleFunc("/healthz", handlerHealthz)
	mux.Handle(PrometheusMetricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}))
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
