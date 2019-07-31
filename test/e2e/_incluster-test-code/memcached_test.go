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
	"bytes"
	goctx "context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	apis "github.com/example-inc/memcached-operator/pkg/apis"
	operator "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/util/promlint"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 120
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 10
	operatorName         = "memcached-operator"
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
		t.Run("Local", MemcachedLocal)
	})
}

func memcachedLeaderTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, operatorName, 2, retryInterval, timeout)
	if err != nil {
		return err
	}

	label := map[string]string{"name": operatorName}

	leader, err := verifyLeader(t, namespace, f, label)
	if err != nil {
		return err
	}

	// delete the leader's pod so a new leader will get elected
	err = f.Client.Delete(goctx.TODO(), leader)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeletion(t, f.Client.Client, leader, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, operatorName, 2, retryInterval, timeout)
	if err != nil {
		return err
	}

	newLeader, err := verifyLeader(t, namespace, f, label)
	if err != nil {
		return err
	}
	if newLeader.Name == leader.Name {
		return fmt.Errorf("leader pod name did not change across pod delete")
	}

	return nil
}

func verifyLeader(t *testing.T, namespace string, f *framework.Framework, labels map[string]string) (*v1.Pod, error) {
	// get configmap, which is the lock
	lockName := "memcached-operator-lock"
	lock := v1.ConfigMap{}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: lockName, Namespace: namespace}, &lock)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of leader lock configmap %s\n", lockName)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error getting leader lock configmap: %v", err)
	}
	t.Logf("Found leader lock configmap %s\n", lockName)

	owners := lock.GetOwnerReferences()
	if len(owners) != 1 {
		return nil, fmt.Errorf("leader lock has %d owner refs, expected 1", len(owners))
	}
	owner := owners[0]

	// get operator pods
	pods := v1.PodList{}
	opts := client.ListOptions{Namespace: namespace}
	for k, v := range labels {
		if err := opts.SetLabelSelector(fmt.Sprintf("%s=%s", k, v)); err != nil {
			return nil, fmt.Errorf("failed to set list label selector: (%v)", err)
		}
	}
	if err := opts.SetFieldSelector("status.phase=Running"); err != nil {
		t.Fatalf("Failed to set list field selector: (%v)", err)
	}
	err = f.Client.List(goctx.TODO(), &opts, &pods)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) != 2 {
		return nil, fmt.Errorf("expected 2 pods, found %d", len(pods.Items))
	}

	// find and return the leader
	for _, pod := range pods.Items {
		if pod.Name == owner.Name {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("did not find operator pod that was referenced by configmap")
}

func memcachedScaleTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, fromReplicas, toReplicas int) error {
	name := "example-memcached"
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	// create memcached custom resource
	exampleMemcached := &operator.Memcached{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: operator.MemcachedSpec{
			Size: int32(fromReplicas),
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), exampleMemcached, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return fmt.Errorf("could no create CR: %v", err)
	}
	// wait for example-memcached to reach `fromReplicas` replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, key.Namespace, key.Name, fromReplicas, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("failed waiting for %d deployment/%s replicas: %v", fromReplicas, key.Name, err)
	}

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err = f.Client.Get(goctx.TODO(), key, exampleMemcached)
		if err != nil {
			return fmt.Errorf("could not get memcached CR %q: %v", key, err)
		}
		// update memcached CR size to `toReplicas` replicas
		exampleMemcached.Spec.Size = int32(toReplicas)
		t.Logf("Attempting memcached CR %q update, resourceVersion: %s", key, exampleMemcached.GetResourceVersion())
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

