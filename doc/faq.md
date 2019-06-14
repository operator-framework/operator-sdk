# FAQ

### How can I have separate logic for Create, Update, and Delete events? When reconciling an object can I access its previous state?

You should not have separate logic. Instead design your reconciler to be idempotent. See the [controller-runtime FAQ][controller-runtime_faq] for more details.

### When my Custom Resource is deleted, I need to know it's contents or perform cleanup tasks. How can I do that?

Use a [finalizer].

### I keep seeing the following intermitent warning in my Operator's logs: `The resourceVersion for the provided watch is too old.` What's wrong?

This is completely normal and expected behavior.

The `kube-apiserver` watch request handler is designed to periodicially close a watch to spread out load among controller node instances. Once disconnected, your Operator's informer will automatically reconnect and re-establish the watch. If an event is missed during re-establishment, the watch will fail with the above warning message. The Operator's informer then does a list request and uses the new `resourceVersion` from that list to restablish the watch and replace the cache with the latest objects.

This warning should not be stifled. It ensures that the informer is not stuck or wedged.

Never seeing this warning may suggest that your watch or cache is not healthy. If the message is repeating every few seconds, this may signal a network connection problem or issue with etcd.

For more information on `kube-apiserver` request timeout options, see the [Kubernetes API Server Command Line Tool Reference][kube-apiserver_options]



[kube-apiserver_options]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/#options
[controller-runtime_faq]: https://github.com/kubernetes-sigs/controller-runtime/blob/master/FAQ.md#q-how-do-i-have-different-logic-in-my-reconciler-for-different-types-of-events-eg-create-update-delete
[finalizer]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#handle-cleanup-on-deletion
