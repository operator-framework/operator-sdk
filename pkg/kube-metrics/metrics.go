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
	"fmt"
	"reflect"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	kcollector "k8s.io/kube-state-metrics/pkg/collector"
	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("kubemetrics")

// ServeCRMetrics generates CR specific metrics based on the operator GVK.
// It starts serving collections of those metrics on given host and port.
func ServeCRMetrics(cfg *rest.Config, ns []string, operatorTypes map[schema.GroupVersionKind]reflect.Type, host string, port int32) error {
	// Create new unstructured client.
	uc := NewClientForConfig(cfg)
	var collectors [][]*kcollector.Collector
	log.V(1).Info("Starting collecting operator types")

	// Loop through all the possible operator specific types.
	for gvk, _ := range operatorTypes {
		// Generate metric based on the kind.
		metricFamilies := generateMetricFamilies(gvk.Kind)
		log.V(1).Info("Generating metric families for kind ", gvk.Kind, "and group/version", gvk.GroupVersion().String())
		// Generate collector based on the resource, kind and the metric families.
		c, err := NewCollectors(uc, ns, gvk.GroupVersion().String(), gvk.Kind, metricFamilies)
		if err != nil {
			if err == k8sutil.ErrNoNamespace {
				log.Info("Skipping operator specific metrics; not running in a cluster.")
				return nil
			}
			return err
		}
		collectors = append(collectors, c)
	}
	// Start serving metrics.
	log.V(1).Info("Starting serving metric families")
	go ServeMetrics(collectors, host, port)

	return nil
}

func generateMetricFamilies(kind string) []ksmetric.FamilyGenerator {
	helpText := fmt.Sprintf("Information about the %s operator replica.", kind)
	kindName := fmt.Sprintf("%s", kind)
	metricName := fmt.Sprintf("%s_info", strings.ToLower(kind))

	return []ksmetric.FamilyGenerator{
		ksmetric.FamilyGenerator{
			Name: metricName,
			Type: ksmetric.Gauge,
			Help: helpText,
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				crd := obj.(*unstructured.Unstructured)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       1,
							LabelKeys:   []string{"namespace", kindName},
							LabelValues: []string{crd.GetNamespace(), crd.GetName()},
						},
					},
				}
			},
		},
	}
}

// FilterOut takes in the operator specific scheme and filters out all generic apimachinery meta types.
// It returns the GVK specific to this operator.
func FilterOutMetaTypes(operatorSpecificScheme *runtime.Scheme) map[schema.GroupVersionKind]reflect.Type {
	allOperatorKnownTypes := operatorSpecificScheme.AllKnownTypes()
	for gvk, _ := range allOperatorKnownTypes {
		kind := gvk.Kind
		if strings.HasSuffix(kind, "List") ||
			kind == "GetOptions" ||
			kind == "DeleteOptions" ||
			kind == "ExportOptions" ||
			kind == "APIVersions" ||
			kind == "APIGroupList" ||
			kind == "APIResourceList" ||
			kind == "UpdateOptions" ||
			kind == "CreateOptions" ||
			kind == "Status" ||
			kind == "WatchEvent" ||
			kind == "ListOptions" ||
			kind == "APIGroup" {
			delete(allOperatorKnownTypes, gvk)
		}
	}

	return allOperatorKnownTypes
}
