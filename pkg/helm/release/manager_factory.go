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

package release

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	crmanager "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/operator-framework/operator-sdk/pkg/helm/client"
	"github.com/operator-framework/operator-sdk/pkg/helm/internal/types"
)

// ManagerFactory creates Managers that are specific to custom resources. It is
// used by the HelmOperatorReconciler during resource reconciliation, and it
// improves decoupling between reconciliation logic and the Helm backend
// components used to manage releases.
type ManagerFactory interface {
	NewManager(r *unstructured.Unstructured, overrideValues map[string]string) (Manager, error)
}

type managerFactory struct {
	mgr      crmanager.Manager
	chartDir string
}

// NewManagerFactory returns a new Helm manager factory capable of installing and uninstalling releases.
func NewManagerFactory(mgr crmanager.Manager, chartDir string) ManagerFactory {
	return &managerFactory{mgr, chartDir}
}

func (f managerFactory) NewManager(cr *unstructured.Unstructured, overrideValues map[string]string) (Manager, error) {
	// Get both v2 and v3 storage backends
	clientv1, err := v1.NewForConfig(f.mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to get core/v1 client: %w", err)
	}
	storageBackend := storage.Init(driver.NewSecrets(clientv1.Secrets(cr.GetNamespace())))

	// Get the necessary clients and client getters. Use a client that injects the CR
	// as an owner reference into all resources templated by the chart.
	rcg, err := client.NewRESTClientGetter(f.mgr, cr.GetNamespace())
	if err != nil {
		return nil, fmt.Errorf("failed to get REST client getter from manager: %w", err)
	}

	kubeClient := kube.New(rcg)
	restMapper := f.mgr.GetRESTMapper()
	ownerRefClient, err := client.NewOwnerRefInjectingClient(*kubeClient, restMapper, cr)
	if err != nil {
		return nil, fmt.Errorf("failed to inject owner references: %w", err)
	}

	crChart, err := loader.LoadDir(f.chartDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart dir: %w", err)
	}

	releaseName, err := getReleaseName(storageBackend, crChart.Name(), cr)
	if err != nil {
		return nil, fmt.Errorf("failed to get helm release name: %w", err)
	}

	crValues, ok := cr.Object["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to get spec: expected map[string]interface{}")
	}

	expOverrides, err := parseOverrides(overrideValues)
	if err != nil {
		return nil, fmt.Errorf("failed to parse override values: %w", err)
	}
	values := mergeMaps(crValues, expOverrides)

	actionConfig := &action.Configuration{
		RESTClientGetter: rcg,
		Releases:         storageBackend,
		KubeClient:       ownerRefClient,
		Log:              func(_ string, _ ...interface{}) {},
	}

	return &manager{
		actionConfig:   actionConfig,
		storageBackend: storageBackend,
		kubeClient:     ownerRefClient,

		releaseName: releaseName,
		namespace:   cr.GetNamespace(),

		chart:  crChart,
		values: values,
		status: types.StatusFor(cr),
	}, nil
}

// getReleaseName returns a release name for the CR.
//
// getReleaseName searches for a release using the CR name. If a release
// cannot be found, or if it is found and was created by the chart managed
// by this manager, the CR name is returned.
//
// If a release is found but it was created by another chart, that means we
// have a release name collision, so return an error. This case is possible
// because Kubernetes allows instances of different types to have the same name
// in the same namespace.
//
// TODO(jlanford): As noted above, using the CR name as the release name raises
//   the possibility of collision. We should move this logic to a validating
//   admission webhook so that the CR owner receives immediate feedback of the
//   collision. As is, the only indication of collision will be in the CR status
//   and operator logs.
func getReleaseName(storageBackend *storage.Storage, crChartName string,
	cr *unstructured.Unstructured) (string, error) {
	// If a release with the CR name does not exist, return the CR name.
	releaseName := cr.GetName()
	history, exists, err := releaseHistory(storageBackend, releaseName)
	if err != nil {
		return "", err
	}
	if !exists {
		return releaseName, nil
	}

	// If a release name with the CR name exists, but the release's chart is
	// different than the chart managed by this operator, return an error
	// because something else created the existing release.
	if history[0].Chart == nil {
		return "", fmt.Errorf("could not find chart metadata in release with name %q", releaseName)
	}
	existingChartName := history[0].Chart.Name()
	if existingChartName != crChartName {
		return "", fmt.Errorf("duplicate release name: found existing release with name %q for chart %q",
			releaseName, existingChartName)
	}

	return releaseName, nil
}

func releaseHistory(storageBackend *storage.Storage, releaseName string) ([]*helmrelease.Release, bool, error) {
	releaseHistory, err := storageBackend.History(releaseName)
	if err != nil {
		if notFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return releaseHistory, len(releaseHistory) > 0, nil
}

func parseOverrides(in map[string]string) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	for k, v := range in {
		val := fmt.Sprintf("%s=%s", k, v)
		if err := strvals.ParseIntoString(val, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
