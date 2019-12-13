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
	"strings"

	helm2to3 "github.com/helm/helm-2to3/pkg/v3"
	"github.com/martinlindhe/base36"
	"github.com/pborman/uuid"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"
	helmreleasev3 "helm.sh/helm/v3/pkg/release"
	storagev3 "helm.sh/helm/v3/pkg/storage"
	driverv3 "helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apitypes "k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	helmreleasev2 "k8s.io/helm/pkg/proto/hapi/release"
	storagev2 "k8s.io/helm/pkg/storage"
	driverv2 "k8s.io/helm/pkg/storage/driver"
	crmanager "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/operator-framework/operator-sdk/pkg/helm/client"
	"github.com/operator-framework/operator-sdk/pkg/helm/internal/types"
)

// ManagerFactory creates Managers that are specific to custom resources. It is
// used by the HelmOperatorReconciler during resource reconciliation, and it
// improves decoupling between reconciliation logic and the Helm backend
// components used to manage releases.
type ManagerFactory interface {
	NewManager(r *unstructured.Unstructured) (Manager, error)
}

type managerFactory struct {
	mgr      crmanager.Manager
	chartDir string
}

// NewManagerFactory returns a new Helm manager factory capable of installing and uninstalling releases.
func NewManagerFactory(mgr crmanager.Manager, chartDir string) ManagerFactory {
	return &managerFactory{mgr, chartDir}
}

func (f managerFactory) NewManager(cr *unstructured.Unstructured) (Manager, error) {
	// Get both v2 and v3 storage backends
	clientv1, err := v1.NewForConfig(f.mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to get core/v1 client: %w", err)
	}
	storageBackendV2 := storagev2.Init(driverv2.NewSecrets(clientv1.Secrets(cr.GetNamespace())))
	storageBackendV3 := storagev3.Init(driverv3.NewSecrets(clientv1.Secrets(cr.GetNamespace())))

	// Automatically convert V2 releases to V3 releases. This is required to
	// maintain backward compatibility with old releases now that the
	// operator reconciliation loop expects Helm V3 releases.
	if err := convertV2ToV3(storageBackendV2, storageBackendV3, cr); err != nil {
		return nil, fmt.Errorf("failed to convert releases from v2 to v3: %w", err)
	}

	// Get the necessary clients and client getters. Use a client that injects the CR
	// as an owner reference into all resources templated by the chart.
	rcg, err := client.NewRESTClientGetter(f.mgr)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST client getter from manager: %w", err)
	}
	kubeClient := kube.New(nil)
	ownerRef := metav1.NewControllerRef(cr, cr.GroupVersionKind())
	ownerRefClient := client.NewOwnerRefInjectingClient(*kubeClient, *ownerRef)

	crChart, err := loader.LoadDir(f.chartDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart dir: %w", err)
	}

	releaseName, err := getReleaseName(storageBackendV3, crChart.Name(), cr)
	if err != nil {
		return nil, fmt.Errorf("failed to get helm release name: %w", err)
	}

	values, ok := cr.Object["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to get spec: expected map[string]interface{}")
	}

	actionConfig := &action.Configuration{
		RESTClientGetter: rcg,
		Releases:         storageBackendV3,
		KubeClient:       ownerRefClient,
		Log:              func(_ string, _ ...interface{}) {},
	}

	return &manager{
		actionConfig:   actionConfig,
		storageBackend: storageBackendV3,
		kubeClient:     ownerRefClient,

		releaseName: releaseName,
		namespace:   cr.GetNamespace(),

		chart:  crChart,
		values: values,
		status: types.StatusFor(cr),
	}, nil
}

func convertV2ToV3(storageBackendV2 *storagev2.Storage, storageBackendV3 *storagev3.Storage, cr *unstructured.Unstructured) error {
	// If a v2 release with the legacy name exists, convert it to v3.
	legacyName := getLegacyName(cr)
	legacyHistoryV2, legacyExistsV2, err := releaseHistoryV2(storageBackendV2, legacyName)
	if err != nil {
		return err
	}
	if legacyExistsV2 {
		return convertHistoryToV3(legacyHistoryV2, storageBackendV2, storageBackendV3)
	}

	// If a v2 release with the CR name exists, convert it to v3.
	releaseName := cr.GetName()
	historyV2, existsV2, err := releaseHistoryV2(storageBackendV2, releaseName)
	if err != nil {
		return err
	}
	if existsV2 {
		return convertHistoryToV3(historyV2, storageBackendV2, storageBackendV3)
	}
	return nil
}