func MemcachedLocal(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("operator-sdk", "up", "local", "--namespace="+namespace)
	stderr, err := os.Create("stderr.txt")
	if err != nil {
		t.Fatalf("Failed to create stderr.txt: %v", err)
	}
	cmd.Stderr = stderr
	defer func() {
		if err := stderr.Close(); err != nil {
			t.Errorf("Failed to close stderr: (%v)", err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	ctx.AddCleanupFn(func() error { return cmd.Process.Signal(os.Interrupt) })

	// wait for operator to start (may take a minute to compile the command...)
	err = wait.Poll(time.Second*5, time.Second*100, func() (done bool, err error) {
		file, err := ioutil.ReadFile("stderr.txt")
		if err != nil {
			return false, err
		}
		if len(file) == 0 {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Local operator not ready after 100 seconds: %v\n", err)
	}

	if err = memcachedScaleTest(t, framework.Global, ctx, 3, 4); err != nil {
		file, fileErr := ioutil.ReadFile("stderr.txt")
		if fileErr != nil {
			t.Logf("Failed to read operator logs after test failure: %v", fileErr)
		} else {
			t.Logf("Operator Logs: %s", string(file))
		}
		t.Fatal(err)
	}
}

func MemcachedCluster(t *testing.T) {
	// get global framework variables
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
	if err := e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, operatorName, 2, retryInterval, timeout); err != nil {
		t.Fatal(err)
	}

	if err := memcachedLeaderTest(t, f, ctx); err != nil {
		t.Error(err)
	}
	t.Log("Completed leader test")

	if err := memcachedScaleTest(t, f, ctx, 3, 4); err != nil {
		t.Error(err)
	}
	t.Log("Completed scale test")

	if err := memcachedMetricsTest(t, f, ctx); err != nil {
		t.Error(err)
	}
	t.Log("Completed memcached metrics test")

	if err := memcachedOperatorMetricsTest(t, f, ctx); err != nil {
		t.Error(err)
	}
	t.Log("Completed memcached custom resource metrics test")
}

func memcachedMetricsTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	// Make sure metrics Service exists
	s := v1.Service{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-metrics", operatorName), Namespace: namespace}, &s)
	if err != nil {
		return fmt.Errorf("could not get metrics Service: (%v)", err)
	}
	if len(s.Spec.Selector) == 0 {
		return fmt.Errorf("no labels found in metrics Service")
	}

	// TODO(lili): Make port a constant in internal/scaffold/cmd.go.
	response, err := getMetrics(t, f, s.Spec.Selector, namespace, "8383")
	if err != nil {
		return fmt.Errorf("failed to get metrics: %v", err)
	}
	// Make sure metrics are present
	if len(response) == 0 {
		return fmt.Errorf("metrics body is empty")
	}

	// Perform prometheus metrics lint checks
	l := promlint.New(bytes.NewReader(response))
	problems, err := l.Lint()
	if err != nil {
		return fmt.Errorf("failed to lint metrics: %v", err)
	}
	// TODO(lili): Change to 0, when we upgrade to 1.14.
	// currently there is a problem with one of the metrics in upstream Kubernetes:
	// `workqueue_longest_running_processor_microseconds`.
	// This has been fixed in 1.14 release.
	if len(problems) > 1 {
		return fmt.Errorf("found problems with metrics: %#+v", problems)
	}

	return nil
}

func memcachedOperatorMetricsTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	// TODO(lili): Make port a constant in internal/scaffold/cmd.go.
	response, err := getMetrics(t, f, map[string]string{"name": operatorName}, namespace, "8686")
	if err != nil {
		return fmt.Errorf("failed to lint metrics: %v", err)
	}
	// Make sure metrics are present
	if len(response) == 0 {
		return fmt.Errorf("metrics body is empty")
	}

	// Perform prometheus metrics lint checks
	l := promlint.New(bytes.NewReader(response))
	problems, err := l.Lint()
	if err != nil {
		return fmt.Errorf("failed to lint metrics: %v", err)
	}
	if len(problems) > 0 {
		return fmt.Errorf("found problems with metrics: %#+v", problems)
	}

	// Make sure the metrics are the way we expect them.
	d := expfmt.NewDecoder(bytes.NewReader(response), expfmt.FmtText)
	var mf dto.MetricFamily
	for {
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		/*
			Metric:
			# HELP memcached_info Information about the Memcached operator replica.
			# TYPE memcached_info gauge
			memcached_info{namespace="memcached-memcached-group-cluster-1553683239",memcached="example-memcached"} 1
		*/
		if mf.GetName() != "memcached_info" {
			return fmt.Errorf("metric name was incorrect: expected %s, got %s", "memcached_info", mf.GetName())
		}
		if mf.GetType() != dto.MetricType_GAUGE {
			return fmt.Errorf("metric type was incorrect: expected %v, got %v", dto.MetricType_GAUGE, mf.GetType())
		}

		mlabels := mf.Metric[0].GetLabel()
		if mlabels[0].GetName() != "namespace" {
			return fmt.Errorf("metric label name was incorrect: expected %s, got %s", "namespace", mlabels[0].GetName())
		}
		if mlabels[0].GetValue() != namespace {
			return fmt.Errorf("metric label value was incorrect: expected %s, got %s", namespace, mlabels[0].GetValue())
		}
		if mlabels[1].GetName() != "memcached" {
			return fmt.Errorf("metric label name was incorrect: expected %s, got %s", "memcached", mlabels[1].GetName())
		}
		if mlabels[1].GetValue() != "example-memcached" {
			return fmt.Errorf("metric label value was incorrect: expected %s, got %s", "example-memcached", mlabels[1].GetValue())
		}

		if mf.Metric[0].GetGauge().GetValue() != float64(1) {
			return fmt.Errorf("metric counter was incorrect: expected %f, got %f", float64(1), mf.Metric[0].GetGauge().GetValue())
		}
	}

	return nil
}

func getMetrics(t *testing.T, f *framework.Framework, label map[string]string, ns, port string) ([]byte, error) {
	// Get operator pod
	pods := v1.PodList{}
	opts := client.InNamespace(ns)
	for k, v := range label {
		if err := opts.SetLabelSelector(fmt.Sprintf("%s=%s", k, v)); err != nil {
			return nil, fmt.Errorf("failed to set list label selector: (%v)", err)
		}
	}
	if err := opts.SetFieldSelector("status.phase=Running"); err != nil {
		return nil, fmt.Errorf("failed to set list field selector: (%v)", err)
	}
	err := f.Client.List(goctx.TODO(), opts, &pods)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: (%v)", err)
	}

	podName := ""
	numPods := len(pods.Items)
	// TODO(lili): Remove below logic when we enable exposing metrics in all pods.
	if numPods == 0 {
		podName = pods.Items[0].Name
	} else if numPods > 1 {
		// If we got more than one pod, get leader pod name.
		leader, err := verifyLeader(t, ns, f, label)
		if err != nil {
			return nil, err
		}
		podName = leader.Name
	} else {
		return nil, fmt.Errorf("failed to get operator pod: could not select any pods with selector %v", label)
	}
	// Pod name must be there, otherwise we cannot read metrics data via pod proxy.
	if podName == "" {
		return nil, fmt.Errorf("failed to get pod name")
	}

	// Get metrics data
	request := proxyViaPod(f.KubeClient, ns, podName, port, "/metrics")
	response, err := request.DoRaw()
	if err != nil {
		return nil, fmt.Errorf("failed to get response from metrics: %v", err)
	}

	return response, nil

}

func proxyViaPod(kubeClient kubernetes.Interface, namespace, podName, podPortName, path string) *rest.Request {
	return kubeClient.
		CoreV1().
		RESTClient().
		Get().
		Namespace(namespace).
		Resource("pods").
		SubResource("proxy").
		Name(fmt.Sprintf("%s:%s", podName, podPortName)).
		Suffix(path)
}
