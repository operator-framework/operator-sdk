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

package sdk

import (
	"net/http"
	"strconv"

	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
)

// ExposeMetricsPort generate a Kubernetes Service to expose metrics port
func ExposeMetricsPort() {
	http.Handle("/"+k8sutil.PrometheusMetricsPortName, promhttp.Handler())
	go http.ListenAndServe(":"+strconv.Itoa(k8sutil.PrometheusMetricsPort), nil)

	service, err := k8sutil.InitOperatorService()
	if err != nil {
		logrus.Errorf("Failed to initialize service object for operator metrics: %v", err)
		return
	}
	err = Create(service)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Errorf("Failed to create service for operator metrics: %v", err)
		return
	}
	logrus.Infof("Metrics service %s created", service.Name)
}
