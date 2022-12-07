# MemcachedOperatorDown

## Meaning
No running memcached-operator-controller-manager pods were detected in the last 5 min.

## Impact
Complete failure in the <code>Memcached</code> CR lifecycle management.
i.e. launching a new <code>Memcached</code> instance or shutting down an existing one.
## Diagnosis
- Observe the status of the memcached-operator-controller-manager deployment:

  <code>kubectl get deploy memcached-operator-controller-manager -n mecmached-operator-system -o yaml</code>

## Mitigation
There can be several reasons for the memcached-operator-controller-manager pod to be down, identify the root cause and fix it.

- Check the status of the memcached-operator-controller-manager deployment to
find out more information. The following command will provide the associated events and show if there are any issues with pulling an image, crashing pod, etc.

<code>kubectl describe deploy memcached-operator-controller-manager -n memcached-operator-system</code>

- Check if there are issues with the nodes. For example, if they are in a NotReady state.

  </code>kubectl get nodes</code>
