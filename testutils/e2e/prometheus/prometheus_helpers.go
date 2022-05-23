package prometheus

import (
	"fmt"
	"strconv"

	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
)

// InstallPrometheusOperator will install the Prometheus operator onto a Kubernetes cluster
func InstallPrometheusOperator(kubectl kubernetes.Kubectl) error {
	url, err := getPrometheusOperatorUrl(kubectl)
	if err != nil {
		return fmt.Errorf("encountered an error when getting the bundle URL: %w", err)
	}

	_, err = kubectl.Apply(false, "-f", url)
	if err != nil {
		return fmt.Errorf("encountered an error when getting the bundle URL: %w", err)
	}

	return nil
}

// UninstallPrometheusOperator will uninstall a Prometheus operator from a Kubernetes cluster
func UninstallPrometheusOperator(kubectl kubernetes.Kubectl) error {
	url, err := getPrometheusOperatorUrl(kubectl)
	if err != nil {
		return fmt.Errorf("encountered an error when getting the bundle URL: %w", err)
	}
	_, err = kubectl.Apply(false, "-f", url)
	if err != nil {
		return fmt.Errorf("encountered an error when deleting the bundle: %w", err)
	}

	return nil
}

// getPrometheusOperatorUrl is a helper function to determine the Prometheus
// operator that should be installed on a cluster based on the Kubernetes version
func getPrometheusOperatorUrl(kubectl kubernetes.Kubectl) (string, error) {
	prometheusOperatorLegacyVersion := "0.33"
	prometheusOperatorLegacyURL := "https://raw.githubusercontent.com/coreos/prometheus-operator/release-%s/bundle.yaml"
	prometheusOperatorVersion := "0.51"
	prometheusOperatorURL := "https://raw.githubusercontent.com/prometheus-operator/" +
		"prometheus-operator/release-%s/bundle.yaml"

	var url string

	kubeVersion, err := kubectl.Version()
	if err != nil {
		return "", fmt.Errorf("encountered an error trying to get Kubernetes Version: %w", err)
	}
	serverMajor, err := strconv.ParseUint(kubeVersion.ServerVersion().Major(), 10, 64)
	if err != nil {
		return "", fmt.Errorf("encountered an error trying to parse Kubernetes Major Version: %w", err)
	}

	serverMinor, err := strconv.ParseUint(kubeVersion.ServerVersion().Minor(), 10, 64)
	if err != nil {
		return "", fmt.Errorf("encountered an error trying to parse Kubernetes Minor Version: %w", err)
	}

	if serverMajor <= 1 && serverMinor < 16 {
		url = fmt.Sprintf(prometheusOperatorLegacyURL, prometheusOperatorLegacyVersion)
	} else {
		url = fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	}

	return url, nil
}
