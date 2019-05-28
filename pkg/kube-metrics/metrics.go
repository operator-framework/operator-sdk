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
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	kcollector "k8s.io/kube-state-metrics/pkg/collector"
	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("kubemetrics")

// GetGVKsFromAddToScheme takes in the operator specific scheme and
// filters out all generic apimachinery meta types. It returns the GVK specific to this operator.
func GetGVKsFromAddToScheme(addToSchemeFunc func(*runtime.Scheme) error) ([]schema.GroupVersionKind, error) {
	s := kuberuntime.NewScheme()
	err := addToSchemeFunc(s)
	if err != nil {
		return nil, err
	}
	operatorKnownTypes := s.AllKnownTypes()
	operatorGVKs := []schema.GroupVersionKind{}
	for gvk, _ := range operatorKnownTypes {
		if !isKubeMetaKind(gvk.Kind) {
			operatorGVKs = append(operatorGVKs, gvk)
		}
	}

	return operatorGVKs, nil
}

func isKubeMetaKind(kind string) bool {
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
		return true
	}

	return false
}

// ServeCRMetrics generates CR specific metrics based on the operator GVK.
// It starts serving collections of those metrics on given host and port.
func ServeCRMetrics(cfg *rest.Config,
	ns []string,
	operatorGVKs []schema.GroupVersionKind,
	host string, port int32) error {
	// Create new unstructured client.
	uc := NewClientForConfig(cfg)
	var collectors [][]*kcollector.Collector
	log.V(1).Info("Starting collecting operator types")
	// Loop through all the possible operator/custom resource specific types.
	for _, gvk := range operatorGVKs {
		// Generate metric based on the kind.
		metricFamilies := generateMetricFamilies(gvk.Kind)
		log.V(1).Info("Generating metric families", "apiVersion", gvk.GroupVersion().String(), "kind", gvk.Kind)
		// Generate collector based on the group/version, kind and the metric families.
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
	log.V(1).Info("Starting serving custom resource metrics")
	go ServeMetrics(collectors, host, port)

	return nil
}

func generateMetricFamilies(kind string) []ksmetric.FamilyGenerator {
	helpText := fmt.Sprintf("Information about the %s operator custom resource replica.", kind)
	kindName := fmt.Sprintf("%s", strings.ToLower(kind))
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
