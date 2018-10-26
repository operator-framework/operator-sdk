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

package scorecard

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkSpecAndStat(runtimeClient client.Client, obj unstructured.Unstructured) error {
	testSpec := scorecardTest{testType: basicOperator, name: "Spec Block Exists", maximumPoints: 1}
	testStat := scorecardTest{testType: basicOperator, name: "Status Block Exist", maximumPoints: 1}
	var specPoints, statusPoints int
	err := wait.Poll(time.Second*1, time.Second*time.Duration(SCConf.InitTimeout), func() (bool, error) {
		err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: name}, &obj)
		if err != nil {
			return false, fmt.Errorf("error getting custom resource: %v", err)
		}
		pass := true
		if obj.Object["spec"] == nil {
			pass = false
			specPoints = 0
		} else {
			specPoints = 1
		}
		if obj.Object["status"] == nil {
			pass = false
			statusPoints = 0
		} else {
			statusPoints = 1
		}
		return pass, nil
	})
	testSpec.earnedPoints = specPoints
	testStat.earnedPoints = statusPoints
	scTests = append(scTests, testSpec)
	scTests = append(scTests, testStat)
	if err != nil && !reflect.DeepEqual(err, wait.ErrWaitTimeout) {
		return err
	}
	return nil
}

func checkStatusUpdate(runtimeClient client.Client, obj unstructured.Unstructured) error {
	test := scorecardTest{testType: basicOperator, name: "Operator actions are reflected in status", maximumPoints: 1}
	err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: name}, &obj)
	if err != nil {
		return fmt.Errorf("error getting custom resource: %v", err)
	}
	if obj.Object["status"] == nil || obj.Object["spec"] == nil {
		scTests = append(scTests, test)
		return nil
	}
	statCopy := make(map[string]interface{})
	for k, v := range obj.Object["status"].(map[string]interface{}) {
		statCopy[k] = v
	}
	pass := true
	for k, v := range obj.Object["spec"].(map[string]interface{}) {
		switch t := v.(type) {
		case int64:
			obj.Object["spec"].(map[string]interface{})[k] = obj.Object["spec"].(map[string]interface{})[k].(int64) + 1
		case float64:
			obj.Object["spec"].(map[string]interface{})[k] = obj.Object["spec"].(map[string]interface{})[k].(float64) + 1
		case string:
			// TODO: try and find out how to make this better
			// Since strings may be very operator specific, this test may not work correctly in many cases
			obj.Object["spec"].(map[string]interface{})[k] = fmt.Sprintf("operator sdk test value %f", rand.Float64())
		case bool:
			if obj.Object["spec"].(map[string]interface{})[k].(bool) {
				obj.Object["spec"].(map[string]interface{})[k] = false
			} else {
				obj.Object["spec"].(map[string]interface{})[k] = true
			}
		case []interface{}:
			fmt.Printf("This is unhandled at the moment\n")
		default:
			fmt.Printf("Unknown type for key %s: %v\n", k, reflect.TypeOf(t))
		}
		runtimeClient.Update(context.TODO(), &obj)
		err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
			runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: name}, &obj)
			if err != nil {
				return false, err
			}
			return !reflect.DeepEqual(statCopy, obj.Object["status"]), nil
		})
		if err != nil {
			pass = false
		}
		//reset stat copy to match
		statCopy = make(map[string]interface{})
		for k, v := range obj.Object["status"].(map[string]interface{}) {
			statCopy[k] = v
		}
	}
	if pass {
		test.earnedPoints = 1
	}
	scTests = append(scTests, test)
	return nil
}

// At the moment, this will just read the logs and print them if enabled. We will add more complex functionality
func writingIntoCRsHasEffect(obj unstructured.Unstructured) (string, error) {
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("failed to create kubeclient: %v", err)
	}
	dep := &appsv1.Deployment{}
	err = runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: deploymentName}, dep)
	if err != nil {
		return "", fmt.Errorf("failed to get newly created deployment: %v", err)
	}
	set := labels.Set(dep.Spec.Selector.MatchLabels)
	pods := &v1.PodList{}
	err = runtimeClient.List(context.TODO(), &client.ListOptions{LabelSelector: set.AsSelector()}, pods)
	if err != nil {
		return "", fmt.Errorf("failed to get list of pods in deployment: %v", err)
	}
	proxyPod = &pods.Items[0]
	req := kubeclient.CoreV1().Pods(SCConf.Namespace).GetLogs(proxyPod.GetName(), &v1.PodLogOptions{Container: "scorecard-proxy"})
	readCloser, err := req.Stream()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %v", err)
	}
	defer readCloser.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(readCloser)
	if err != nil {
		return "", fmt.Errorf("test failed and failed to read pod logs: %v", err)
	}
	return buf.String(), nil
}
