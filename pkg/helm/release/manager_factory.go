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

	"github.com/martinlindhe/base36"
	"github.com/pborman/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apitypes "k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/helm/pkg/chartutil"
	helmengine "k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/kube"
	rpb "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/tiller/environment"
	crmanager "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/operator-framework/operator-sdk/pkg/helm/client"
	"github.com/operator-framework/operator-sdk/pkg/helm/engine"
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
	clientv1, err := v1.NewForConfig(f.mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to get core/v1 client: %s", err)
	}
	storageBackend := storage.Init(driver.NewSecrets(clientv1.Secrets(cr.GetNamespace())))
	tillerKubeClient, err := client.NewFromManager(f.mgr)
	if err != nil {
		return nil, fmt.Errorf("failed to get client from manager: %s", err)
	}
	crChart, err := chartutil.LoadDir(f.chartDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart dir: %s", err)
	}
	releaseServer, err := getReleaseServer(cr, storageBackend, tillerKubeClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get helm release server: %s", err)
	}
	releaseName, err := getReleaseName(storageBackend, crChart.GetMetadata().GetName(), cr)
	if err != nil {
		return nil, fmt.Errorf("failed to get helm release name: %s", err)
	}
	return &manager{
		storageBackend:   storageBackend,
		tillerKubeClient: tillerKubeClient,
		chartDir:         f.chartDir,

		tiller:      releaseServer,
		releaseName: releaseName,
		namespace:   cr.GetNamespace(),

		spec:   cr.Object["spec"],
		status: types.StatusFor(cr),
	}, nil
}

// getReleaseServer creates a ReleaseServer configured with a rendering engine that adds ownerrefs to rendered assets
// based on the CR.
func getReleaseServer(cr *unstructured.Unstructured, storageBackend *storage.Storage, tillerKubeClient *kube.Client) (*tiller.ReleaseServer, error) {
	controllerRef := metav1.NewControllerRef(cr, cr.GroupVersionKind())
	ownerRefs := []metav1.OwnerReference{
		*controllerRef,
	}
	baseEngine := helmengine.New()
	restMapper, err := tillerKubeClient.Factory.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	e := engine.NewOwnerRefEngine(baseEngine, restMapper, ownerRefs)
	var ey environment.EngineYard = map[string]environment.Engine{
		environment.GoTplEngine: e,
	}
	env := &environment.Environment{
		EngineYard: ey,
		Releases:   storageBackend,
		KubeClient: tillerKubeClient,
	}
	kubeconfig, err := tillerKubeClient.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	cs, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return tiller.NewReleaseServer(env, cs, false), nil
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
// the possibility of collision. We should move this logic to a validating
// admission webhook so that the CR owner receives immediate feedback of the
// collision. As is, the only indication of collision will be in the CR status
// and operator logs.
func getReleaseName(storageBackend *storage.Storage, crChartName string, cr *unstructured.Unstructured) (string, error) {
	// If a release with the legacy name exists, return the legacy name.
	legacyName := fmt.Sprintf("%s-%s", cr.GetName(), shortenUID(cr.GetUID()))
	_, legacyExists, err := releaseHistory(storageBackend, legacyName)
	if err != nil {
		return "", err
	}
	if legacyExists {
		return legacyName, nil
	}

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
	existingChartName := history[0].GetChart().GetMetadata().GetName()
	if existingChartName != crChartName {
		return "", fmt.Errorf("duplicate release name: found existing release with name %q for chart %q", releaseName, existingChartName)
	}

	return releaseName, nil
}

func releaseHistory(storageBackend *storage.Storage, releaseName string) ([]*rpb.Release, bool, error) {
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
