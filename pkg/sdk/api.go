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

package sdk

import (
	"context"
	"net/http"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	sdkHandler "github.com/operator-framework/operator-sdk/pkg/sdk/handler"
	sdkInformer "github.com/operator-framework/operator-sdk/pkg/sdk/informer"
	sdkMetrics "github.com/operator-framework/operator-sdk/pkg/sdk/metrics"
	_ "github.com/operator-framework/operator-sdk/pkg/sdk/pprof"

	"github.com/sirupsen/logrus"
)

var (
	// informers is the set of all informers for the resources watched by the user
	informers []sdkInformer.Informer
)

// Watch watches for changes on the given resource.
// apiVersion for a resource is of the format "Group/Version" except for the "Core" group whose APIVersion is just "v1". For e.g:
//   - Deployments have Group "apps" and Version "v1beta2" giving the APIVersion "apps/v1beta2"
//   - Pods have Group "Core" and Version "v1" giving the APIVersion "v1"
//   - The custom resource Memcached might have Group "cache.example.com" and Version "v1alpha1" giving the APIVersion "cache.example.com/v1alpha1"
// kind is the Kind of the resource, e.g "Pod" for pods
// resyncPeriod is the time period in seconds for how often an event with the latest resource version will be sent to the handler, even if there is no change.
//   - 0 means no periodic events will be sent
// Consult the API reference for the Group, Version and Kind of a resource: https://kubernetes.io/docs/reference/
// namespace is the Namespace to watch for the resource
// TODO: support opts for specifying label selector
func Watch(apiVersion, kind, namespace string, resyncPeriod int) {
	resourceClient, resourcePluralName, err := k8sclient.GetResourceClient(apiVersion, kind, namespace)
	// TODO: Better error handling, e.g retry
	if err != nil {
		logrus.Errorf("failed to get resource client for (apiVersion:%s, kind:%s, ns:%s): %v", apiVersion, kind, namespace, err)
		panic(err)
	}
	informer := sdkInformer.New(resourcePluralName, namespace, resourceClient, resyncPeriod)
	informers = append(informers, informer)
}

// Handle registers the handler for all events.
// In the future, we would have a mux-pattern to dispatch events to matched handlers.
func Handle(handler sdkHandler.Handler) {
	sdkHandler.RegisteredHandler = handler
}

// Run starts the process of Watching resources, handling Events, and processing Actions
func Run(ctx context.Context) {
	for _, informer := range informers {
		go informer.Run(ctx)
	}
	<-ctx.Done()
}

func RegisterMetrics() {
	addr := ":8080"
	if err := sdkMetrics.Register(); err != nil {
		logrus.Errorf("failed to register metrics: %v", err)
		panic(err)
	}
	// pprof was registered to http.DefaultServeMutex just by importing the package
	go func() {
		logrus.Println(http.ListenAndServe(addr, nil))
	}()
	logrus.Infof("Metrics and pprof registered on: %s", addr)
}
