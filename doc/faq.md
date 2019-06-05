# FAQ


### I keep seeing the following intermitent warning in my Operator's logs: `The resourceVersion for the provided watch is too old.` What's wrong?

This is completely normal and expected behavior.

The `kube-apiserver` watch request handler is designed to periodicially close a watch to spread out load among controller node instances. Once disconnected, your Operator's informer will automatically reconnect and re-establish the watch. If an event is missed during re-establishment, the watch will fail with the above warning message. The Operator's informer then does a list request and uses the new `resourceVersion` from that list to restablish the watch and replace the cache with the latest objects.

This warning should not be stifled. It ensures that the informer is not stuck or wedged.

Never seeing this warning may suggest that your watch or cache is not healthy. If the message is repeating every few seconds, this may signal a network connection problem or issue with etcd.

For more information on `kube-apiserver` request timeout options, see the [Kubernetes API Server Command Line Tool Reference][kube-apiserver_options]

[kube-apiserver_options]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/#options
