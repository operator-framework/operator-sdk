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

package storage

import (
	"fmt"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"
)

func NewFromFlag(cfg *rest.Config, namespace, backend string) (*storage.Storage, error) {
	switch backend {
	case "configmap":
		return NewConfigMaps(cfg, namespace)
	case "secret":
		return NewSecrets(cfg, namespace)
	case "memory":
		return NewMemory(), nil
	default:
		return nil, fmt.Errorf("invalid storage backend \"%s\"", backend)
	}
}

func NewConfigMaps(cfg *rest.Config, namespace string) (*storage.Storage, error) {
	client, err := corev1.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get v1 client: %s", err)
	}
	cmc := client.ConfigMaps(namespace)
	return storage.Init(driver.NewConfigMaps(cmc)), nil
}

func NewSecrets(cfg *rest.Config, namespace string) (*storage.Storage, error) {
	client, err := corev1.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get v1 client: %s", err)
	}
	sc := client.Secrets(namespace)
	return storage.Init(driver.NewSecrets(sc)), nil
}

func NewMemory() *storage.Storage {
	return storage.Init(driver.NewMemory())
}
