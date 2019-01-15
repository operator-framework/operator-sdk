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

// NewFromFlag returns a storage backend based on the passed driver flag. For
// persistent storage drivers, release history will be stored using the
// corresponding resource type in the passed namespace. An error is returned
// if the driver is not recognized or if a failure occurs creating a client
// from the passed REST config.
func NewFromFlag(cfg *rest.Config, namespace, storageDriver string) (*storage.Storage, error) {
	switch storageDriver {
	case driver.ConfigMapsDriverName:
		return NewConfigMaps(cfg, namespace)
	case driver.SecretsDriverName:
		return NewSecrets(cfg, namespace)
	case driver.MemoryDriverName:
		return NewMemory(), nil
	default:
		return nil, fmt.Errorf("invalid storage driver \"%s\"", storageDriver)
	}
}

// NewConfigMaps returns a ConfigMap storage backend. Release history will be
// stored using ConfigMap resources in the passed namespace. An error is returned
// if a failure occurs creating a client from the passed REST config.
func NewConfigMaps(cfg *rest.Config, namespace string) (*storage.Storage, error) {
	client, err := corev1.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get v1 client: %s", err)
	}
	cmc := client.ConfigMaps(namespace)
	return storage.Init(driver.NewConfigMaps(cmc)), nil
}

// NewSecrets returns a Secret storage backend. Release history will be
// stored using Secret resources in the passed namespace. An error is returned
// if a failure occurs creating a client from the passed REST config.
func NewSecrets(cfg *rest.Config, namespace string) (*storage.Storage, error) {
	client, err := corev1.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get v1 client: %s", err)
	}
	sc := client.Secrets(namespace)
	return storage.Init(driver.NewSecrets(sc)), nil
}

// NewMemory returns an in-memory storage driver. Release history will be
// stored in the memory of the calling process.
//
// NOTE: This driver is not recommended for production use. When the process
// exits, all state about release history is lost.
func NewMemory() *storage.Storage {
	return storage.Init(driver.NewMemory())
}
