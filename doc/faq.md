# FAQ

### How can I have separate logic for Create, Update, and Delete events? When reconciling an object can I access its previous state?

You should not have separate logic. Instead design your reconciler to be idempotent. See the [controller-runtime FAQ][controller-runtime_faq] for more details.

### When my Custom Resource is deleted, I need to know it's contents or perform cleanup tasks. How can I do that?

Use a [finalizer].

### I keep seeing the following intermittent warning in my Operator's logs: `The resourceVersion for the provided watch is too old.` What's wrong?

This is completely normal and expected behavior.

The `kube-apiserver` watch request handler is designed to periodically close a watch to spread out load among controller node instances. Once disconnected, your Operator's informer will automatically reconnect and re-establish the watch. If an event is missed during re-establishment, the watch will fail with the above warning message. The Operator's informer then does a list request and uses the new `resourceVersion` from that list to restablish the watch and replace the cache with the latest objects.

This warning should not be stifled. It ensures that the informer is not stuck or wedged.

Never seeing this warning may suggest that your watch or cache is not healthy. If the message is repeating every few seconds, this may signal a network connection problem or issue with etcd.

For more information on `kube-apiserver` request timeout options, see the [Kubernetes API Server Command Line Tool Reference][kube-apiserver_options]


### I keep seeing errors like "Failed to create metrics Service", how do I fix this?

If you run into the following error message:

```
time="2019-06-05T12:29:54Z" level=fatal msg="failed to create or get service for metrics: services \"my-operator\" is forbidden: cannot set blockOwnerDeletion if an ownerReference refers to a resource you can't set finalizers on: , <nil>"
```

Add the following to your `deploy/role.yaml` file to grant the operator permissions to set owner references to the metrics Service resource. This is needed so that the metrics Service will get deleted as soon as you delete the operators Deployment. If you are using another way of deploying your operator, have a look at [this guide][gc-metrics] for more information.

```
  resources:
  - deployments/finalizers
```


[kube-apiserver_options]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/#options
[controller-runtime_faq]: https://github.com/kubernetes-sigs/controller-runtime/blob/master/FAQ.md#q-how-do-i-have-different-logic-in-my-reconciler-for-different-types-of-events-eg-create-update-delete
[finalizer]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#handle-cleanup-on-deletion
[gc-metrics]:./user/metrics/README.md#garbage-collection
