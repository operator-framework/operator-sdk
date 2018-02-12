package sdk

import (
	"context"
	"fmt"

	"github.com/coreos/operator-sdk/pkg/sdk/dispatcher"
	sdkInformer "github.com/coreos/operator-sdk/pkg/sdk/informer"
	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type SDK struct {
	// informers is the set of all informers registered by the user
	informers map[string]sdkInformer.Informer
	handler   sdkTypes.Handler
}

// NewKubeInformer returns an Informer for the specified resourceType
// - resourceName is the plural name of the resource kind, e.g: configmaps, secrets
// - namespace is the namespace to watch for that resource
// - objType is the runtime object to infer the type, e.g: &v1.ConfigMap{}
// - resourceClient is the REST client used to list and watch the resource
func NewKubeInformer(resourceName, namespace string, objType runtime.Object, resourceClient cache.Getter) sdkInformer.Informer {
	return sdkInformer.New(resourceName, namespace, objType, resourceClient)
}

// RegisterInformer registers an SDK informer under the specified name
func (sdk *SDK) RegisterInformer(informerName string, informer sdkInformer.Informer) error {
	if _, ok := sdk.informers[informerName]; ok {
		return fmt.Errorf("informer (%v) is already registered", informerName)
	}
	sdk.informers[informerName] = informer
	return nil
}

func (sdk *SDK) RegisterHandler(handler sdkTypes.Handler) {
	sdk.handler = handler
}

func (sdk *SDK) Run(ctx context.Context) {
	// Run all informers and get the event channels
	var eventChans []<-chan *sdkTypes.Event
	for _, informer := range sdk.informers {
		evc, err := informer.Run(ctx)
		if err != nil {
			panic("TODO")
		}
		eventChans = append(eventChans, evc)
	}

	// Create a new dispatcher to pass events to the registered handler
	dp := dispatcher.New(eventChans, sdk.handler)
	dp.Run(ctx)
}

/*
```main.go
func main() {
	sdk.RegisterInformer("play-informer", informer.NewKubeInformer(&Play{}))
  	sdk.RegisterInformer("pod-informer", informer.NewKubeInformer(&Pod{}))

  	sdk.RegisterActor("kube-apply", actor.KubeResourceApply)
  	sdk.RegisterActor("kube-delete", actor.KubeResourceDelete)

  	sdk.RegisterHandle(stub.NewHandler())
	sdk.Run(ctx)
}
*/
