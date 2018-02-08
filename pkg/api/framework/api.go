package framework

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

/*
'framework' asks only the user business logic to handle resource events.
It hides all nitty-gritty of k8s controller loop.

An example:
```main.go
	func main() {
		framework.RegisterSync(
			SyncCallback{
				SyncFunc:   syncEtcdCluster,
				ResourceType: &etcdapi.EtcdCluster{},
			},
		)
		framework.Run(context.TODO())
	}

	func syncEtcdCluster(obj runtime.Object, existed bool) error {
		if !existed {
			destroyEtcdCluster()
			return nil
		}
		ec := obj.(*etcdapi.EtcdCluster)
		reconcile(ec)
		return nil
	}
```
*/

// SyncCallback defines user business logic for when resource events are triggered.
type SyncCallback struct {
	// SyncFunc defines the func to sync a resource event.
	// 'obj' is the current state of the resource on the event.
	// 'existed' indicates whether this object still exists. If not, its relevant resources
	// should be destroyed.
	SyncFunc func(obj runtime.Object, existed bool) error
	// ResourceType is used to decide what resource to watch and sync events
	// It is an instantiation of the resource object which will be used to decide type at runtime.
	ResourceType runtime.Object
}

// RegisterSync registers the callbacks where user business logic is defined.
func RegisterSync(cbs ...SyncCallback) {
	panic("TODO: unimplemented")
}

// Run is the main entry point.
func Run(ctx context.Context) error {
	panic("TODO: unimplemented")
}
