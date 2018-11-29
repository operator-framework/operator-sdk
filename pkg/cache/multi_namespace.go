package cache

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// MultiNamespaceCache - allows for using multiple namespaces for the cache.
type multiNamespaceCache struct {
	caches map[string]cache.Cache
}

// NewMultiNamespaceCache - creates a new cache that can handle multiple namespaces
func NewMultiNamespaceCache(namespaces []string, mgr manager.Manager) (cache.Cache, error) {
	m := multiNamespaceCache{
		caches: map[string]cache.Cache{},
	}
	for _, namespace := range namespaces {
		o := cache.Options{
			Scheme:    mgr.GetScheme(),
			Mapper:    mgr.GetRESTMapper(),
			Namespace: namespace,
		}

		cache, err := cache.New(mgr.GetConfig(), o)
		if err != nil {
			return nil, err
		}
		m.caches[namespace] = cache
	}
	return &m, nil
}

var _ cache.Cache = &multiNamespaceCache{}

// Get - Gets the object from the cache based on the key
func (m *multiNamespaceCache) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	c, ok := m.caches[key.Namespace]
	if !ok {
		return fmt.Errorf("unknown namespace for cache: %v", key.Namespace)
	}
	return c.Get(ctx, key, obj)
}

// List - Gets the list from the cache.
func (m *multiNamespaceCache) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
	// TODO: we may want to assume that empty namespace means seach in all namespaces I am watching.
	c, ok := m.caches[opts.Namespace]
	if !ok {
		return fmt.Errorf("unknown namespace for cache: %v", opts.Namespace)
	}
	return c.List(ctx, opts, list)
}

// GetInformer - get an informer for an obj
func (m *multiNamespaceCache) GetInformer(obj runtime.Object) (toolscache.SharedIndexInformer, error) {
	// TODO: we need to create a way to return a shared index informer that deals with multiple namespaces.
	// TODO: This could just create a new index informer but use a multiNamespaced list watcher.
	return nil, nil
}

// GetInformerForKind - get an informer for an kind
func (m *multiNamespaceCache) GetInformerForKind(gvk schema.GroupVersionKind) (toolscache.SharedIndexInformer, error) {
	// TODO: we need to create a way to return a shared index informer that deals with multiple namespaces.
	// TODO: This could just create a new index informer but use a multiNamespaced list watcher.
	return nil, nil
}

// Start - starts all the underlying caches
func (m *multiNamespaceCache) Start(stopCh <-chan struct{}) error {
	for _, c := range m.caches {
		go c.Start(stopCh)
	}
	<-stopCh
	return nil
}

// WaitForCacheSync - waits for all the underlying caches to sync
func (m *multiNamespaceCache) WaitForCacheSync(stop <-chan struct{}) bool {
	for _, c := range m.caches {
		ok := c.WaitForCacheSync(stop)
		if !ok {
			return ok
		}
	}
	return true
}

// IndexField - adds indexer to all of the underlying caches
func (m *multiNamespaceCache) IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error {
	for _, c := range m.caches {
		err := c.IndexField(obj, field, extractValue)
		if err != nil {
			return err
		}
	}
	return nil
}
