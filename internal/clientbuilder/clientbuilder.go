// Copyright 2021 The Operator-SDK Authors
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

package clientbuilder

import (
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewUnstructedCached returns a builder to build new client clients.
// The new delegating client allows caching of unstructured objects.
func NewUnstructedCached() manager.ClientBuilder {
	return &newUnstructedCached{}
}

type newUnstructedCached struct {
	uncached []client.Object
}

func (n *newUnstructedCached) WithUncached(objs ...client.Object) manager.ClientBuilder {
	n.uncached = append(n.uncached, objs...)
	return n
}

func (n *newUnstructedCached) Build(cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
	// Create the Client for Write operations.
	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	return client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader:       cache,
		Client:            c,
		UncachedObjects:   n.uncached,
		CacheUnstructured: true,
	})
}
