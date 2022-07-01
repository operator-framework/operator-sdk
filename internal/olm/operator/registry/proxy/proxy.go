package proxy

import (
	"context"
	"fmt"
	"strings"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetProxyVars is a utility function to the Proxy configuration of a cluster if the cluster
// has the `config.openshift.io/v1` api with the `Proxy` resource present.
// GetProxyVars returns an EnvVar list with the proxy environment variables specified
// by the `Proxy` object with the name `cluster` and no namespace. If an error occurs during
// this process, a nil EnvVar list is returned along with an error.
// If the `config.openshift.io/v1` api is not found to be present on the cluster OR
// the `Proxy` object with name `cluster` is not found, the EnvVar list returned will be nil with no error.
// For more information on OpenShift Proxy configuration, see:
// https://docs.openshift.com/container-platform/4.10/networking/enable-cluster-wide-proxy.html
func GetProxyVars(cfg *operator.Configuration, discov discovery.DiscoveryInterface) ([]corev1.EnvVar, error) {
	proxyEnv := []corev1.EnvVar{}

	if cfg == nil {
		return nil, fmt.Errorf("configuration can not be nil when trying to get the Proxy config")
	}

	_, err := discov.ServerResourcesForGroupVersion(apiconfigv1.GroupVersion.Identifier())
	if err != nil {
		// if the error is that the `config.openshift.io/v1` api does not exist just return nil
		if isMissingResourceError(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("encountered an error getting server resources for GroupVersion `%s`: %w", apiconfigv1.GroupVersion.Identifier(), err)
	}

	// If we made it here the `config.openshift.io/v1` api exists and should
	// contain the `Proxy` resource, so lets try to query for the `Proxy` object that
	// should exist with the name `cluster` and no namespace
	proxy := &apiconfigv1.Proxy{}
	cfg.Scheme.AddKnownTypeWithName(apiconfigv1.SchemeGroupVersion.WithKind("Proxy"), proxy)

	err = cfg.Client.Get(context.Background(), client.ObjectKey{Name: "cluster"}, proxy)
	if err != nil {
		// if the Proxy is not found then return nil
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}

	// Go through the `Proxy` resources `ProxyStatus` fields
	// and append the corresponding proxy vars
	if proxy.Status.HTTPProxy != "" {
		proxyEnv = append(proxyEnv, corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: proxy.Status.HTTPProxy,
		}, corev1.EnvVar{
			Name:  "http_proxy",
			Value: proxy.Status.HTTPProxy,
		})
	}

	if proxy.Status.HTTPSProxy != "" {
		proxyEnv = append(proxyEnv, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: proxy.Status.HTTPSProxy,
		}, corev1.EnvVar{
			Name:  "https_proxy",
			Value: proxy.Status.HTTPSProxy,
		})
	}

	if proxy.Status.NoProxy != "" {
		proxyEnv = append(proxyEnv, corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: proxy.Status.NoProxy,
		}, corev1.EnvVar{
			Name:  "no_proxy",
			Value: proxy.Status.NoProxy,
		})
	}

	return proxyEnv, nil
}

// isMissingResourceError is a utility function to help
// determine if an error from the discovery client states
// that the requested resource could not be found.
func isMissingResourceError(err error) bool {
	check := "the server could not find the requested resource"
	return strings.Contains(err.Error(), check)
}
