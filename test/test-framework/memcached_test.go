package e2e

import (
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/util/e2eutil"
)

func TestMemcached(t *testing.T) {
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	err := ctx.InitializeClusterResources(t)
	if err != nil {
		t.Fatalf("Failed to initialize clister resources: %v", err)
	}
	t.Log("Initialized cluster resources")

	// run subtests
	t.Run("memcached-group", func(t *testing.T) {
		t.Run("Scale", MemcachedScale)
		t.Run("PodFail", MemcachedPodFail)
	})
}

func MemcachedScale(t *testing.T) {
	t.Parallel()
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)

	// create memcached custom resource
	crYAML := []byte("apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached-scale\"\nspec:\n  size: 3")
	err := ctx.CreateFromYAML(crYAML)
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// wait for example-memcached-scale to reach 3 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached-scale", 3, 6)
	if err != nil {
		t.Fatal(err)
	}

	err = ctx.UpdateCR("example-memcached-scale", "memcacheds", "/spec/size", "4")
	if err != nil {
		t.Fatal(err)
	}

	// wait for example-memcached-scale to reach 4 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached-scale", 4, 6)
	if err != nil {
		t.Fatal(err)
	}
}

func MemcachedPodFail(t *testing.T) {
	t.Parallel()
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)

	// create memcached custom resource
	crYAML := []byte("apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached-podfail\"\nspec:\n  size: 3")
	err := ctx.CreateFromYAML(crYAML)
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// wait for example-memcached-podfail to reach 3 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached-podfail", 3, 6)
	if err != nil {
		t.Fatal(err)
	}

	err = ctx.SimulatePodFailure("example-memcached-podfail")
	if err != nil {
		t.Fatal(err)
	}
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached-podfail", 3, 6)
	if err != nil {
		t.Fatal(err)
	}
}
