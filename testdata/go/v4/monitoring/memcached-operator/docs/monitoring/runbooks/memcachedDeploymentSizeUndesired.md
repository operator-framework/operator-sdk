# MemcachedDeploymentSizeUndesired

## Meaning
MemcachedDeploymentSizeUndesired is triggered when the number of available
<code>memcached-sample</code> replicas doesn't match the requested configuration.

## Impact
Unavailability of distributed memory object caching system in the cluster.

## Diagnosis
- Check memcached-sample's pod namespace:

  <code>export NAMESPACE="$(kubectl get deployment -A | grep memcached-sample | awk '{print $1}')"</code>

- Observe the status of the memcached-sample deployment:

  <code>kubectl get deploy memcached-sample -n $NAMESPACE -o yaml</code>

- Observe the logs of the memcached manager pod, to see why it cannot create the memcached-sample pods.

   <code>kubectl get logs <memcached-operator-controller-manager-pod> -n memcached-operator-system</code>

## Mitigation
There can be several reasons. Like:
- Node resource exhaustion
- Not enough memory on the cluster
- Nodes are down

Try to identify the root cause and fix it.