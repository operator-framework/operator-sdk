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
	"context"
	"net/http"
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("metrics")

// ExposeMetricsPort generate a Kubernetes Service to expose metrics port
func ExposeMetricsPort() *v1.Service {
	http.Handle("/"+k8sutil.PrometheusMetricsPortName, promhttp.Handler())
	go http.ListenAndServe(":"+strconv.Itoa(k8sutil.PrometheusMetricsPort), nil)

	service, err := k8sutil.InitOperatorService()
	if err != nil {
		log.Error(err, "failed to initialize service object for operator metrics")
		return nil
	}
	kubeconfig, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	runtimeClient, err := client.New(kubeconfig, client.Options{})
	if err != nil {
		panic(err)
	}
	err = runtimeClient.Create(context.TODO(), service)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Error(err, "failed to create service for operator metrics")
		return nil
	}

	log.Info("Metrics service created.", "ServiceName", service.Name)
	return service
}
