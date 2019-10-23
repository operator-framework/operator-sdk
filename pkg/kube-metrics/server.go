// Copyright 2019 The Operator-SDK Authors
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

package kubemetrics

import (
	"fmt"
	"net"
	"net/http"

	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

func ServeMetrics(stores [][]*metricsstore.MetricsStore, host string, port int32) {
	listenAddress := net.JoinHostPort(host, fmt.Sprint(port))
	mux := http.NewServeMux()
	// Add metricsPath
	mux.Handle(metricsPath, &metricHandler{stores})
	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Operator SDK Metrics</title></head>
             <body>
             <h1>kube-metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
             <li><a href='` + healthzPath + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`))
	})
	err := http.ListenAndServe(listenAddress, mux)
	log.Error(err, "Failed to serve custom metrics")
}

type metricHandler struct {
	stores [][]*metricsstore.MetricsStore
}

func (m *metricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	// 0.0.4 is the exposition format version of prometheus
	// https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format
	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")
	for _, stores := range m.stores {
		for _, s := range stores {
			s.WriteAll(w)
		}
	}
}
