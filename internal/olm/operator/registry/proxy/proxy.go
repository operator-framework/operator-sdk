package proxy

import (
	"context"
	"fmt"
	"strings"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const noProxyAPI = "no matches for kind \"Proxy\" in version \"config.openshift.io/v1\""

func GetProxyConfig(cfg *operator.Configuration) (*apiconfigv1.Proxy, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration can not be nil when trying to get the Proxy config")
	}

	discov, err := discovery.NewDiscoveryClientForConfig(cfg.RESTConfig)
	if err != nil {
		return nil, fmt.Errorf("encountered an error creating a discovery client: %w", err)
	}

	_, err = discov.ServerResourcesForGroupVersion(apiconfigv1.GroupVersion.Identifier())
	if err != nil {
		if isMissingResourceError(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("encountered an error getting server resources for GroupVersion `%s`: %w", apiconfigv1.GroupVersion.Identifier(), err)
	}

	proxy := &apiconfigv1.Proxy{}
	cfg.Scheme.AddKnownTypeWithName(apiconfigv1.SchemeGroupVersion.WithKind("Proxy"), proxy)

	err = cfg.Client.Get(context.Background(), client.ObjectKey{Name: "cluster"}, proxy)
	if err != nil && !strings.Contains(err.Error(), noProxyAPI) {
		return nil, err
	}

	return proxy, nil
}

func isMissingResourceError(err error) bool {
	check := "the server could not find the requested resource"

	return err.Error() == check
}
