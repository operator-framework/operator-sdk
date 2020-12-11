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

package configmap

import (
	"context"
	"fmt"
	"path"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	registrypod "github.com/operator-framework/operator-sdk/internal/olm/operator/registry/pod"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const (
	// The root directory containing of a package manifests format for an
	// operator, with the package manifest being top-level.
	containerManifestsDir = "/registry/manifests"
)

// Manager configures creation/deletion of internal registry-related
// resources.
type Manager struct {
	pkg     *apimanifests.PackageManifest
	bundles []*apimanifests.Bundle
	cfg     *operator.Configuration
}

func NewManager(cfg *operator.Configuration, pkg *apimanifests.PackageManifest, bundles []*apimanifests.Bundle) *Manager {
	m := &Manager{}
	m.pkg = pkg
	m.bundles = bundles
	m.cfg = cfg
	return m
}

// IsRegistryExist returns true if a registry Pod exists in namespace.
func (m *Manager) IsRegistryExist(ctx context.Context) (bool, error) {
	podKey := types.NamespacedName{
		Name:      getRegistryPodName(m.pkg.PackageName),
		Namespace: m.cfg.Namespace,
	}
	err := m.cfg.Client.Get(ctx, podKey, &corev1.Pod{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsRegistryDataStale checks if manifest data stored in the on-cluster
// registry is stale by comparing it to the currently managed data.
func (m *Manager) IsRegistryDataStale(ctx context.Context) (bool, error) {
	configMaps, err := m.getRegistryConfigMaps(ctx)
	if err != nil {
		return false, err
	}
	// Simple length comparison for packages + package manifest ConfigMaps.
	if len(configMaps) != len(m.bundles)+1 {
		return true, nil
	}

	binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(m.pkg, m.bundles)
	if err != nil {
		return false, err
	}

	for _, configMap := range configMaps {
		binaryData, hasName := binaryDataByConfigMap[configMap.GetName()]
		if !hasName {
			return true, nil
		}
		// If the number of files to be added to the registry don't match the number
		// of files currently in the registry, we have added or removed a file.
		if len(binaryData) != len(configMap.BinaryData) {
			return true, nil
		}
		// Check each binary value's key, which contains a base32-encoded md5 digest
		// component, against the new set of manifest keys.
		for fileKey := range configMap.BinaryData {
			if _, match := binaryData[fileKey]; !match {
				return true, nil
			}
			delete(binaryData, fileKey)
		}
		// Registry is stale if there are new keys not in the existing ConfigMap.
		if len(binaryData) != 0 {
			return true, nil
		}
	}
	return false, nil
}

// CreateRegistry creates all registry objects required to serve
// manifests from m.manifests in namespace.
func (m *Manager) CreateRegistry(ctx context.Context, cs *v1alpha1.CatalogSource) (*corev1.Pod, error) {
	pkgName := m.pkg.PackageName
	labels := makeRegistryLabels(pkgName)

	binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(m.pkg, m.bundles)
	if err != nil {
		return nil, err
	}

	csKey := types.NamespacedName{
		Namespace: cs.GetNamespace(),
		Name:      cs.GetName(),
	}
	if err := m.cfg.Client.Get(ctx, csKey, cs); err != nil {
		return nil, fmt.Errorf("get catalog source: %v", err)
	}
	fmt.Println(csKey)

	// Objects to create.
	objs := make([]client.Object, 0, len(binaryDataByConfigMap))
	// Options for creating a Pod, since we need to mount all package
	// ConfigMaps as volumes into pods.
	opts := make([]func(*corev1.Pod), 0, 2*len(binaryDataByConfigMap)+1)
	opts = append(opts, withRegistryGRPCContainer(pkgName))
	// Build all package ConfigMaps.
	for cmName, binaryData := range binaryDataByConfigMap {
		cm := newConfigMap(cmName, m.cfg.Namespace, withBinaryData(binaryData))
		cm.SetLabels(labels)
		objs = append(objs, cm)

		volName := k8sutil.TrimDNS1123Label(cmName + "-volume")
		opts = append(opts,
			withConfigMapVolume(volName, cmName),
			withContainerVolumeMounts(volName, path.Join(containerManifestsDir, cmName)),
		)
	}

	log.Info("Creating registry data configmaps")
	for _, obj := range objs {
		// Set cs as the owner of cm to cascade delete when cs is deleted.
		if err := controllerutil.SetOwnerReference(cs, obj, m.cfg.Scheme); err != nil {
			return nil, fmt.Errorf("error seting configmap owner reference: %w", err)
		}
		if err := m.cfg.Client.Create(ctx, obj); err != nil {
			return nil, fmt.Errorf("error creating configmap: %w", err)
		}
	}

	// Create registry pod.
	pod := newRegistryPod(pkgName, m.cfg.Namespace, opts...)
	pod.SetLabels(labels)
	if err := registrypod.CreateOwnedPod(ctx, m.cfg, pod, cs); err != nil {
		return nil, fmt.Errorf("error creating registry pod: %w", err)
	}

	return pod, nil
}

// DeleteRegistry deletes all registry objects owned by cs.
func (m *Manager) DeleteRegistry(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	if err := m.cfg.Client.Get(ctx, client.ObjectKeyFromObject(cs), cs); err != nil {
		return fmt.Errorf("error getting catalog source: %w", err)
	}
	for _, owned := range cs.GetOwnerReferences() {
		var obj client.Object
		switch owned.Kind {
		case "Pod":
			obj = &corev1.Pod{}
		case "ConfigMap":
			obj = &corev1.ConfigMap{}
		}
		obj.SetName(owned.Name)
		obj.SetNamespace(m.cfg.Namespace)
		fmt.Println("deleting", owned.Kind, client.ObjectKeyFromObject(obj))
		err := m.cfg.Client.Delete(ctx, obj)
		if err != nil {
			return fmt.Errorf("error deleting registry resource: %w", err)
		}
	}
	return nil
}

// makeRegistryLabels creates a set of labels to identify operator-registry objects.
func makeRegistryLabels(pkgName string) map[string]string {
	labels := map[string]string{
		"owner":        "operator-sdk",
		"package-name": k8sutil.TrimDNS1123Label(pkgName),
	}
	return labels
}
