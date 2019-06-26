// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	apis "github.com/operator-framework/operator-sdk/test/test-framework/pkg/apis"
	operator "github.com/operator-framework/operator-sdk/test/test-framework/pkg/apis/cache/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestMemcached(t *testing.T) {
	memcachedList := &operator.MemcachedList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, memcachedList)
	if err != nil {
		t.Fatalf("Failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("memcached-group", func(t *testing.T) {
		t.Run("Cluster", MemcachedCluster)
		t.Run("Cluster2", MemcachedCluster)
	})
}

func memcachedScaleTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, fromReplicas, toReplicas int) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	// create memcached custom resource
	exampleMemcached := &operator.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-memcached",
			Namespace: namespace,
		},
		Spec: operator.MemcachedSpec{
			Size: int32(fromReplicas),
		},
	}
	key := types.NamespacedName{Name: exampleMemcached.GetName(), Namespace: exampleMemcached.GetNamespace()}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), exampleMemcached, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return fmt.Errorf("could not create %q: %v", key, err)
	}
	// wait for example-memcached to reach `fromReplicas` replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached", fromReplicas, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("failed waiting for %d deployment/%s replicas: %v", fromReplicas, key.Name, err)
	}

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err = f.Client.Get(goctx.TODO(), key, exampleMemcached)
		if err != nil {
			return fmt.Errorf("could not get memcached CR %q: %v", key, err)
		}

		exampleMemcached.Spec.Size = int32(toReplicas)
		t.Logf("Attempting memcached %q update, resourceVersion: %s", key, exampleMemcached.GetResourceVersion())
		return f.Client.Update(goctx.TODO(), exampleMemcached)
	})
	if err != nil {
		return fmt.Errorf("could not update memcached CR %q: %v", key, err)
	}

	// wait for example-memcached to reach `toReplicas` replicas
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, key.Namespace, key.Name, toReplicas, retryInterval, timeout); err != nil {
		return fmt.Errorf("failed waiting for %d deployment/%s replicas: %v", toReplicas, key.Name, err)
	}
	return nil
}

func MemcachedCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("Failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "memcached-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = memcachedScaleTest(t, f, ctx, 3, 4); err != nil {
		t.Fatal(err)
	}
}
