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
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
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

// checkSpecAndStat checks that the spec and status blocks exist. If noStore is set to true, this function
// will not store the result of the test in sCTests and will instead just wait wait until the spec and
// status blocks exist or return an error after the timeout.
func checkSpecAndStat(runtimeClient client.Client, obj unstructured.Unstructured, noStore bool) error {
	testSpec := scorecardTest{testType: basicOperator, name: "Spec Block Exists", maximumPoints: 1}
	testStat := scorecardTest{testType: basicOperator, name: "Status Block Exist", maximumPoints: 1}
	err := wait.Poll(time.Second*1, time.Second*time.Duration(SCConf.InitTimeout), func() (bool, error) {
		err := runtimeClient.Get(context.TODO(), types.NamespacedName{Namespace: SCConf.Namespace, Name: name}, &obj)
		if err != nil {
			return false, fmt.Errorf("error getting custom resource: %v", err)
		}
		var specPass, statusPass bool
		if obj.Object["spec"] != nil {
			testSpec.earnedPoints = 1
			specPass = true
		}

		if obj.Object["status"] != nil {
			testStat.earnedPoints = 1
			statusPass = true
		}
		return statusPass && specPass, nil
	})
	if !noStore {
		scTests = append(scTests, testSpec, testStat)
	}
	if err != nil && !reflect.DeepEqual(err, wait.ErrWaitTimeout) {
		return err
	}
	return nil
}

// TODO: user specified tests for operators

// checkStatusUpdate looks at all fields in the spec section of a custom resource and attempts to modify them and
// see if the status changes as a result. This is a bit prone to breakage as this is a black box test and we don't
// know much about how the operators we are testing actually work and may pass an invalid value. In the future, we
// should use user-specified tests
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
	specMap := obj.Object["spec"].(map[string]interface{})
	for k, v := range specMap {
		switch t := v.(type) {
		case int64:
			specMap[k] = specMap[k].(int64) + 1
		case float64:
			specMap[k] = specMap[k].(float64) + 1
		case string:
			// TODO: try and find out how to make this better
			// Since strings may be very operator specific, this test may not work correctly in many cases
			specMap[k] = fmt.Sprintf("operator sdk test value %f", rand.Float64())
		case bool:
			specMap[k] = !specMap[k].(bool)
		case []interface{}: // TODO: Decide how this should be handled
		default:
			fmt.Printf("Unknown type for key (%s) in status: (%v)\n", k, reflect.TypeOf(t))
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
			test.earnedPoints = 0
			scTests = append(scTests, test)
			return nil
		}
		//reset stat copy to match
		statCopy = make(map[string]interface{})
		for k, v := range obj.Object["status"].(map[string]interface{}) {
			statCopy[k] = v
		}
	}
	test.earnedPoints = 1
	scTests = append(scTests, test)
	return nil
}

// At the moment, this will just read the logs and print them if enabled. We will add more complex functionality later
func writingIntoCRsHasEffect(obj unstructured.Unstructured) (string, error) {
	test := scorecardTest{testType: basicOperator, name: "Writing into CRs has an effect", maximumPoints: 1}
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
	logs := buf.String()
	msgMap := make(map[string]interface{})
	for _, msg := range strings.Split(logs, "\n") {
		if err := json.Unmarshal([]byte(msg), &msgMap); err != nil {
			continue
		}
		method, ok := msgMap["method"].(string)
		if !ok {
			continue
		}
		if method == "PUT" || method == "POST" {
			test.earnedPoints = 1
			break
		}
	}
	scTests = append(scTests, test)
	return buf.String(), nil
}
