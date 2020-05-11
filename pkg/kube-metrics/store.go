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

package kubemetrics

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

// NewNamespacedMetricsStores returns collections of metrics in the namespaces provided, per the api/kind resource.
// The metrics are registered in the custom metrics.FamilyGenerator that needs to be defined.
func NewNamespacedMetricsStores(dclient dynamic.NamespaceableResourceInterface, namespaces []string,
	api string, kind string, metricFamily []metric.FamilyGenerator) []*metricsstore.MetricsStore {
	namespaces = deduplicateNamespaces(namespaces)
	var stores []*metricsstore.MetricsStore
	// Generate collector per namespace.
	for _, ns := range namespaces {
		composedMetricGenFuncs := metric.ComposeMetricGenFuncs(metricFamily)
		headers := metric.ExtractMetricFamilyHeaders(metricFamily)
		store := metricsstore.NewMetricsStore(headers, composedMetricGenFuncs)
		reflectorPerNamespace(context.TODO(), dclient, &unstructured.Unstructured{}, store, ns)
		stores = append(stores, store)
	}
	return stores
}

// NewClusterScopedMetricsStores returns collections of metrics per the api/kind resource.
// The metrics are registered in the custom metrics.FamilyGenerator that needs to be defined.
func NewClusterScopedMetricsStores(dclient dynamic.NamespaceableResourceInterface, api string, kind string,
	metricFamily []metric.FamilyGenerator) []*metricsstore.MetricsStore {
	var stores []*metricsstore.MetricsStore
	// Generate collector per cluster scoped resources.
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(metricFamily)
	headers := metric.ExtractMetricFamilyHeaders(metricFamily)
	store := metricsstore.NewMetricsStore(headers, composedMetricGenFuncs)
	reflectorPerClusterScoped(context.TODO(), dclient, &unstructured.Unstructured{}, store)
	stores = append(stores, store)

	return stores
}

// NewMetricsStores returns collections of metrics in the namespaces provided, per the api/kind resource.
// The metrics are registered in the custom metrics.FamilyGenerator that needs to be defined.
//
// Deprecated: Use NewNamespacedMetricsStores instead.
func NewMetricsStores(dclient dynamic.NamespaceableResourceInterface, namespaces []string,
	api string, kind string, metricFamily []metric.FamilyGenerator) []*metricsstore.MetricsStore {
	return NewNamespacedMetricsStores(dclient, namespaces, api, kind, metricFamily)
}

func deduplicateNamespaces(ns []string) (list []string) {
	keys := make(map[string]struct{})
	for _, entry := range ns {
		if _, ok := keys[entry]; !ok {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}

func reflectorPerClusterScoped(
	ctx context.Context,
	dynamicInterface dynamic.NamespaceableResourceInterface,
	expectedType interface{},
	store cache.Store,
) {
	lw := clusterScopeListWatchFunc(dynamicInterface)
	reflector := cache.NewReflector(&lw, expectedType, store, 0)
	go reflector.Run(ctx.Done())
}

func reflectorPerNamespace(
	ctx context.Context,
	dynamicInterface dynamic.NamespaceableResourceInterface,
	expectedType interface{},
	store cache.Store,
	ns string,
) {
	lw := namespacedListWatchFunc(dynamicInterface, ns)
	reflector := cache.NewReflector(&lw, expectedType, store, 0)
	go reflector.Run(ctx.Done())
}

func namespacedListWatchFunc(dynamicInterface dynamic.NamespaceableResourceInterface,
	namespace string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return dynamicInterface.Namespace(namespace).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return dynamicInterface.Namespace(namespace).Watch(context.TODO(), opts)
		},
	}
}

func clusterScopeListWatchFunc(dynamicInterface dynamic.NamespaceableResourceInterface) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return dynamicInterface.List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return dynamicInterface.Watch(context.TODO(), opts)
		},
	}
}
