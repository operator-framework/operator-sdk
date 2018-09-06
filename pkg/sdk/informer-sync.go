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
	"github.com/operator-framework/operator-sdk/pkg/sdk/internal/metrics"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// Copy from deployment_controller.go:
	// maxRetries is the number of times a Vault will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a Vault is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

func (i *informer) runWorker() {
	for i.processNextItem() {
	}
}

func (i *informer) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := i.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer i.queue.Done(key)

	// Invoke the method containing the business logic
	err := i.sync(key.(string))

	// Handle the error if something went wrong during the execution of the business logic
	i.handleErr(err, key)
	return true
}

// sync creates the event for the object and sends it to the handler
func (i *informer) sync(key string) error {
	obj, exists, err := i.sharedIndexInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		logrus.Debugf("Object (%s) is deleted", key)
		// Lookup the last saved state for the deleted object
		_, ok := i.deletedObjects[key]
		if !ok {
			logrus.Errorf("no last known state found for deleted object (%s)", key)
			return nil
		}
		obj = i.deletedObjects[key]
	}

	unstructObj := obj.(*unstructured.Unstructured).DeepCopy()
	object, err := k8sutil.RuntimeObjectFromUnstructured(unstructObj)
	if err != nil {
		return err
	}

	event := Event{
		Object:  object,
		Deleted: !exists,
	}

	// TODO: Add option to prevent multiple informers from invoking Handle() concurrently?
	err = RegisteredHandler.Handle(i.context, event)
	if !exists && err == nil {
		delete(i.deletedObjects, key)
	}
	switch {
	case err == nil:
		i.collector.ReconcileResult.WithLabelValues(metrics.ReconcileResultSuccess).Inc()
	case err != nil:
		i.collector.ReconcileResult.WithLabelValues(metrics.ReconcileResultFailure).Inc()
	}
	return err
}

// handleErr checks if an error happened and makes sure we will retry later.
func (i *informer) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		i.queue.Forget(key)
		return
	}

	// This controller retries maxRetries times if something goes wrong. After that, it stops trying.
	if i.queue.NumRequeues(key) < maxRetries {
		logrus.Errorf("error syncing key (%v): %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		i.queue.AddRateLimited(key)
		return
	}

	i.queue.Forget(key)
	// Report that, even after several retries, we could not successfully process this key
	logrus.Warnf("Dropping key (%v) out of the queue: %v", key, err)
}
