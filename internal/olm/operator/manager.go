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

package olm

import (
	"context"
	"fmt"
	"io/ioutil"

	olmresourceclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	opinternal "github.com/operator-framework/operator-sdk/internal/olm/operator/internal"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	registryutil "github.com/operator-framework/operator-sdk/internal/util/operator-registry"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	olmapiv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
)

// TODO(estroz): ensure OLM errors are percolated up to the user.

var (
	Scheme = olmresourceclient.Scheme

	defaultNamespace = "default"
)

func init() {
	if err := apiextv1beta1.AddToScheme(Scheme); err != nil {
		log.Fatalf("Failed to add Kubhernetes API extensions v1beta1 types to scheme: %v", err)
	}
	if err := olmapiv1.AddToScheme(Scheme); err != nil {
		log.Fatalf("Failed to add OLM operator API v1 types to scheme: %v", err)
	}
}

type operatorManager struct {
	client    *olmresourceclient.Client
	version   string
	namespace string
	force     bool

	installMode           olmapiv1alpha1.InstallModeType
	installModeNamespaces []string
	olmObjects            []runtime.Object
	manifests             registryutil.ManifestsStore
}

func (c *OLMCmd) newManager() (*operatorManager, error) {
	m := &operatorManager{
		force:   c.Force,
		version: c.OperatorVersion,
	}
	rc, ns, err := k8sutil.GetKubeconfigAndNamespace(c.KubeconfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get namespace from kubeconfig %s", c.KubeconfigPath)
	}
	if ns == "" {
		ns = defaultNamespace
	}
	if c.OperatorNamespace == "" {
		m.namespace = ns
	} else {
		m.namespace = c.OperatorNamespace
	}
	if m.client == nil {
		m.client, err = olmresourceclient.ClientForConfig(rc)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create SDK OLM client")
		}
	}
	for _, path := range c.IncludePaths {
		if path != "" {
			objs, err := readObjectsFromFile(path)
			if err != nil {
				return nil, err
			}
			for _, obj := range objs {
				m.olmObjects = append(m.olmObjects, obj)
			}
		}
	}
	// Since a Subscription refers to a CatalogSource, supplying one but
	// not the other is an error.
	hasSub, hasCatSrc := m.hasSubscription(), m.hasCatalogSource()
	if hasSub || hasCatSrc && !(hasSub && hasCatSrc) {
		return nil, errors.New("both a CatalogSource and Subscription must be supplied if one is supplied")
	}
	m.manifests, err = registryutil.ManifestsStoreForDir(c.ManifestsDir)
	if err != nil {
		return nil, err
	}
	if c.InstallMode == "" {
		// Default to OwnNamespace.
		m.installMode = olmapiv1alpha1.InstallModeTypeOwnNamespace
		m.installModeNamespaces = []string{m.namespace}
	} else {
		m.installMode, m.installModeNamespaces, err = parseInstallModeKV(c.InstallMode)
		if err != nil {
			return nil, err
		}
	}
	if err := m.installModeCompatible(m.installMode); err != nil {
		return nil, err
	}
	return m, nil
}

func readObjectsFromFile(path string) (objs []*unstructured.Unstructured, err error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := serializer.NewCodecFactory(Scheme).UniversalDeserializer()
	scanner := yamlutil.NewYAMLScanner(b)
	for scanner.Scan() {
		u := unstructured.Unstructured{}
		_, _, err := dec.Decode(scanner.Bytes(), nil, &u)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode object from manifest %s", path)
		}
		objs = append(objs, &u)
	}
	if scanner.Err() != nil {
		return nil, errors.Wrapf(scanner.Err(), "failed to scan manifest %s", path)
	}
	if len(objs) == 0 {
		return nil, errors.Errorf("no objects found in manifest %s", path)
	}
	return objs, nil
}

func (m operatorManager) hasCatalogSource() bool {
	return m.hasKind(olmapiv1alpha1.CatalogSourceKind)
}

func (m operatorManager) hasSubscription() bool {
	return m.hasKind(olmapiv1alpha1.SubscriptionKind)
}

func (m operatorManager) hasOperatorGroup() bool {
	return m.hasKind(olmapiv1.OperatorGroupKind)
}

func (m operatorManager) hasKind(kind string) bool {
	for _, obj := range m.olmObjects {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			return true
		}
	}
	return false
}

