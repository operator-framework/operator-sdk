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

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	internalregistry "github.com/operator-framework/operator-sdk/internal/olm/operator/internal"
)

type packageManifestsManager struct {
	*operatorManager

	version       string
	forceRegistry bool
	pkg           *apimanifests.PackageManifest
	bundles       []*apimanifests.Bundle
}

func (c *PackageManifestsCmd) newManager() (m *packageManifestsManager, err error) {
	m = &packageManifestsManager{
		version:       c.OperatorVersion,
		forceRegistry: c.ForceRegistry,
	}
	if m.operatorManager, err = c.OperatorCmd.newManager(); err != nil {
		return nil, err
	}

	// Parse k8s objects.
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

	// Operator bundles and metadata.
	m.pkg, m.bundles, err = apimanifests.GetManifestsDir(c.ManifestsDir)
	if err != nil {
		return nil, err
	}
	if len(m.bundles) == 0 {
		return nil, errors.New("no packages found")
	}
	if m.pkg == nil || m.pkg.PackageName == "" {
		return nil, errors.New("no package manifest found")
	}

	// Handle installModes.
	if c.InstallMode == "" {
		// Default to OwnNamespace.
		m.installMode = operatorsv1alpha1.InstallModeTypeOwnNamespace
		m.targetNamespaces = []string{m.operatorNamespace}
	} else {
		m.installMode, m.targetNamespaces, err = parseInstallModeKV(c.InstallMode)
		if err != nil {
			return nil, err
		}
	}

	// Ensure CSV supports installMode.
	bundle, err := getPackageForVersion(m.bundles, m.version)
	if err != nil {
		return nil, err
	}
	if err := installModeCompatible(bundle.CSV, m.installMode, m.operatorNamespace, m.targetNamespaces); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *packageManifestsManager) run(ctx context.Context) (err error) {
	// Ensure OLM is installed.
	olmVer, err := m.client.GetInstalledVersion(ctx, m.olmNamespace)
	if err != nil {
		return fmt.Errorf("error getting installed OLM version: %w", err)
	}

	pkgName := m.pkg.PackageName
	bundle, err := getPackageForVersion(m.bundles, m.version)
	if err != nil {
		return fmt.Errorf("error getting package for version %s: %w", m.version, err)
	}
	csv := bundle.CSV

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

	if err = m.registryUp(ctx, m.olmNamespace); err != nil {
		return fmt.Errorf("error creating registry resources: %w", err)
	}

	log.Info("Creating resources")
	if !m.hasCatalogSource() {
		registryGRPCAddr := internalregistry.GetRegistryServiceAddr(pkgName, m.olmNamespace)
		catsrc := newCatalogSource(pkgName, m.operatorNamespace, withGRPC(registryGRPCAddr))
		m.olmObjects = append(m.olmObjects, catsrc)
	}
	if !m.hasSubscription() {
		channel, err := getChannelForCSVName(m.pkg, csv.GetName())
		if err != nil {
			return err
		}
		sub := newSubscription(csv.GetName(), m.operatorNamespace,
			withPackageChannel(pkgName, channel),
			withCatalogSource(getCatalogSourceName(pkgName), m.operatorNamespace))
		m.olmObjects = append(m.olmObjects, sub)
	}
	if !m.hasOperatorGroup() {
		og := newSDKOperatorGroup(m.operatorNamespace,
			withTargetNamespaces(m.targetNamespaces...))
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

	// BUG(estroz): if operatorNamespace is not contained in targetNamespaces,
	// DoCSVWait will fail because the CSV is not deployed in operatorNamespace.
	nn := types.NamespacedName{
		Name:      csv.GetName(),
		Namespace: m.operatorNamespace,
	}
	log.Printf("Waiting for ClusterServiceVersion %q to reach 'Succeeded' phase", nn)
	if err = m.client.DoCSVWait(ctx, nn); err != nil {
		return fmt.Errorf("error waiting for CSV to install: %w", err)
	}

	status = m.status(ctx, bundle.Objects...)
	if installed, err := status.HasInstalledResources(); !installed {
		return fmt.Errorf("operator %s did not install successfully\n%s", pkgName, status)
	} else if err != nil {
		return fmt.Errorf("operator %q has resource errors\n%s", pkgName, status)
	}
	log.Infof("Successfully installed %q on OLM version %q", csv.GetName(), olmVer)
	fmt.Print(status)

	return nil
}

func (m *packageManifestsManager) cleanup(ctx context.Context) (err error) {
	// Ensure OLM is installed.
	olmVer, err := m.client.GetInstalledVersion(ctx, m.olmNamespace)
	if err != nil {
		return fmt.Errorf("error getting installed OLM version: %w", err)
	}

	pkgName := m.pkg.PackageName
	bundle, err := getPackageForVersion(m.bundles, m.version)
	if err != nil {
		return fmt.Errorf("error getting package for version %s: %w", m.version, err)
	}
	csv := bundle.CSV

	if err = m.registryDown(ctx, m.olmNamespace); err != nil {
		return fmt.Errorf("error removing registry resources: %w", err)
	}

	log.Info("Deleting resources")
	if !m.hasCatalogSource() {
		m.olmObjects = append(m.olmObjects, newCatalogSource(pkgName, m.operatorNamespace))
	}
	if !m.hasSubscription() {
		m.olmObjects = append(m.olmObjects, newSubscription(csv.GetName(), m.operatorNamespace))
	}
	if !m.hasOperatorGroup() {
		m.olmObjects = append(m.olmObjects, newSDKOperatorGroup(m.operatorNamespace))
	}
	toDelete := []runtime.Object{}
	for _, obj := range m.olmObjects {
		toDelete = append(toDelete, obj.DeepCopyObject())
	}
	for _, obj := range bundle.Objects {
		objc := obj.DeepCopy()
		objc.SetNamespace(m.operatorNamespace)
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

func (m packageManifestsManager) registryUp(ctx context.Context, namespace string) error {
	rr := internalregistry.RegistryResources{
		Client:  m.client,
		Pkg:     m.pkg,
		Bundles: m.bundles,
	}

	if exists, err := rr.IsRegistryExist(ctx, namespace); err != nil {
		return fmt.Errorf("error checking registry existence: %v", err)
	} else if exists {
		if isRegistryStale, err := rr.IsRegistryDataStale(ctx, namespace); err == nil {
			if !isRegistryStale {
				log.Infof("%s registry data is current", m.pkg.PackageName)
				return nil
			}
			log.Infof("A stale %s registry exists, deleting", m.pkg.PackageName)
			if err = rr.DeletePackageManifestsRegistry(ctx, namespace); err != nil {
				return fmt.Errorf("error deleting registered package: %w", err)
			}
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking registry data: %w", err)
		}
	}
	log.Infof("Creating %s registry", m.pkg.PackageName)
	if err := rr.CreatePackageManifestsRegistry(ctx, namespace); err != nil {
		return fmt.Errorf("error registering package: %w", err)
	}

	return nil
}

func (m *packageManifestsManager) registryDown(ctx context.Context, namespace string) error {
	rr := internalregistry.RegistryResources{
		Client:  m.client,
		Pkg:     m.pkg,
		Bundles: m.bundles,
	}

	if m.forceRegistry {
		log.Print("Deleting registry")
		if err := rr.DeletePackageManifestsRegistry(ctx, namespace); err != nil {
			return fmt.Errorf("error deleting registered package: %w", err)
		}
	}

	return nil
}

func getPackageForVersion(bundles []*apimanifests.Bundle, version string) (*apimanifests.Bundle, error) {
	versions := []string{}
	for _, bundle := range bundles {
		verStr := bundle.CSV.Spec.Version.String()
		if verStr == version {
			return bundle, nil
		}
		versions = append(versions, verStr)
	}
	return nil, fmt.Errorf("no package found for version %s; valid versions: %+q", version, versions)
}
