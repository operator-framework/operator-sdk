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

package k8sutil

import (
	yaml "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime"
)

// GetObjectBytes marshalls an object and removes runtime-managed fields:
// 'status', 'creationTimestamp'
func GetObjectBytes(obj interface{}) ([]byte, error) {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	deleteKeys := []string{"status", "creationTimestamp"}
	for _, dk := range deleteKeys {
		deleteKeyFromUnstructured(u, dk)
	}
	return yaml.Marshal(u)
}

func deleteKeyFromUnstructured(u map[string]interface{}, key string) {
	if _, ok := u[key]; ok {
		delete(u, key)
		return
	}

	for _, v := range u {
		switch t := v.(type) {
		case map[string]interface{}:
			deleteKeyFromUnstructured(t, key)
		case []interface{}:
			for _, ti := range t {
				if m, ok := ti.(map[string]interface{}); ok {
					deleteKeyFromUnstructured(m, key)
				}
			}
		}
	}
}
