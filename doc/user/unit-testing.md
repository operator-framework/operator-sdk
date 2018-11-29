# Unit testing

Testing your operator should involve both unit and [end-to-end][doc_e2e_test] tests. Unit tests assess the expected outcomes of individual operator components without requiring coordination between components. Operator unit tests should test multiple scenarios likely to be encountered by your custom operator logic at runtime. Much of your custom logic will involve API server calls via a [client][doc_client]; `Reconcile()` in particular will be making API calls on each reconciliation loop. These API calls can be mocked by using `controller-runtime`'s [fake client][doc_cr_fake_client], perfect for unit testing. This document steps through writing a unit test for the [memcached-operator][repo_memcached_reconcile]'s `Reconcile()` method using a fake client.

## Fake client

`controller-runtime`'s fake client exposes the same set of operations as a typical client, but simply tracks objects rather than sending requests over a network. You can create a new fake client that tracks an initial set of objects with the following code:

```Go
import (
  "testing"

  cachev1alpha1 "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"

  "k8s.io/apimachinery/pkg/runtime"
  "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMemcachedController(t *testing.T) {

  // A Memcached object with metadata and spec.
  memcached := &cachev1alpha1.Memcached{
    ObjectMeta: metav1.ObjectMeta{
      Name:      "memcached",
      Namespace: "memcached-operator",
    },
  }
  objs := []runtime.Object{memcached}
  cl := fake.NewFakeClient(objs...)

  ...
}

```

`cl` will cache `memcached` in an internal object tracker so that CRUD operations via `cl` can be performed on it.

## Testing Reconcile

[`Reconcile()`][doc_reconcile] performs most API server calls a particular operator controller will make. `ReconcileMemcached.Reconcile()` will ensure the `Memcached` resource exists as well as reconcile the state of owned Deployments and Pods. We can test runtime reconciliation scenarios using the above client. The following is an example that tests if `Reconcile()` creates a deployment if one is not found, and whether the created deployment is correct:

```Go
import (
  "context"
  "testing"

  cachev1alpha1 "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"

  appsv1 "k8s.io/api/apps/v1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/apimachinery/pkg/runtime"
  "k8s.io/apimachinery/pkg/types"
  "k8s.io/client-go/kubernetes/scheme"
  "sigs.k8s.io/controller-runtime/pkg/client/fake"
  "sigs.k8s.io/controller-runtime/pkg/reconcile"
  logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestMemcachedControllerDeploymentCreate(t *testing.T) {
  var (
    name            = "memcached-operator"
    namespace       = "memcached"
    replicas  int32 = 3
  )

  // A Memcached object with metadata and spec.
  memcached := &cachev1alpha1.Memcached{
    ObjectMeta: metav1.ObjectMeta{
      Name:      name,
      Namespace: namespace,
    },
    Spec: cachev1alpha1.MemcachedSpec{
      Size: replicas, // Set desired number of Memcached replicas.
    },
  }
  // Objects to track in the fake client.
  objs := []runtime.Object{
    memcached,
  }

  // Register operator types with the runtime scheme.
  s := scheme.Scheme
  s.AddKnownTypes(cachev1alpha1.SchemeGroupVersion, memcached)
  // Create a fake client to mock API calls.
  cl := fake.NewFakeClient(objs...)
  // Create a ReconcileMemcached object with the scheme and fake client.
  r := &ReconcileMemcached{client: cl, scheme: s}

  // Mock request to simulate Reconcile() being called on an event for a
  // watched resource .
  req := reconcile.Request{
    NamespacedName: types.NamespacedName{
      Name:      name,
      Namespace: namespace,
    },
  }
  res, err := r.Reconcile(req)
  if err != nil {
    t.Fatalf("reconcile: (%v)", err)
  }
  // Check the result of reconciliation to make sure it has the desired state.
  if !res.Requeue {
    t.Error("reconcile did not requeue request as expected")
  }

  // Check if deployment has been created and has the correct size.
  dep := &appsv1.Deployment{}
  err = cl.Get(context.TODO(), req.NamespacedName, dep)
  if err != nil {
    t.Fatalf("get deployment: (%v)", err)
  }
  dsize := *dep.Spec.Replicas
  if dsize != replicas {
    t.Errorf("dep size (%d) is not the expected size (%d)", dsize, replicas)
  }
}
```

The above tests if:
- `Reconcile()` fails to find a Deployment object.
- A Deployment is created.
- The request is requeued in the expected manner.
- The number of replicas in the created Deployment's spec is as expected.

A unit test checking more cases can be found in our [`samples repo`][code_test_example].

[doc_e2e_test]:https://github.com/operator-framework/operator-sdk/blob/2f772d1dc2340dd19bdc3ec8c2dc9f0f77cc8297/doc/test-framework/writing-e2e-tests.md
[doc_client]:https://github.com/operator-framework/operator-sdk/blob/5c50126e7a112d67826894997eca143e12dc165f/doc/user/client.md
[doc_cr_fake_client]:https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/client/fake
[repo_memcached_reconcile]:https://github.com/operator-framework/operator-sdk-samples/blob/4c6934448684a6953ece4d3d9f3f77494b1c125e/memcached-operator/pkg/controller/memcached/memcached_controller.go#L82
[doc_reconcile]:https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler
[code_test_example]: https://github.com/operator-framework/operator-sdk-samples/blob/master/memcached-operator/pkg/controller/memcached/memcached_controller_test.go#L25
