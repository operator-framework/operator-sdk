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

package alpha

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// createConfigMap creates a ConfigMap that will hold the bundle
// contents to be mounted into the test Pods
func createConfigMap(o Scorecard, bundleData []byte) (configMapName string, err error) {
	cfg := getConfigMapDefinition(o.Namespace, bundleData)
	configMap, err := o.Client.CoreV1().ConfigMaps(o.Namespace).Create(context.TODO(), cfg, metav1.CreateOptions{})
	return configMap.Name, err
}

// getConfigMapDefinition returns a ConfigMap definition that
// will hold the bundle contents and eventually will be mounted
// into each test Pod
func getConfigMapDefinition(namespace string, bundleData []byte) *v1.ConfigMap {
	configMapName := fmt.Sprintf("scorecard-test-%s", rand.String(4))
	data := make(map[string][]byte)
	data["bundle.tar"] = bundleData
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "scorecard-test",
			},
		},
		BinaryData: data,
	}
}

// deleteConfigMap deletes the test bundle ConfigMap and is called
// as part of the test run cleanup
func deleteConfigMap(o Scorecard) {
	err := o.Client.CoreV1().ConfigMaps(o.Namespace).Delete(context.TODO(), o.BundleConfigMapName, metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Error deleting configMap %s %w", o.BundleConfigMapName, err)
	}
}
