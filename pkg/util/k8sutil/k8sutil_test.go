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
		Scenario{
			name:        "Simple case",
			envVarKey:   OperatorNameEnvVar,
			envVarValue: "myoperator",
			expectedOutput: Output{
				operatorName: "myoperator",
				err:          nil,
			},
		},
		Scenario{
			name:        "Unset env var",
			envVarKey:   "",
			envVarValue: "",
			expectedOutput: Output{
				operatorName: "",
				err:          fmt.Errorf("%s must be set", OperatorNameEnvVar),
			},
		},
		Scenario{
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
