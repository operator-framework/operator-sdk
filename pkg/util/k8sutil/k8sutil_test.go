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
