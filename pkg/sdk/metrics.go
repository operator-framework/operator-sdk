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
		logrus.Fatalf("Failed to init operator service: %v", err)
	}
	err = Create(service)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Infof("Failed to create operator service: %v", err)
		return
	}
	logrus.Infof("Metrics service %s created", service.Name)
}
