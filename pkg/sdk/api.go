package sdk

import (
	"context"

	sdkInformer "github.com/coreos/operator-sdk/pkg/sdk/informer"
	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

var (
	// informers is the set of all informers for the resources watched by the user
	informers []sdkInformer.Informer
)

// Watch watches for changes on the given resource.
// obj is an instance of the resource type, e.g. &Pod{}.
// resourcePluralName is the plural name of the resource, e.g. “pods”.
// resourceClient is the rest client for the resource, e.g. `kubeclient.CoreV1().RESTClient()`.
// opts provide more options for doing the watch.
// TODO: support opts for specifying label selector
func Watch(resourcePluralName, namespace string, obj runtime.Object, resourceClient rest.Interface) {
	informer := sdkInformer.New(resourcePluralName, namespace, obj, resourceClient)
	informers = append(informers, informer)
}

// TODO: func Handle(handler sdkTypes.Handler)

// Run starts the process of Watching resources, handling Events, and processing Actions
func Run(ctx context.Context) {
	sdkCtx := sdkTypes.Context{Context: ctx}
	for _, informer := range informers {
		err := informer.Run(sdkCtx)
		if err != nil {
			logrus.Errorf("failed to run informer: %v", err)
			return
		}
	}
	<-ctx.Done()
}
