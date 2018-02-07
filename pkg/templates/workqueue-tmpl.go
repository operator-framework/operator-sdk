package templates

// TODO: fix imports

const ControllerTemplate = `
package controller

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
	}
}

// sync is the business logic of the controller.
// In case an error happened, it has to simply return the error and will be retried after a backoff.
func (c *Controller) sync(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		return fmt.Errorf("Fetching object with key %s from store failed with %v", key, err)
	}

	if !exists {
		// We warmed up the cache, so this could only imply the object was deleted
		return nil
	}
	
	return nil
}
`

const WorkqueueTemplate = `
package controller

const (
	// maxRetries is the number of times an event will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times of requeues:
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.sync(key.(string))

	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries maxRetries times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < maxRetries {
		glog.Infof("Error syncing pod %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

// numWorkers is the number of goroutine workers to process events concurrently.
func (c *Controller) Run(ctx context.Context, numWorkers int) {
	// Let the workers stop when we are done
	defer c.queue.ShutDown()

	go c.informer.Run(stopCh)

	// Wait for the cache to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-ctx.Done()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
`