func (m *operatorManager) up(ctx context.Context) (err error) {
	// Ensure OLM is installed.
	olmVer, err := m.client.GetInstalledVersion(ctx)
	if err != nil {
		return err
	}
	pkg := m.manifests.GetPackageManifest()
	pkgName := pkg.PackageName
	bundle, err := m.manifests.GetBundleForVersion(m.version)
	if err != nil {
		return err
	}
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return err
	}
	if !m.force {
		// Only check CSV here, since other deployed operators/versions may be
		// running with shared CRDs.
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(csv)
		if err != nil {
			return err
		}
		u := unstructured.Unstructured{Object: obj}
		if status := m.status(ctx, &u); status.HasExistingResources() {
			return errors.Errorf("an operator with name %q is already running\n%s", pkgName, status)
		}
	}

	log.Info("Creating resources")
	if err = m.registryUp(ctx, olmresourceclient.OLMNamespace); err != nil {
		return err
	}
	if !m.hasCatalogSource() {
		registryGRPCAddr := opinternal.GetRegistryServiceAddr(pkgName, olmresourceclient.OLMNamespace)
		catsrc := newCatalogSource(pkgName, m.namespace, withGRPC(registryGRPCAddr))
		m.olmObjects = append(m.olmObjects, catsrc)
	}
	if !m.hasSubscription() {
		channel, err := getChannelForCSVName(pkg, csv.GetName())
		if err != nil {
			return err
		}
		sub := newSubscription(csv.GetName(), m.namespace,
			withPackageChannel(pkgName, channel),
			withCatalogSource(getCatalogSourceName(pkgName), m.namespace))
		m.olmObjects = append(m.olmObjects, sub)
	}
	if !m.hasOperatorGroup() {
		if err = m.operatorGroupUp(ctx); err != nil {
			return err
		}
	}
	// Check for Namespace objects and create those first.
	namespaces, objects := []runtime.Object{}, []runtime.Object{}
	for _, obj := range m.olmObjects {
		if obj.GetObjectKind().GroupVersionKind().Kind == "Namespace" {
			namespaces = append(namespaces, obj)
		} else {
			objects = append(objects, obj)
		}
	}
	if err = m.client.DoCreate(ctx, namespaces...); err != nil {
		return err
	}
	if err = m.client.DoCreate(ctx, objects...); err != nil {
		return err
	}
	nn := types.NamespacedName{
		Name:      csv.GetName(),
		Namespace: m.namespace,
	}
	log.Printf("Waiting for ClusterServiceVersion %q to reach 'Succeeded' phase", nn)
	if err = m.client.DoCSVWait(ctx, nn); err != nil {
		return err
	}

	status := m.status(ctx, bundle.Objects...)
	if len(status.Resources) != len(bundle.Objects) {
		return errors.Errorf("some operator %q resources did not install\n%s", csv.GetName(), status)
	}
	log.Infof("Successfully installed %q on OLM version %q", csv.GetName(), olmVer)
	fmt.Print(status)

	return nil
}

func (m operatorManager) registryUp(ctx context.Context, namespace string) error {
	rr := opinternal.RegistryResources{
		Client:    m.client,
		Manifests: m.manifests,
	}
	registryStale, err := rr.RegistryDataStale(ctx, namespace)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "error checking registry data")
		}
		// ConfigMap doesn't exist yet, so create it as usual.
		if err = rr.CreateRegistryManifests(ctx, namespace); err != nil {
			return errors.Wrap(err, "error registering bundle")
		}
		return nil
	}
	if !registryStale && !m.force {
		log.Printf("Registry data is current")
		return nil
	}
	if m.force {
		log.Printf("Forcefully recreating registry")
	} else {
		log.Printf("Registry data stale. Recreating registry")
	}
	if err = rr.DeleteRegistryManifests(ctx, namespace); err != nil {
		return errors.Wrap(err, "error deleting registered bundle")
	}
	if err = rr.CreateRegistryManifests(ctx, namespace); err != nil {
		return errors.Wrap(err, "error registering bundle")
	}
	return nil
}

func (m *operatorManager) down(ctx context.Context) (err error) {
	// Ensure OLM is installed.
	olmVer, err := m.client.GetInstalledVersion(ctx)
	if err != nil {
		return err
	}
	pkg := m.manifests.GetPackageManifest()
	pkgName := pkg.PackageName
	bundle, err := m.manifests.GetBundleForVersion(m.version)
	if err != nil {
		return err
	}
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return err
	}
	if !m.force {
		if status := m.status(ctx, bundle.Objects...); !status.HasExistingResources() {
			return errors.Errorf("no operator with name %q is running", pkgName)
		}
	}

	log.Info("Deleting resources")
	if err = m.registryDown(ctx, olmresourceclient.OLMNamespace); err != nil {
		return err
	}
	if !m.hasCatalogSource() {
		m.olmObjects = append(m.olmObjects, newCatalogSource(pkgName, m.namespace))
	}
	if !m.hasSubscription() {
		m.olmObjects = append(m.olmObjects, newSubscription(csv.GetName(), m.namespace))
	}
	if !m.hasOperatorGroup() {
		if err = m.operatorGroupDown(ctx); err != nil {
			return err
		}
	}
	toDelete := make([]runtime.Object, len(m.olmObjects))
	copy(toDelete, m.olmObjects)
	for _, o := range bundle.Objects {
		oc := o.DeepCopy()
		oc.SetNamespace(m.namespace)
		toDelete = append(toDelete, oc)
	}
	if err = m.client.DoDelete(ctx, toDelete...); err != nil {
		return err
	}

	status := m.status(ctx, bundle.Objects...)
	if status.HasExistingResources() {
		return errors.Errorf("operator %q resources still exist\n%s", csv.GetName(), status)
	}
	log.Infof("Successfully uninstalled %q on OLM version %q", csv.GetName(), olmVer)

	return nil
}

func (m *operatorManager) registryDown(ctx context.Context, namespace string) error {
	rr := opinternal.RegistryResources{
		Client:    m.client,
		Manifests: m.manifests,
	}
	if m.force {
		log.Printf("Forcefully deleting registry")
		if err := rr.DeleteRegistryManifests(ctx, namespace); err != nil {
			return errors.Wrap(err, "error deleting registered bundle")
		}
	}
	return nil
}

// TODO(estroz): "status" subcommand
// TODO(estroz): check registry health on each "status" subcommand invokation
func (m *operatorManager) status(ctx context.Context, us ...*unstructured.Unstructured) olmresourceclient.Status {
	objs := []runtime.Object{}
	for _, u := range us {
		uc := u.DeepCopy()
		uc.SetNamespace(m.namespace)
		objs = append(objs, uc)
	}
	return m.client.GetObjectsStatus(ctx, objs...)
}
