// Copyright 2020 The Operator-SDK Authors
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
	"log"
	"os"
	"testing"
)

func TestGetKubeNamespace(t *testing.T) {

	// create temp kubeconfig file
	file, err := os.CreateTemp("/tmp", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.Remove(file.Name())

	data := []byte(testKubeconfig)
	err = os.WriteFile(file.Name(), data, 0644)
	if err != nil {
		log.Fatal(err)
	}

	cases := []struct {
		kubeconfigPath string
		namespace      string
		expectedValue  string
	}{
		{"", "userspecified", "userspecified"},
		{"/tmp/doesnotexist", "", "default"},
		{file.Name(), "", "goo"},
	}

	for _, c := range cases {
		t.Run(c.kubeconfigPath, func(t *testing.T) {

			oNamespace := GetKubeNamespace(c.kubeconfigPath, c.namespace)
			if oNamespace != c.expectedValue {
				t.Errorf("Wanted namespace %s, got: %s", c.expectedValue, oNamespace)
			}
		})

	}
}

func TestGetKubeNamespaceEnvVar(t *testing.T) {

	// create temp kubeconfig file
	file, err := os.CreateTemp("/tmp", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.Remove(file.Name())

	data := []byte(testKubeconfig)
	err = os.WriteFile(file.Name(), data, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// set KUBECONFIG env var
	err = os.Setenv("KUBECONFIG", file.Name())
	if err != nil {
		log.Fatal(err)
	}

	cases := []struct {
		kubeconfigPath string
		namespace      string
		expectedValue  string
	}{
		{"", "", "goo"},
	}

	for _, c := range cases {
		t.Run(c.kubeconfigPath, func(t *testing.T) {
			oNamespace := GetKubeNamespace(c.kubeconfigPath, c.namespace)
			if oNamespace != c.expectedValue {
				t.Errorf("Wanted namespace %s, got: %s", c.expectedValue, oNamespace)
			}
		})

	}
}

const testKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    server: https://192.168.0.130:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    namespace: foo
    user: kubernetes-admin
  name: dev
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
- context:
    cluster: kubernetes
    namespace: goo
    user: kubernetes-admin
  name: prod
current-context: prod
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
`
