package informer

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Informer interface {
	Run(ctx context.Context) error
}

type informer struct {
	resourcePluralName  string
	sharedIndexInformer cache.SharedIndexInformer
	queue               workqueue.RateLimitingInterface
	kubeClient          kubernetes.Interface
	namespace           string
	context             context.Context
}

func New(resourcePluralName, namespace string, objType runtime.Object, resourceClient rest.Interface) Informer {
	i := &informer{
		resourcePluralName: resourcePluralName,
		queue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), resourcePluralName),
		// TODO: set the kube client
		kubeClient: nil,
		namespace:  namespace,
	}

	i.sharedIndexInformer = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(resourceClient, resourcePluralName, namespace, fields.Everything()),
		objType, 0, cache.Indexers{},
	)
	i.sharedIndexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    i.handleAddResourceEvent,
		DeleteFunc: i.handleDeleteResourceEvent,
		UpdateFunc: i.handleUpdateResourceEvent,
	})
	return i
}

func (i *informer) Run(ctx context.Context) error {
	i.context = ctx
	defer i.queue.ShutDown()

	logrus.Info("starting %s controller", i.resourcePluralName)
	go i.sharedIndexInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), i.sharedIndexInformer.HasSynced) {
		return errors.New("Timed out waiting for caches to sync")
	}

	const numWorkers = 1
	for n := 0; n < numWorkers; n++ {
		go wait.Until(i.runWorker, time.Second, ctx.Done())
	}
	return nil
}

func (i *informer) handleAddResourceEvent(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	i.queue.Add(key)
}

func (i *informer) handleDeleteResourceEvent(obj interface{}) {
	// For deletes we have to use this key function
	// to handle the DeletedFinalStateUnknown case for the object
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	i.queue.Add(key)
}

func (i *informer) handleUpdateResourceEvent(oldObj, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		panic(err)
	}
	i.queue.Add(key)
}
