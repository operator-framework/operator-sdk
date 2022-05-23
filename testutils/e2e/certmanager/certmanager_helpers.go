package certmanager

import (
	"fmt"
	"strconv"

	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
)

// InstallCertManagerBundle installs CertManager onto a Kubernetes cluster
func InstallCertManagerBundle(hasv1beta1CRs bool, kubectl kubernetes.Kubectl) error {
	url, err := getCertManagerURL(hasv1beta1CRs, kubectl)
	if err != nil {
		return fmt.Errorf("encountered an error when getting the bundle URL: %w", err)
	}

	_, err = kubectl.Apply(false, "-f", url, "--validate=false")
	if err != nil {
		return fmt.Errorf("encountered an error when applying the bundle: %w", err)
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	_, err = kubectl.Wait(false, "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)
	if err != nil {
		return fmt.Errorf("encountered an error when waiting for the webhook to be ready: %w", err)
	}

	return nil
}

// UninstallCertManagerBundle uninstalls CertManager from a Kubernetes cluster
func UninstallCertManagerBundle(hasv1beta1CRs bool, kubectl kubernetes.Kubectl) error {
	url, err := getCertManagerURL(hasv1beta1CRs, kubectl)
	if err != nil {
		return fmt.Errorf("encountered an error when getting the bundle URL: %w", err)
	}

	_, err = kubectl.Delete(false, "-f", url)
	if err != nil {
		return fmt.Errorf("encountered an error when deleting the bundle: %w", err)
	}

	return nil
}

// getCertManagerUrl is a helper function to determine the CertManager
// that should be installed on a cluster based on the Kubernetes version
func getCertManagerURL(hasv1beta1CRs bool, kubectl kubernetes.Kubectl) (string, error) {
	certmanagerVersionWithv1beta2CRs := "v0.11.0"
	certmanagerLegacyVersion := "v1.0.4"
	certmanagerVersion := "v1.5.3"

	certmanagerURLTmplLegacy := "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager-legacy.yaml"
	certmanagerURLTmpl := "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"
	// Return a URL for the manifest bundle with v1beta1 CRs.

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

	if hasv1beta1CRs {
		return fmt.Sprintf(certmanagerURLTmpl, certmanagerVersionWithv1beta2CRs), nil
	}

	// Determine which URL to use for a manifest bundle with v1 CRs.
	// The most up-to-date bundle uses v1 CRDs, which were introduced in k8s v1.16.
	if serverMajor <= 1 && serverMinor < 16 {
		return fmt.Sprintf(certmanagerURLTmplLegacy, certmanagerLegacyVersion), nil
	}
	return fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion), nil
}
