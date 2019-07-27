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

package e2eutil

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/operator-framework/operator-sdk/pkg/test"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitForDeployment checks to see if a given deployment has a certain number of available replicas after a specified
// amount of time. If the deployment does not have the required number of replicas after 5 * retries seconds,
// the function returns an error. This can be used in multiple ways, like verifying that a required resource is ready
// before trying to use it, or to test. Failure handling, like simulated in SimulatePodFail.
func WaitForDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int,
	retryInterval, timeout time.Duration) error {
	return waitForDeployment(t, kubeclient, namespace, name, replicas, retryInterval, timeout, false)
}

// WaitForOperatorDeployment has the same functionality as WaitForDeployment but will no wait for the deployment if the
// test was run with a locally run operator (--up-local flag)
func WaitForOperatorDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int,
	retryInterval, timeout time.Duration) error {
	return waitForDeployment(t, kubeclient, namespace, name, replicas, retryInterval, timeout, true)
}

func waitForDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int,
	retryInterval, timeout time.Duration, isOperator bool) error {
	if isOperator && test.Global.LocalOperator {
		t.Log("Operator is running locally; skip waitForDeployment")
		return nil
	}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s deployment\n", name)
				return false, nil
			}
			return false, err
		}

		if int(deployment.Status.AvailableReplicas) >= replicas {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name,
			deployment.Status.AvailableReplicas, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Deployment available (%d/%d)\n", replicas, replicas)
	return nil
}

// WaitForDeletion checks to see if a given object is deleted, trying every retryInterval until the timeout has elapsed.
// If the object has not been deleted before the timeout, the function returns an error.
func WaitForDeletion(t *testing.T, dynclient client.Client, obj runtime.Object, retryInterval,
	timeout time.Duration) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return err
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = dynclient.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		t.Logf("Waiting for %s %s to be deleted\n", kind, key)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("%s %s was deleted\n", kind, key)
	return nil
}

// WaitForCondition checks to see if a given object has a given condition type/status pair, trying every retryInterval until
// the timeout has elapsed. If the condition is not found before the timeout, the function returns an error.
func WaitForCondition(t *testing.T, dynclient client.Client, obj runtime.Object, cType status.ConditionType, cStatus v1.ConditionStatus, retryInterval, timeout time.Duration) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = dynclient.Get(ctx, key, obj)
		if err != nil {
			return false, err
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return false, err
		}

		var cObj struct {
			Status struct {
				Conditions status.Conditions `json:"conditions"`
			} `json:"status"`
		}
		err = json.Unmarshal(data, &cObj)
		if err != nil {
			return false, err
		}

		c := cObj.Status.Conditions.GetCondition(cType)
		if c == nil {
			t.Logf("waiting for status %s %s, condition not found", cType, cStatus)
			return false, nil
		}

		if cStatus != c.Status {
			t.Logf("waiting for status %s %s, got %s", cType, cStatus, c.Status)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Found condition %s %s\n", cType, cStatus)
	return nil
}
