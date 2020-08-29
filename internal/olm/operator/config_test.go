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

package operator

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

// TODO(joelanford): refactor to use ginkgo/gomega
func TestConfigLoad(t *testing.T) {
	// create temp kubeconfig files
	defaultFile, err := ioutil.TempFile("/tmp", "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(defaultFile.Name())

	if err := ioutil.WriteFile(defaultFile.Name(), []byte(defaultKubeconfig), 0644); err != nil {
		log.Fatal(err)
	}
	clientcmd.RecommendedHomeFile = defaultFile.Name()

	customFile, err := ioutil.TempFile("/tmp", "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(customFile.Name())

	if err := ioutil.WriteFile(customFile.Name(), []byte(testKubeconfig), 0644); err != nil {
		log.Fatal(err)
	}

	cases := []struct {
		name                 string
		customKubeconfigPath string
		kubeconfigEnvValue   string
		namespace            string
		expectErr            bool
		expectNamespace      string
	}{
		{"default", "", "", "", false, "my-default"},
		{"custom namespace", "", "", "userspecified", false, "userspecified"},
		{"non-existent override", "/tmp/doesnotexist", "", "", true, ""},
		{"custom override", customFile.Name(), "", "", false, "goo"},
		{"custom KUBECONFIG", "", customFile.Name(), "", false, "goo"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// set KUBECONFIG env var
			err = os.Setenv("KUBECONFIG", c.kubeconfigEnvValue)
			if err != nil {
				log.Fatal(err)
			}
			defer os.Unsetenv("KUBECONFIG")

			cfg := Configuration{
				KubeconfigPath: c.customKubeconfigPath,
				Namespace:      c.namespace,
			}
			err := cfg.Load()
			if err != nil {
				if !c.expectErr {
					t.Fatalf("Expected no error, got error: %v", err)
				}
			} else {
				if c.expectErr {
					t.Fatalf("Expected error, got no error")
				}
				if cfg.Namespace != c.expectNamespace {
					t.Errorf("Wanted namespace %q, got: %q", c.expectNamespace, cfg.Namespace)
				}
			}
		})
	}
}

const defaultKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    server: http://localhost:8080
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    namespace: my-default
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
`

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
