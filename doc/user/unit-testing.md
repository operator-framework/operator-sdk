# Unit testing
------------

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  <!-- *generated with [DocToc](https://github.com/thlorenz/doctoc)*  -->

- [Overview](#overview)
- [Using a Fake client](#using-a-fake-client)
- [Testing Reconcile](#testing-reconcile)
- [Testing with 3rd Party Resources](#testing-with-3rd-party-resources)
- [How to increase the verbosity of the logs?](#how-to-increase-the-verbosity-of-the-logs)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

Testing your operator should involve both unit and [end-to-end][doc-e2e-test] tests. Unit tests assess the expected outcomes of individual operator components without requiring coordination between components. Operator unit tests should test multiple scenarios likely to be encountered by your custom operator logic at runtime. Much of your custom logic will involve API server calls via a [client][doc-client]; `Reconcile()` in particular will be making API calls on each reconciliation loop. These API calls can be mocked by using `controller-runtime`'s [fake client][doc-cr-fake-client], perfect for unit testing. This document steps through writing a unit test for the [memcached-operator][repo-memcached-reconcile]'s `Reconcile()` method using a fake client.

## Using a Fake client

The `controller-runtime`'s fake client exposes the same set of operations as a typical client, but simply tracks objects rather than sending requests over a network. You can create a new fake client that tracks an initial set of objects with the following code:

```Go
import (
    "context"
    "testing"

    cachev1alpha1 "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"
    "k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMemcachedController(t *testing.T) {
    ...
    // A Memcached object with metadata and spec.
    memcached := &cachev1alpha1.Memcached{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "memcached",
            Namespace: "memcached-operator",
            Labels: map[string]string{
                "label-key": "label-value",
            },
        },
    }

    // Objects to track in the fake client.
    objs := []runtime.Object{memcached}

    // Create a fake client to mock API calls.
    cl := fake.NewFakeClient(objs...)

    // List Memcached objects filtering by labels
    memcachedList := &cachev1alpha1.MemcachedList{}
    err := cl.List(context.TODO(), client.MatchingLabels(map[string]string{
		"label-key": "label-value",
    }), memcachedList)
    if err != nil {
        t.Fatalf("list memcached: (%v)", err)
    }
    ...
}
```
The fake client `cl` will cache `memcached` in an internal object tracker so that CRUD operations via `cl` can be performed on it.

## Testing Reconcile

[`Reconcile()`][doc-reconcile] performs most API server calls a particular operator controller will make. `ReconcileMemcached.Reconcile()` will ensure the `Memcached` resource exists as well as reconcile the state of owned Deployments and Pods. We can test runtime reconciliation scenarios using the above client. The following is an example that tests if `Reconcile()` creates a deployment if one is not found, and whether the created deployment is correct:

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
    objs := []runtime.Object{ memcached }

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
    err = r.client.Get(context.TODO(), req.NamespacedName, dep)
    if err != nil {
        t.Fatalf("get deployment: (%v)", err)
    }
    // Check if the quantity of Replicas for this deployment is equals the specification
    dsize := *dep.Spec.Replicas
    if dsize != replicas {
        t.Errorf("dep size (%d) is not the expected size (%d)", dsize, replicas)
    }
}
```

**The above tests check if:**

- `Reconcile()` fails to find a Deployment object
- A Deployment is created
- The request is requeued in the expected manner
- The number of replicas in the created Deployment's spec is as expected.

**NOTE**: A unit test checking more cases can be found in our [`samples repo`][code-test-example].

## Testing with 3rd Party Resources

You may have added third-party resources in your operator as described in the [`Advanced Topics section of the user guide`][user-guide]. In order to create a unit-test to test these kinds of resources, it might be necessary to update the Scheme with the third-party resources and pass it to your Reconciler.
The following code snippet is an example that adds the [`v1.Route`][ocp-doc-v1-route] OpenShift scheme to the ReconcileMemcached reconciler's scheme.

```go

import (
    ...
    routev1 "github.com/openshift/api/route/v1"
    ...
)

// TestMemcachedController runs ReconcileMemcached.Reconcile() against a
// fake client that tracks a Memcached object.
func TestMemcachedController(t *testing.T) {
    ...
    // Register operator types with the runtime scheme.
    s := scheme.Scheme

    // Add route Openshift scheme
    if err := routev1.AddToScheme(s); err != nil {
        t.Fatalf("Unable to add route scheme: (%v)", err)
    }

    // Create the mock for the Route
    // NOTE: If the object will be created by the reconcile you do not need add a mock for it
    route := &routev1.Route{
        ObjectMeta: v1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
            Labels:    getAppLabels(name),
        },
    }

    s.AddKnownTypes(appv1alpha1.SchemeGroupVersion, memcached)

    // Create a fake client to mock API calls.
    cl := fake.NewFakeClient(objs...)

    // Create a ReconcileMemcached object with the scheme and fake client.
    r := &ReconcileMemcached{client: cl, scheme: s}
    ...
}
```

**NOTE:** If your Reconcile has not the scheme attribute you may create the client fake as `cl := fake.NewFakeClientWithScheme(s, objs...)` in order to add the schema.

In this way, you will be able to get the mock object injected into the Reconcile as the following example.

```go
    route := &routev1.Route{}
    err = r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, route)
    if err != nil {
        t.Fatalf("get route: (%v)", err)
    }
```

**NOTE:** Following an example of issue that can be faced because of an invalid `TypeMeta.APIVersion` informed. It is not recommended declared the `TypeMeta` since it will be implicit generated.

```shell
get route: (no kind "Route" is registered for version "v1" in scheme "k8s.io/client-go/kubernetes/scheme/register.go:61")
```

Following an example which could cause this error.

```go
    ...
    route := &routev1.Route{
        TypeMeta: v1.TypeMeta{  // TODO (user): Remove the TypeMeta declared
            APIVersion: "v1",   // the correct value will be `"route.openshift.io/v1"`
            Kind:       "Route",
        },
        ObjectMeta: v1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
            Labels:    ls,
        },
    }
    ...
```

Following another example of the issue that can be faced when the third-party resource schema was not added properly.

```shell
create a route: (no kind is registered for the type v1.Route in scheme "k8s.io/client-go/kubernetes/scheme/register.go:61")`
```

## How to increase the verbosity of the logs?

Following is a snippet code as an example to increase the verbosity of the logs in order to better troubleshoot your tests.

```go
import (
    ...
    logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
    ...
)
func TestMemcachedController(t *testing.T) {
    //dev logs
    logf.SetLogger(logf.ZapLogger(true))
    ...
}
```

<!-- Link vars declaration -->
<!-- NOTE: The CI has a bug and the test will not pass with _ . Use "-" instead of "_" -->

[doc-e2e-test]: ../test-framework/writing-e2e-tests.md
[doc-client]: client.md
[doc-cr-fake-client]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/client/fake
[repo-memcached-reconcile]: https://github.com/operator-framework/operator-sdk-samples/blob/4c6934448684a6953ece4d3d9f3f77494b1c125e/memcached-operator/pkg/controller/memcached/memcached_controller.go#L82
[doc-reconcile]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Reconciler
[code-test-example]: https://github.com/operator-framework/operator-sdk-samples/blob/master/memcached-operator/pkg/controller/memcached/memcached_controller_test.go#L25
[user-guide]: ../user-guide.md#register-with-the-managers-scheme
[ocp-doc-v1-route]: https://docs.openshift.com/container-platform/3.11/rest_api/apis-route.openshift.io/v1.Route.html
