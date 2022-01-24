// Copyright 2022 The Operator-SDK Authors
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

package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/operator-framework/operator-sdk/internal/ansible/metrics"
)

var log = logf.Log.WithName("apiserver")

type Options struct{}

func Run(done chan error, options Options) error {

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metricsHandler)

	server := http.Server{
		Addr:    "localhost:5050",
		Handler: mux,
	}
	go func() {
		log.Info("Starting to serve", "Address", server.Addr)
		done <- server.ListenAndServe()
	}()
	return nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf("Request: %+v", r))

	var userMetric metrics.UserMetric

	switch r.Method {
	case "POST":
		log.Info("apiserver has received a POST")
		log.Info("The POST BODY", "Body", r.Body)
		err := json.NewDecoder(r.Body).Decode(&userMetric)
		if err != nil {
			log.Info(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Info("apiserver is about to attempt to handle userMetric")
		err = metrics.HandleUserMetric(crmetrics.Registry, userMetric)
		if err != nil {
			log.Info(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}
