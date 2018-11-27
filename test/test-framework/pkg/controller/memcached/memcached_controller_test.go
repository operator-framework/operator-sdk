package memcached

import (
	"context"
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	cachev1alpha1 "github.com/operator-framework/operator-sdk/test/test-framework/pkg/apis/cache/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TestMemcachedController runs ReconcileMemcached.Reconcile() against a
// fake client that tracks a Memcached object.
func TestMemcachedController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name            = "memcached-operator"
		namespace       = "memcached"
		replicas  int32 = 3
	)

	// A Memcached resource with metadata and spec.
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

	// Create the 3 expected pods in namespace and collect their names to check
	// later.
	podLabels := labelsForMemcached(name)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Labels:    podLabels,
		},
	}
	podNames := make([]string, 3)
	for i := 0; i < 3; i++ {
		pod.ObjectMeta.Name = name + ".pod." + strconv.Itoa(rand.Int())
		podNames[i] = pod.ObjectMeta.Name
		if err = cl.Create(context.TODO(), pod.DeepCopy()); err != nil {
			t.Fatalf("create pod %d: (%v)", i, err)
		}
	}

	// Reconcile again so Reconcile() checks pods and updates the Memcached
	// resources' Status.
	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if res != (reconcile.Result{}) {
		t.Error("reconcile did not return an empty Result")
	}

	// Get the updated Memcached object.
	memcached = &cachev1alpha1.Memcached{}
	err = r.client.Get(context.TODO(), req.NamespacedName, memcached)
	if err != nil {
		t.Errorf("get memcached: (%v)", err)
	}

	// Ensure Reconcile() updated the Memcached's Status as expected.
	nodes := memcached.Status.Nodes
	if !reflect.DeepEqual(podNames, nodes) {
		t.Errorf("pod names %v did not match expected %v", nodes, podNames)
	}
}
