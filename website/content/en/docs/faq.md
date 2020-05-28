---
title: Operator SDK FAQ
linkTitle: FAQ
weight: 80
---

## Controller Runtime FAQ

Please see the upstream [Controller Runtime FAQ][cr-faq] first for any questions related to runtime mechanics or controller-runtime APIs.

## How can I have separate logic for Create, Update, and Delete events? When reconciling an object can I access its previous state?

You should not have separate logic. Instead design your reconciler to be idempotent. See the [controller-runtime FAQ][controller-runtime_faq] for more details.

## When my Custom Resource is deleted, I need to know its contents or perform cleanup tasks. How can I do that?

Use a [finalizer].

## I keep seeing the following intermittent warning in my Operator's logs: `The resourceVersion for the provided watch is too old.` What's wrong?

This is completely normal and expected behavior.

The `kube-apiserver` watch request handler is designed to periodically close a watch to spread out load among controller node instances. Once disconnected, your Operator's informer will automatically reconnect and re-establish the watch. If an event is missed during re-establishment, the watch will fail with the above warning message. The Operator's informer then does a list request and uses the new `resourceVersion` from that list to restablish the watch and replace the cache with the latest objects.

This warning should not be stifled. It ensures that the informer is not stuck or wedged.

Never seeing this warning may suggest that your watch or cache is not healthy. If the message is repeating every few seconds, this may signal a network connection problem or issue with etcd.

For more information on `kube-apiserver` request timeout options, see the [Kubernetes API Server Command Line Tool Reference][kube-apiserver_options]


## I keep seeing errors like "Failed to create metrics Service", how do I fix this?

If you run into the following error message:

```
time="2019-06-05T12:29:54Z" level=fatal msg="failed to create or get service for metrics: services \"my-operator\" is forbidden: cannot set blockOwnerDeletion if an ownerReference refers to a resource you can't set finalizers on: , <nil>"
```

Add the following to your `deploy/role.yaml` file to grant the operator permissions to set owner references to the metrics Service resource. This is needed so that the metrics Service will get deleted as soon as you delete the operators Deployment. If you are using another way of deploying your operator, have a look at [this guide][gc-metrics] for more information.

```
- apiGroups:
  - apps
  resources:
  - deployments/finalizers
  resourceNames:
  - <operator-name>
  verbs:
  - "update"
```

## My Ansible module is missing a dependency. How do I add it to the image?

Unfortunately, adding the entire dependency tree for all Ansible modules would be excessive. Fortunately, you can add it easily. Simply edit your build/Dockerfile. You'll want to change to root for the install command, just be sure to swap back using a series of commands like the following right after the `FROM` line.

```
USER 0
RUN yum -y install my-dependency
RUN pip3 install my-python-dependency
USER 1001
```

If you aren't sure what dependencies are required, start up a container using the image in the `FROM` line as root. That will look something like this.
`docker run -u 0 -it --rm --entrypoint /bin/bash quay.io/operator-framework/ansible-operator:<sdk-tag-version>`

## I keep seeing errors like "Failed to watch", how do I fix this?

If you run into the following error message:

```
E0320 15:42:17.676888       1 reflector.go:280] pkg/mod/k8s.io/client-go@v0.0.0-20191016111102-bec269661e48/tools/cache/reflector.go:96: Failed to watch *v1.ImageStreamTag: unknown (get imagestreamtags.image.openshift.io)
{"level":"info","ts":1584718937.766342,"logger":"controller_memcached","msg":"ImageStreamTag resource not found.
```

Then, it means that your Operator is unable to watch the resource. This scenario can be faced because the Operator does not have the permission [(RBAC)[rbac]] to `Watch` the resource, or may be the Schema from the API used, did not implement this verb. In this way the solution would be to grant the permission in the `role.yaml` , or when it is not  possible, use the [client.Reader][client.Reader] instead of the client provided.

The client provided will work with a cache, and because of this, the `WATCH` verb is required.   

**Example**

Following are the changes in the `conttroler.go`,  to address the need to get the resource via the [client.Reader][client.Reader]. See:

```go

import (
	...
	imagev1 "github.com/openshift/api/image/v1"
)

...

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMemcached{client: mgr.GetClient(), scheme: mgr.GetScheme(), APIReader: mgr.GetAPIReader() }

}

...

// ReconcileMemcached reconciles a Memcached object
type ReconcileMemcached struct {
	// TODO: Clarify the split client
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	APIReader client.Reader // the APIReader  will not use the cache
}

...
func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Get the ImageStreamTag from OCP API which has not the WATCH verb.
	img := &imagev1.ImageStreamTag{}
	err = r.APIReader.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s:%s", "example-name", "example-tag"), img)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("resource not found")
		} else {
			reqLogger.Error(err, "unexpected error")
		}
	}
```

## I see deepcopy errors and image build fails. How do I fix this?

When you run the ```operator-sdk generate k8s``` command, you might see an error like this

```
INFO[0000] Running deepcopy code-generation for Custom Resource group versions: [cache:[v1alpha1], ] 
F0523 01:18:27.122034    5157 deepcopy.go:885] Hit an unsupported type invalid type for invalid type, from github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1.Memcached
```

This is because of the `GOROOT` environment variable not being set. More details [here][goroot-github-issue].

In order to fix this, you simply need to export the `GOROOT` environment variable

```
$ export GOROOT=$(go env GOROOT)
```

This will work for the current environment. To persist this fix, add the above line to your environment's config file, ex. `bashrc` file.

[kube-apiserver_options]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/#options
[controller-runtime_faq]: https://github.com/kubernetes-sigs/controller-runtime/blob/master/FAQ.md#q-how-do-i-have-different-logic-in-my-reconciler-for-different-types-of-events-eg-create-update-delete
[finalizer]: /docs/golang/quickstart/#handle-cleanup-on-deletion
[gc-metrics]:/docs/golang/monitoring/prometheus/#garbage-collection
[cr-faq]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/FAQ.md
[client.Reader]:https://godoc.org/sigs.k8s.io/controller-runtime/pkg/client#Reader
[rbac]:https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[goroot-github-issue]:https://github.com/operator-framework/operator-sdk/issues/1854#issuecomment-525132306
