package sdk

import (
	"context"

	"github.com/coreos/operator-sdk/pkg/k8sclient"
	sdkHandler "github.com/coreos/operator-sdk/pkg/sdk/handler"
	sdkInformer "github.com/coreos/operator-sdk/pkg/sdk/informer"

	"github.com/sirupsen/logrus"
)

var (
	// informers is the set of all informers for the resources watched by the user
	informers []sdkInformer.Informer
)

// Watch watches for changes on the given resource.
// apiVersion for a resource is of the format "Group/Version" except for the "Core" group whose APIVersion is just "v1". For e.g:
//   - Deployments have Group "apps" and Version "v1beta2" giving the APIVersion "apps/v1beta2"
//   - Pods have Group "Core" and Version "v1" giving the APIVersion "v1"
//   - The custom resource Memcached might have Group "cache.example.com" and Version "v1alpha1" giving the APIVersion "cache.example.com/v1alpha1"
// kind is the Kind of the resource, e.g "Pod" for pods
// Consult the API reference for the Group, Version and Kind of a resource: https://kubernetes.io/docs/reference/
// namespace is the Namespace to watch for the resource
// TODO: support opts for specifying label selector
func Watch(apiVersion, kind, namespace string) {
	resourceClient, resourcePluralName, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	// TODO: Better error handling, e.g retry
	if err != nil {
		logrus.Errorf("failed to get resource client for (apiVersion:%s, kind:%s, ns:%s): %v", apiVersion, kind, namespace, err)
		panic(err)
	}
	informer := sdkInformer.New(resourcePluralName, namespace, resourceClient)
	informers = append(informers, informer)
}

// Handle registers the handler for all events.
// In the future, we would have a mux-pattern to dispatch events to matched handlers.
func Handle(handler sdkHandler.Handler) {
	sdkHandler.RegisteredHandler = handler
}

// Run starts the process of Watching resources, handling Events, and processing Actions
func Run(ctx context.Context) {
	for _, informer := range informers {
		go informer.Run(ctx)
	}
	<-ctx.Done()
}
