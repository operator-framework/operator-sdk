// Copyright 2019 The Operator-SDK Authors
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
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *CheckSpecTest) Run(ctx context.Context) TestResult {
	res := TestResult{Test: t, MaximumPoints: 1}
	err := t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("error getting custom resource: %v", err))
		return res
	}
	if t.CR.Object["spec"] != nil {
		res.EarnedPoints++
	}
	if res.EarnedPoints != 1 {
		res.Suggestions = append(res.Suggestions, "Add a 'spec' field to your Custom Resource")
	}
	return res
}

// checkStat checks that the status block exists
func (t *CheckStatusTest) Run(ctx context.Context) TestResult {
	res := TestResult{Test: t, MaximumPoints: 1}
	err := t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: t.CR.GetName()}, t.CR)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("error getting custom resource: %v", err))
		return res
	}
	if t.CR.Object["status"] != nil {
		res.EarnedPoints++
	}
	if res.EarnedPoints != 1 {
		res.Suggestions = append(res.Suggestions, "Add a 'status' field to your Custom Resource")
	}
	return res
}

// writingIntoCRsHasEffect simply looks at the proxy logs and verifies that the operator is sending PUT
// and/or POST requests to the API server, which should mean that it is creating or modifying resources.
func (t *WritingIntoCRsHasEffectTest) Run(ctx context.Context) TestResult {
	res := TestResult{Test: t, MaximumPoints: 1}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("failed to create kubeclient: %v", err))
		return res
	}
	dep := &appsv1.Deployment{}
	err = t.Client.Get(ctx, types.NamespacedName{Namespace: t.CR.GetNamespace(), Name: deploymentName}, dep)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("failed to get newly created operator deployment: %v", err))
		return res
	}
	set := labels.Set(dep.Spec.Selector.MatchLabels)
	pods := &v1.PodList{}
	err = t.Client.List(ctx, &client.ListOptions{LabelSelector: set.AsSelector()}, pods)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("failed to get list of pods in deployment: %v", err))
		return res
	}
	proxyPod = &pods.Items[0]
	// this is a temporary workaround; will be fixed with PR #1027
	t.ProxyPod = proxyPod
	req := kubeclient.CoreV1().Pods(t.CR.GetNamespace()).GetLogs(t.ProxyPod.GetName(), &v1.PodLogOptions{Container: "scorecard-proxy"})
	readCloser, err := req.Stream()
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("failed to get logs: %v", err))
		return res
	}
	defer func() {
		if err := readCloser.Close(); err != nil && !fileutil.IsClosedError(err) {
			log.Errorf("Failed to close pod log reader: (%v)", err)
		}
	}()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(readCloser)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("test failed and failed to read pod logs: %v", err))
		return res
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
			res.EarnedPoints = 1
			break
		}
	}
	if res.EarnedPoints != 1 {
		res.Suggestions = append(res.Suggestions, "The operator should write into objects to update state. No PUT or POST requests from you operator were recorded by the scorecard.")
	}
	return res
}
