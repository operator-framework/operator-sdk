package manager

import (
	sdkcache "github.com/operator-framework/operator-sdk/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8smanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type manager struct {
	k8smanager.Manager
	cache  cache.Cache
	client client.Client
}

// NewManager - creating manager to handle multiple namespaces
func NewManager(mgr k8smanager.Manager, namespaces []string) (k8smanager.Manager, error) {
	c, err := sdkcache.NewMultiNamespaceCache(namespaces, mgr)
	if err != nil {
		return nil, err
	}

	cl := client.DelegatingClient{
		Reader: c,
		Writer: mgr.GetClient(),
	}

	return &manager{
		Manager: mgr,
		client:  cl,
		cache:   c,
	}, nil
}

func (m *manager) SetFields(i interface{}) error {
	if err := m.Manager.SetFields(i); err != nil {
		return err
	}

	if _, err := inject.ClientInto(m.client, i); err != nil {
		return err
	}

	if _, err := inject.CacheInto(m.cache, i); err != nil {
		return err
	}
	return nil
}

func (m *manager) GetCache() cache.Cache {
	return m.cache
}

func (m *manager) GetClient() client.Client {
	return m.client
}
