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
	"errors"
	"fmt"
	"io/ioutil"

	olmresourceclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	opinternal "github.com/operator-framework/operator-sdk/internal/olm/operator/internal"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	manifests "github.com/operator-framework/api/pkg/manifests"
	valerrors "github.com/operator-framework/api/pkg/validation/errors"
	olmapiv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	registry "github.com/operator-framework/operator-registry/pkg/registry"
	log "github.com/sirupsen/logrus"
	apiextinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

const defaultNamespace = "default"

func init() {
	// OLM schemes must be added to the global Scheme so controller-runtime's
	// client recognizes OLM objects.
	apiextinstall.Install(scheme.Scheme)
	if err := olmapiv1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("Failed to add OLM operator API v1 types to scheme: %v", err)
	}
}

type operatorManager struct {
	client        *olmresourceclient.Client
	version       string
	namespace     string
	forceRegistry bool

	installMode           olmapiv1alpha1.InstallModeType
	installModeNamespaces []string
	olmObjects            []runtime.Object
	pkg                   registry.PackageManifest
	bundles               []*registry.Bundle
}

func (c *OLMCmd) newManager() (*operatorManager, error) {
	m := &operatorManager{
		version:       c.OperatorVersion,
		forceRegistry: c.ForceRegistry,
	}
	rc, ns, err := k8sutil.GetKubeconfigAndNamespace(c.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace from kubeconfig %s: %w", c.KubeconfigPath, err)
	}
	if ns == "" {
		ns = defaultNamespace
	}
	if m.namespace = c.OperatorNamespace; m.namespace == "" {
		m.namespace = ns
	}
	if m.client == nil {
		m.client, err = olmresourceclient.ClientForConfig(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to create SDK OLM client: %w", err)
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
	pkg, bundles, results := manifests.GetManifestsDir(c.ManifestsDir)
	if len(results) != 0 {
		badResults := []valerrors.ManifestResult{}
		for _, result := range results {
			if result.HasError() || result.HasWarn() {
				badResults = append(badResults, result)
			}
		}
		if len(badResults) != 0 {
			return nil, fmt.Errorf("bundle dir had errors: %s", badResults)
		}
	}
	m.pkg, m.bundles = pkg, bundles
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

func (m *operatorManager) run(ctx context.Context) (err error) {
	// Ensure OLM is installed.
	olmVer, err := m.client.GetInstalledVersion(ctx)
	if err != nil {
		return fmt.Errorf("error getting installed OLM version: %w", err)
	}
	pkgName := m.pkg.PackageName
	bundle, err := getBundleForVersion(m.bundles, m.version)
	if err != nil {
		return fmt.Errorf("error getting bundle for version %s: %w", m.version, err)
	}
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return fmt.Errorf("error getting CSV from bundle: %w", err)
	}
	// Only check CSV here, since other deployed operators/versions may be
	// running with shared CRDs.
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(csv)
	if err != nil {
		return fmt.Errorf("error converting CSV to unstructured: %w", err)
	}
	u := unstructured.Unstructured{Object: obj}
	status := m.status(ctx, &u)
	if installed, err := status.HasInstalledResources(); installed {
		return fmt.Errorf("an operator with name %q is already running\n%s", pkgName, status)
	} else if err != nil {
		return fmt.Errorf("an operator with name %q is present and has resource errors\n%s", pkgName, status)
	}

	if err = m.registryUp(ctx, olmresourceclient.OLMNamespace); err != nil {
		return fmt.Errorf("error creating registry resources: %w", err)
	}
	log.Info("Creating resources")
	if !m.hasCatalogSource() {
		registryGRPCAddr := opinternal.GetRegistryServiceAddr(pkgName, olmresourceclient.OLMNamespace)
		catsrc := newCatalogSource(pkgName, m.namespace, withGRPC(registryGRPCAddr))
		m.olmObjects = append(m.olmObjects, catsrc)
	}
	if !m.hasSubscription() {
		channel, err := getChannelForCSVName(m.pkg, csv.GetName())
		if err != nil {
			return err
		}
		sub := newSubscription(csv.GetName(), m.namespace,
			withPackageChannel(pkgName, channel),
			withCatalogSource(getCatalogSourceName(pkgName), m.namespace))
		m.olmObjects = append(m.olmObjects, sub)
	}
	if !m.hasOperatorGroup() {
		og := newSDKOperatorGroup(m.namespace,
			withTargetNamespaces(m.installModeNamespaces...))
		m.olmObjects = append(m.olmObjects, og)
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
		return fmt.Errorf("error creating operator resources: %w", err)
	}
	if err = m.client.DoCreate(ctx, objects...); err != nil {
		return fmt.Errorf("error creating operator resources: %w", err)
	}
	// BUG(estroz): if m.namespace is not contained in m.installModeNamespaces,
	// DoCSVWait will fail.
	nn := types.NamespacedName{
		Name:      csv.GetName(),
		Namespace: m.namespace,
	}
	log.Printf("Waiting for ClusterServiceVersion %q to reach 'Succeeded' phase", nn)
	if err = m.client.DoCSVWait(ctx, nn); err != nil {
		return fmt.Errorf("error waiting for CSV to install: %w", err)
	}

	status = m.status(ctx, bundle.Objects...)
	if installed, err := status.HasInstalledResources(); !installed {
		return fmt.Errorf("operator %s did not install successfully\n%s", pkgName, status)
	} else if err != nil {
		return fmt.Errorf("operator %qhas resource errors\n%s", pkgName, status)
	}
	log.Infof("Successfully installed %q on OLM version %q", csv.GetName(), olmVer)
	fmt.Print(status)

	return nil
}

func (m *operatorManager) cleanup(ctx context.Context) (err error) {
	// Ensure OLM is installed.
	olmVer, err := m.client.GetInstalledVersion(ctx)
	if err != nil {
		return fmt.Errorf("error getting installed OLM version: %w", err)
	}
	pkgName := m.pkg.PackageName
	bundle, err := getBundleForVersion(m.bundles, m.version)
	if err != nil {
		return fmt.Errorf("error getting bundle for version %s: %w", m.version, err)
	}
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return fmt.Errorf("error getting CSV from bundle: %w", err)
	}

	if err = m.registryDown(ctx, olmresourceclient.OLMNamespace); err != nil {
		return fmt.Errorf("error removing registry resources: %w", err)
	}
	log.Info("Deleting resources")
	if !m.hasCatalogSource() {
		m.olmObjects = append(m.olmObjects, newCatalogSource(pkgName, m.namespace))
	}
	if !m.hasSubscription() {
		m.olmObjects = append(m.olmObjects, newSubscription(csv.GetName(), m.namespace))
	}
	if !m.hasOperatorGroup() {
		m.olmObjects = append(m.olmObjects, newSDKOperatorGroup(m.namespace))
	}
	toDelete := []runtime.Object{}
	for _, obj := range m.olmObjects {
		toDelete = append(toDelete, obj.DeepCopyObject())
	}
	for _, obj := range bundle.Objects {
		objc := obj.DeepCopy()
		objc.SetNamespace(m.namespace)
		toDelete = append(toDelete, objc)
	}
	if err = m.client.DoDelete(ctx, toDelete...); err != nil {
		return fmt.Errorf("error deleting operator resources: %w", err)
	}

	status := m.status(ctx, bundle.Objects...)
	if installed, err := status.HasInstalledResources(); installed {
		return fmt.Errorf("operator %q still exists", pkgName)
	} else if err != nil {
		return fmt.Errorf("operator %q still exists and has resource errors\n%s", pkgName, status)
	}
	log.Infof("Successfully uninstalled %q on OLM version %q", csv.GetName(), olmVer)

	return nil
}

func (m operatorManager) registryUp(ctx context.Context, namespace string) error {
	rr := opinternal.RegistryResources{
		Client:  m.client,
		Pkg:     m.pkg,
		Bundles: m.bundles,
	}
	registryStale, err := rr.IsManifestDataStale(ctx, namespace)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking registry data: %w", err)
		}
		// ConfigMap doesn't exist yet, so create it as usual.
		log.Info("Creating registry")
		if err = rr.CreateRegistryManifests(ctx, namespace); err != nil {
			return fmt.Errorf("error registering bundle: %w", err)
		}
		return nil
	}
	if !registryStale {
		log.Printf("Registry data is current")
		return nil
	}
	log.Printf("Registry data stale. Recreating registry")
	if err = rr.DeleteRegistryManifests(ctx, namespace); err != nil {
		return fmt.Errorf("error deleting registered bundle: %w", err)
	}
	if err = rr.CreateRegistryManifests(ctx, namespace); err != nil {
		return fmt.Errorf("error registering bundle: %w", err)
	}
	return nil
}

