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
