/*
Copyright 2017 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	"io"
)

// Store represents a metrics store e.g.
// k8s.io/kube-state-metrics/pkg/metrics_store.
type Store interface {
	WriteAll(io.Writer)
}

// Collector represents a kube-state-metrics metric collector. It is a stripped
// down version of the Prometheus client_golang collector.
type Collector struct {
	Store Store
}

// NewCollector constructs a collector with the given Store.
func NewCollector(s Store) *Collector {
	return &Collector{s}
}

// Collect returns all metrics of the underlying store of the collector.
func (c *Collector) Collect(w io.Writer) {
	c.Store.WriteAll(w)
}
