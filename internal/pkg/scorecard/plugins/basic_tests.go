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

package scplugins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BasicTestConfig contains all variables required by the BasicTest schelpers.TestSuite
type BasicTestConfig struct {
	Client   client.Client
	CR       *unstructured.Unstructured
	ProxyPod *v1.Pod
}

// Test Defintions

// CheckSpecTest is a scorecard test that verifies that the CR has a spec block
type CheckSpecTest struct {
	schelpers.TestInfo
	BasicTestConfig
}

// NewCheckSpecTest returns a new CheckSpecTest object
func NewCheckSpecTest(conf BasicTestConfig) *CheckSpecTest {
	return &CheckSpecTest{
		BasicTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Spec Block Exists",
			Description: "Custom Resource has a Spec Block",
			Cumulative:  false,
		},
	}
}

// CheckStatusTest is a scorecard test that verifies that the CR has a status block
type CheckStatusTest struct {
	schelpers.TestInfo
	BasicTestConfig
}

// NewCheckStatusTest returns a new CheckStatusTest object
func NewCheckStatusTest(conf BasicTestConfig) *CheckStatusTest {
	return &CheckStatusTest{
		BasicTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Status Block Exists",
			Description: "Custom Resource has a Status Block",
			Cumulative:  false,
		},
	}
}

// WritingIntoCRsHasEffectTest is a scorecard test that verifies that the operator is making PUT and/or POST requests to the API server
type WritingIntoCRsHasEffectTest struct {
	schelpers.TestInfo
	BasicTestConfig
}

// NewWritingIntoCRsHasEffectTest returns a new WritingIntoCRsHasEffectTest object
func NewWritingIntoCRsHasEffectTest(conf BasicTestConfig) *WritingIntoCRsHasEffectTest {
	return &WritingIntoCRsHasEffectTest{
		BasicTestConfig: conf,
		TestInfo: schelpers.TestInfo{
			Name:        "Writing into CRs has an effect",
			Description: "A CR sends PUT/POST requests to the API server to modify resources in response to spec block changes",
			Cumulative:  false,
		},
	}
}

// NewBasicTestSuite returns a new schelpers.TestSuite object containing basic, functional operator tests
func NewBasicTestSuite(conf BasicTestConfig) *schelpers.TestSuite {
	ts := schelpers.NewTestSuite(
		"Basic Tests",
		"Test suite that runs basic, functional operator tests",
	)
	ts.AddTest(NewCheckSpecTest(conf), 1.5)
	ts.AddTest(NewCheckStatusTest(conf), 1)
	ts.AddTest(NewWritingIntoCRsHasEffectTest(conf), 1)

	return ts
}

// Test Implementations

// Run - implements Test interface
func (t *CheckSpecTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, MaximumPoints: 1}
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

// Run - implements Test interface
func (t *CheckStatusTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, MaximumPoints: 1}
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

// Run - implements Test interface
func (t *WritingIntoCRsHasEffectTest) Run(ctx context.Context) *schelpers.TestResult {
	res := &schelpers.TestResult{Test: t, MaximumPoints: 1}
	logs, err := getProxyLogs(t.ProxyPod)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("error getting proxy logs: %v", err))
		return res
	}
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
		res.Suggestions = append(res.Suggestions, "The operator should write into objects to update state. No PUT or POST requests from the operator were recorded by the scorecard.")
	}
	return res
}
