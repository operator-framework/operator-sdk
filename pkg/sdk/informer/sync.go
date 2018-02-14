package informer

import (
	sdkHandler "github.com/coreos/operator-sdk/pkg/sdk/handler"
	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"

	"github.com/sirupsen/logrus"
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

// sync creates the event for the object, sends it to the handler, and processes the resulting actions
func (i *informer) sync(key string) error {
	obj, exists, err := i.sharedIndexInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}
	object := obj.(sdkTypes.Object)

	event := sdkTypes.Event{
		Object:  object,
		Deleted: !exists,
	}

	sdkCtx := sdkTypes.Context{Context: i.context}
	actions := sdkHandler.RegisteredHandler.Handle(sdkCtx, event)
	// TODO: Add option to prevent multiple informers from invoking Handle() concurrently?

	// TODO: process all actions for this event

	return nil
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
	logrus.Infof("Dropping key (%v) out of the queue: %v", key, err)
}
