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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestHasAnnotation(t *testing.T) {
	upgradeForceTests := []struct {
		input       map[string]interface{}
		expectedVal bool
		expectedOut string
		name        string
	}{
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/upgrade-force": "True",
			},
			expectedVal: true,
			name:        "upgrade force base case true",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/upgrade-force": "False",
			},
			expectedVal: false,
			name:        "upgrade force base case false",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/upgrade-force": "1",
			},
			expectedVal: true,
			name:        "upgrade force true as int",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/upgrade-force": "0",
			},
			expectedVal: false,
			name:        "upgrade force false as int",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/wrong-annotation": "true",
			},
			expectedVal: false,
			name:        "upgrade force annotation not set",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/upgrade-force": "invalid",
			},
			expectedVal: false,
			name:        "upgrade force invalid value",
		},
	}

	for _, test := range upgradeForceTests {
		assert.Equal(t, test.expectedVal, hasAnnotation(helmUpgradeForceAnnotation, annotations(test.input)), test.name)
	}

	uninstallWaitTests := []struct {
		input       map[string]interface{}
		expectedVal bool
		expectedOut string
		name        string
	}{
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/uninstall-wait": "True",
			},
			expectedVal: true,
			name:        "uninstall wait base case true",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/uninstall-wait": "False",
			},
			expectedVal: false,
			name:        "uninstall wait base case false",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/uninstall-wait": "1",
			},
			expectedVal: true,
			name:        "uninstall wait true as int",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/uninstall-wait": "0",
			},
			expectedVal: false,
			name:        "uninstall wait false as int",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/wrong-annotation": "true",
			},
			expectedVal: false,
			name:        "uninstall wait annotation not set",
		},
		{
			input: map[string]interface{}{
				"helm.sdk.operatorframework.io/uninstall-wait": "invalid",
			},
			expectedVal: false,
			name:        "uninstall wait invalid value",
		},
	}

	for _, test := range uninstallWaitTests {
		assert.Equal(t, test.expectedVal, hasAnnotation(helmUninstallWaitAnnotation, annotations(test.input)), test.name)
	}
}

func annotations(m map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": m,
			},
		},
	}
}
