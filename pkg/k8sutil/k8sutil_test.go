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

package k8sutil

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetOperatorName(t *testing.T) {
	type Output struct {
		operatorName string
		err          error
	}

	type Scenario struct {
		name           string
		envVarKey      string
		envVarValue    string
		expectedOutput Output
	}

	tests := []Scenario{
		{
			name:        "Simple case",
			envVarKey:   OperatorNameEnvVar,
			envVarValue: "myoperator",
			expectedOutput: Output{
				operatorName: "myoperator",
				err:          nil,
			},
		},
		{
			name:        "Unset env var",
			envVarKey:   "",
			envVarValue: "",
			expectedOutput: Output{
				operatorName: "",
				err:          fmt.Errorf("%s must be set", OperatorNameEnvVar),
			},
		},
		{
			name:        "Empty env var",
			envVarKey:   OperatorNameEnvVar,
			envVarValue: "",
			expectedOutput: Output{
				operatorName: "",
				err:          fmt.Errorf("%s must not be empty", OperatorNameEnvVar),
			},
		},
	}

	for _, test := range tests {
		_ = os.Setenv(test.envVarKey, test.envVarValue)
		operatorName, err := GetOperatorName()
		if !(operatorName == test.expectedOutput.operatorName && reflect.DeepEqual(err, test.expectedOutput.err)) {
			t.Errorf("test %s failed, expected ouput: %s,%v; got: %s,%v", test.name, test.expectedOutput.operatorName, test.expectedOutput.err, operatorName, err)
		}
		_ = os.Unsetenv(test.envVarKey)
	}
}

func TestInitOperatorService(t *testing.T) {
	operatorName := "myTestOperator"
	namespace := "myTestNamespace"

	serviceExp := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: namespace,
			Labels:    map[string]string{"name": operatorName},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:     PrometheusMetricsPort,
					Protocol: v1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: PrometheusMetricsPortName,
					},
					Name: PrometheusMetricsPortName,
				},
			},
			Selector: map[string]string{"name": operatorName},
		},
	}

	type Output struct {
		service *v1.Service
		err     error
	}

	type Scenario struct {
		name           string
		envVars        map[string]string
		expectedOutput Output
	}
	tests := []Scenario{
		{
			name:    "WatchNamespace Case",
			envVars: map[string]string{OperatorNameEnvVar: operatorName, WatchNamespaceEnvVar: namespace},
			expectedOutput: Output{
				service: serviceExp,
				err:     nil,
			},
		},
		{
			name:    "ClusterScope Case",
			envVars: map[string]string{OperatorNameEnvVar: operatorName, OperatorNamespaceEnvVar: namespace, WatchNamespaceEnvVar: ""},
			expectedOutput: Output{
				service: serviceExp,
				err:     nil,
			},
		},
		{
			name:    "Error no namespace and empty watchnamespace",
			envVars: map[string]string{OperatorNameEnvVar: operatorName, WatchNamespaceEnvVar: ""},
			expectedOutput: Output{
				service: nil,
				err:     fmt.Errorf("one of the env var %s or %s must not be empty", WatchNamespaceEnvVar, OperatorNamespaceEnvVar),
			},
		},
		{
			name:    "Error no namespace no watchnamespace",
			envVars: map[string]string{OperatorNameEnvVar: operatorName},
			expectedOutput: Output{
				service: nil,
				err:     fmt.Errorf("%s must be set", WatchNamespaceEnvVar),
			},
		},
		{
			name:    "Error no namespace and watchnamespace are empty",
			envVars: map[string]string{OperatorNameEnvVar: operatorName, OperatorNamespaceEnvVar: "", WatchNamespaceEnvVar: ""},
			expectedOutput: Output{
				service: nil,
				err:     fmt.Errorf("one of the env var %s or %s must not be empty", WatchNamespaceEnvVar, OperatorNamespaceEnvVar),
			},
		},
	}

	for _, test := range tests {
		for k, v := range test.envVars {
			_ = os.Setenv(k, v)
		}
		service, err := InitOperatorService()
		if !(reflect.DeepEqual(err, test.expectedOutput.err) && reflect.DeepEqual(service, test.expectedOutput.service)) {
			t.Errorf("test %s failed, expected ouput: %s,%v; got: %s,%v", test.name, test.expectedOutput.service, test.expectedOutput.err, service, err)
		}

		for k := range test.envVars {
			_ = os.Unsetenv(k)
		}
	}
}
