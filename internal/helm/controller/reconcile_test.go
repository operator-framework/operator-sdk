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
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDetermineReconcilePeriod(t *testing.T) {
	testPeriod1, _ := time.ParseDuration("10s")
	obj1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					"name":                        "test-obj-1",
					helmReconcilePeriodAnnotation: "3s",
				},
			},
		},
	}
	expected1, _ := time.ParseDuration("3s")
	finalPeriod1, err := determineReconcilePeriod(testPeriod1, obj1)
	assert.Equal(t, nil, err, "Verify that no error is returned on parsing the time period")
	assert.Equal(t, expected1, finalPeriod1, "Verify that the annotations period takes precedence")

	testPeriod2, _ := time.ParseDuration("1h3m4s")
	obj2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					"name": "test-obj-2",
				},
			},
		},
	}
	expected2, _ := time.ParseDuration("1h3m4s")
	finalPeriod2, err := determineReconcilePeriod(testPeriod2, obj2)
	assert.Equal(t, nil, err, "Verify that no error is returned on parsing the time period")
	assert.Equal(t, expected2, finalPeriod2, "Verify that when no time period is present under the CR's annotations, the original time period value gets used")

	testPeriod3, _ := time.ParseDuration("5m15s")
	obj3 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					"name":                        "test-obj-3",
					helmReconcilePeriodAnnotation: "4x",
				},
			},
		},
	}
	finalPeriod3, err := determineReconcilePeriod(testPeriod3, obj3)
	expected3, _ := time.ParseDuration("5m15s")
	assert.NotEqual(t, nil, err, "Verify that error is thrown when invalid time period is passed in the CR annotations")
	assert.Equal(t, expected3, finalPeriod3, "Verify that when a faulty time period is present under the CR's annotations, the original time period value gets used")
}

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

func Test_readBoolAnnotationWithDefault(t *testing.T) {
	objBuilder := func(anno map[string]string) *unstructured.Unstructured {
		object := &unstructured.Unstructured{}
		object.SetAnnotations(anno)
		return object
	}

	type args struct {
		obj        *unstructured.Unstructured
		annotation string
		fallback   bool
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Should return value of annotation read",
			args: args{
				obj: objBuilder(map[string]string{
					"helm.sdk.operatorframework.io/rollback-force": "false",
				}),
				annotation: "helm.sdk.operatorframework.io/rollback-force",
				fallback:   true,
			},
			want: false,
		},
		{
			name: "Should return fallback when annotation is not present",
			args: args{
				obj: objBuilder(map[string]string{
					"helm.sdk.operatorframework.io/upgrade-force": "true",
				}),
				annotation: "helm.sdk.operatorframework.io/rollback-force",
				fallback:   false,
			},
			want: false,
		},
		{
			name: "Should return fallback when errors while parsing bool value",
			args: args{
				obj: objBuilder(map[string]string{
					"helm.sdk.operatorframework.io/rollback-force": "force",
				}),
				annotation: "helm.sdk.operatorframework.io/rollback-force",
				fallback:   true,
			},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := readBoolAnnotationWithDefault(tc.args.obj, tc.args.annotation, tc.args.fallback); got != tc.want {
				assert.Equal(t, tc.want, got, "readBoolAnnotationWithDefault() function")
			}
		})
	}
}
