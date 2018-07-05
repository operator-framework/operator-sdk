// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
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
	deletedObjects      map[string]interface{}
}

func NewInformer(resourcePluralName, namespace string, resourceClient dynamic.ResourceInterface, resyncPeriod int) Informer {
	i := &informer{
		resourcePluralName: resourcePluralName,
		queue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), resourcePluralName),
		namespace:          namespace,
		deletedObjects:     map[string]interface{}{},
	}

	resyncDuration := time.Duration(resyncPeriod) * time.Second
	i.sharedIndexInformer = cache.NewSharedIndexInformer(
		newListWatcherFromResourceClient(resourceClient), &unstructured.Unstructured{}, resyncDuration, cache.Indexers{},
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

	logrus.Debugf("starting %s controller", i.resourcePluralName)
	go i.sharedIndexInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), i.sharedIndexInformer.HasSynced) {
		panic("Timed out waiting for caches to sync")
	}

	const numWorkers = 1
	for n := 0; n < numWorkers; n++ {
		go wait.Until(i.runWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
	logrus.Debugf("stopping %s controller", i.resourcePluralName)
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

	// TODO: Revisit the need for passing delete events to the handler
	// Save the last known state for the deleted object
	i.deletedObjects[key] = obj.(*unstructured.Unstructured).DeepCopy()

	i.queue.Add(key)
}

func (i *informer) handleUpdateResourceEvent(oldObj, newObj interface{}) {
	oldMeta, err := meta.Accessor(oldObj)
	if err != nil {
		panic(fmt.Errorf("object has no meta: %v", err))
	}

	newMeta, err := meta.Accessor(newObj)
	if err != nil {
		panic(fmt.Errorf("object has no meta: %v", err))
	}

	if oldMeta.GetResourceVersion() == newMeta.GetResourceVersion() {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		panic(err)
	}
	i.queue.Add(key)
}
