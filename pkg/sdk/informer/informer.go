package informer

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Informer interface {
	Run(ctx context.Context)
}

type informer struct {
	resourcePluralName  string
	sharedIndexInformer cache.SharedIndexInformer
	queue               workqueue.RateLimitingInterface
	namespace           string
	context             context.Context
}

func New(resourcePluralName, namespace string, resourceClient dynamic.ResourceInterface) Informer {
	i := &informer{
		resourcePluralName: resourcePluralName,
		queue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), resourcePluralName),
		namespace:          namespace,
	}

	i.sharedIndexInformer = cache.NewSharedIndexInformer(
		newListWatcherFromResourceClient(resourceClient), &unstructured.Unstructured{}, 0, cache.Indexers{},
	)
	i.sharedIndexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    i.handleAddResourceEvent,
		DeleteFunc: i.handleDeleteResourceEvent,
		UpdateFunc: i.handleUpdateResourceEvent,
	})
	return i
}

func newListWatcherFromResourceClient(resourceClient dynamic.ResourceInterface) *cache.ListWatch {
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		return resourceClient.List(options)
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		return resourceClient.Watch(options)
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (i *informer) Run(ctx context.Context) {
	i.context = ctx
	defer i.queue.ShutDown()

	logrus.Infof("starting %s controller", i.resourcePluralName)
	go i.sharedIndexInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), i.sharedIndexInformer.HasSynced) {
		panic("Timed out waiting for caches to sync")
	}

	const numWorkers = 1
	for n := 0; n < numWorkers; n++ {
		go wait.Until(i.runWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
	logrus.Infof("stopping %s controller", i.resourcePluralName)
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