func convertHistoryToV3(history []*helmreleasev2.Release, storageBackendV2 *storagev2.Storage, storageBackendV3 *storagev3.Storage) error {
	for _, relV2 := range history {
		relV3, err := helm2to3.CreateRelease(relV2)
		if err != nil {
			return fmt.Errorf("generate v3 release: %w", err)
		}
		if err := storageBackendV3.Create(relV3); err != nil {
			return fmt.Errorf("create v3 release: %w", err)
		}
		if _, err := storageBackendV2.Delete(relV2.GetName(), relV2.GetVersion()); err != nil {
			return fmt.Errorf("delete v2 release: %w", err)
		}
	}
	return nil
}

func getLegacyName(cr *unstructured.Unstructured) string {
	return fmt.Sprintf("%s-%s", cr.GetName(), shortenUID(cr.GetUID()))
}

// getReleaseName returns a release name for the CR. If a release for the
// legacy name exists, the legacy name is returned. This ensures
// backwards-compatibility for pre-existing CRs.
//
// If no releases are found with the legacy name, getReleaseName searches for
// a release using the CR name. If a release cannot be found, or if it is found
// and was created by the chart managed by this manager, the CR name is
// returned.
//
// If a release is found but it was created by another chart, that means we
// have a release name collision, so return an error. This case is possible
// because Kubernetes allows instances of different types to have the same name
// in the same namespace.
//
//     NOTE: The motivation for including the CR's UID was to prevent any
//     possibility of a collision between release names of CRs of different
//     types, so we now have to take extra precautions.
//
// The reason for this change is based on an interaction between the Kubernetes
// constraint that limits label values to 63 characters and the Helm convention
// of including the release name as a label on release resources.
//
// Since the legacy release name includes a 25-character value based on the
// parent CR's UID, it leaves little extra space for the CR name and any other
// identifying names or characters added by templates.
//
// TODO(jlanford): As noted above, using the CR name as the release name raises
//   the possibility of collision. We should move this logic to a validating
//   admission webhook so that the CR owner receives immediate feedback of the
//   collision. As is, the only indication of collision will be in the CR status
//   and operator logs.
func getReleaseName(storageBackend *storagev3.Storage, crChartName string, cr *unstructured.Unstructured) (string, error) {
	// If a release with the legacy name exists as a v3 release,
	// return the legacy name.
	legacyName := getLegacyName(cr)
	_, legacyExists, err := releaseHistoryV3(storageBackend, legacyName)
	if err != nil {
		return "", err
	}
	if legacyExists {
		return legacyName, nil
	}

	// If a release with the CR name does not exist, return the CR name.
	releaseName := cr.GetName()
	history, exists, err := releaseHistoryV3(storageBackend, releaseName)
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
		return "", fmt.Errorf("duplicate release name: found existing release with name %q for chart %q", releaseName, existingChartName)
	}

	return releaseName, nil
}

func releaseHistoryV2(storageBackend *storagev2.Storage, releaseName string) ([]*helmreleasev2.Release, bool, error) {
	releaseHistory, err := storageBackend.History(releaseName)
	if err != nil {
		if notFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return releaseHistory, len(releaseHistory) > 0, nil
}

func releaseHistoryV3(storageBackend *storagev3.Storage, releaseName string) ([]*helmreleasev3.Release, bool, error) {
	releaseHistory, err := storageBackend.History(releaseName)
	if err != nil {
		if notFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return releaseHistory, len(releaseHistory) > 0, nil
}

func shortenUID(uid apitypes.UID) string {
	u := uuid.Parse(string(uid))
	uidBytes, err := u.MarshalBinary()
	if err != nil {
		return strings.Replace(string(uid), "-", "", -1)
	}
	return strings.ToLower(base36.EncodeBytes(uidBytes))
}
