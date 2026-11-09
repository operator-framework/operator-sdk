// Copyright 2026 The Operator-SDK Authors
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

package client

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

// Run with -race: reconciles of different namespaces look up and add watched secrets concurrently.
func TestGetWatchedSecretsForNamespaceConcurrently(t *testing.T) {
	acg := &actionConfigGetter{
		kubeClientSet:       fake.NewClientset(),
		watchedSecrets:      map[string]*WatchedSecrets{},
		watchedSecretsMutex: &sync.Mutex{},
	}
	const workers, namespaces = 8, 50
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range namespaces {
				ns := fmt.Sprintf("ns-%d", (n+w)%namespaces)
				assert.NotNil(t, acg.getWatchedSecretsForNamespace(ns))
			}
		}()
	}
	wg.Wait()
	assert.Len(t, acg.watchedSecrets, namespaces)
}
