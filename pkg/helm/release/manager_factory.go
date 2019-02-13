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
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	helmengine "k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/tiller/environment"
	crmanager "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/operator-framework/operator-sdk/pkg/helm/client"
	"github.com/operator-framework/operator-sdk/pkg/helm/engine"
	"github.com/operator-framework/operator-sdk/pkg/helm/internal/types"
	"github.com/operator-framework/operator-sdk/pkg/helm/storage/driver"
)

// ManagerFactory creates Managers that are specific to custom resources. It is
// used by the HelmOperatorReconciler during resource reconciliation, and it
// improves decoupling between reconciliation logic and the Helm backend
// components used to manage releases.
type ManagerFactory interface {
	NewManager(r *unstructured.Unstructured) (Manager, error)
}

type managerFactory struct {
	crmanager crmanager.Manager
	chartDir  string
}

// NewManagerFactory returns a new Helm manager factory capable of installing and uninstalling releases.
func NewManagerFactory(crmanager crmanager.Manager, chartDir string) ManagerFactory {
	return &managerFactory{crmanager, chartDir}
}

func (f managerFactory) NewManager(r *unstructured.Unstructured) (Manager, error) {
	return f.newManagerForCR(r)
}

func (f managerFactory) newManagerForCR(r *unstructured.Unstructured) (Manager, error) {
	storageBackend, err := f.getStorageBackend(r)
	if err != nil {
		return nil, err
	}
	tillerKubeClient, err := client.NewFromManager(f.crmanager)
	if err != nil {
		return nil, err
	}
	releaseServer, err := tillerRendererForCR(r, tillerKubeClient, storageBackend)
	if err != nil {
		return nil, err
	}
	return &manager{
		storageBackend:   storageBackend,
		tillerKubeClient: tillerKubeClient,
		chartDir:         f.chartDir,

		tiller:      releaseServer,
		releaseName: getReleaseName(r),
		namespace:   r.GetNamespace(),

		spec:   r.Object["spec"],
		status: types.StatusFor(r),
	}, nil
}

// tillerRendererForCR creates a ReleaseServer configured with a rendering engine that adds ownerrefs to rendered assets
// based on the CR.
func tillerRendererForCR(r *unstructured.Unstructured, tillerKubeClient *kube.Client, storageBackend *storage.Storage) (*tiller.ReleaseServer, error) {
	controllerRef := metav1.NewControllerRef(r, r.GroupVersionKind())
	ownerRefs := []metav1.OwnerReference{
		*controllerRef,
	}
	baseEngine := helmengine.New()
	e := engine.NewOwnerRefEngine(baseEngine, ownerRefs)
	var ey environment.EngineYard = map[string]environment.Engine{
		environment.GoTplEngine: e,
	}
	env := &environment.Environment{
		EngineYard: ey,
		Releases:   storageBackend,
		KubeClient: tillerKubeClient,
	}
	kubeconfig, _ := tillerKubeClient.ToRESTConfig()
	cs, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return tiller.NewReleaseServer(env, cs, false), nil
}

func getReleaseName(r *unstructured.Unstructured) string {
	return fmt.Sprintf("%s-%s", r.GetName(), shortenUID(r.GetUID()))
}

func shortenUID(uid apitypes.UID) string {
	u := uuid.Parse(string(uid))
	uidBytes, err := u.MarshalBinary()
	if err != nil {
		return strings.Replace(string(uid), "-", "", -1)
	}
	return strings.ToLower(base36.EncodeBytes(uidBytes))
}

func (f managerFactory) getStorageBackend(r *unstructured.Unstructured) (*storage.Storage, error) {
	ownerRef := metav1.NewControllerRef(r, r.GroupVersionKind())
	clientv1, err := corev1.NewForConfig(f.crmanager.GetConfig())
	if err != nil {
		return nil, err
	}
	secrets := driver.NewOwnerSecrets(*ownerRef, clientv1.Secrets(r.GetNamespace()))
	return storage.Init(secrets), nil
}
