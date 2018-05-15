package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Register() error {
	http.Handle("/metrics", promhttp.Handler())
	return nil
}