func (m *operatorManager) registryDown(ctx context.Context, namespace string) error {
	rr := opinternal.RegistryResources{
		Client:  m.client,
		Pkg:     m.pkg,
		Bundles: m.bundles,
	}
	if m.forceRegistry {
		log.Printf("Deleting registry")
		if err := rr.DeleteRegistryManifests(ctx, namespace); err != nil {
			return fmt.Errorf("error deleting registered bundle: %w", err)
		}
	}
	return nil
}

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

func readObjectsFromFile(path string) (objs []*unstructured.Unstructured, err error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	scanner := yamlutil.NewYAMLScanner(b)
	for scanner.Scan() {
		u := &unstructured.Unstructured{}
		if err := u.UnmarshalJSON(scanner.Bytes()); err != nil {
			return nil, fmt.Errorf("failed to decode object from manifest %s: %w", path, err)
		}
		objs = append(objs, u)
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("failed to scan manifest %s: %w", path, scanner.Err())
	}
	if len(objs) == 0 {
		return nil, fmt.Errorf("no objects found in manifest %s", path)
	}
	return objs, nil
}

func getBundleForVersion(bundles []*registry.Bundle, version string) (*registry.Bundle, error) {
	names := []string{}
	for _, bundle := range bundles {
		if bundle.Name == version {
			return bundle, nil
		}
		names = append(names, bundle.Name)
	}
	return nil, fmt.Errorf("no bundle found for version %s; valid names: %+q", version, names)
}
