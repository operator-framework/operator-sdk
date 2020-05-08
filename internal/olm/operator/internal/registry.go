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
	"path"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const (
	// The root directory containing of a package manifests format for an
	// operator, with the package manifest being top-level.
	containerManifestsDir = "/registry/manifests"
)

// SDKLabels are used to identify certain operator-sdk resources.
var SDKLabels = map[string]string{
	"owner": "operator-sdk",
}

// RegistryResources configures creation/deletion of internal registry-related
// resources.
type RegistryResources struct {
	Client  *olmclient.Client
	Pkg     *apimanifests.PackageManifest
	Bundles []*apimanifests.Bundle
}

// IsRegistryExist returns true if a registry Deployment exists in namespace.
func (rr *RegistryResources) IsRegistryExist(ctx context.Context, namespace string) (bool, error) {
	depKey := types.NamespacedName{
		Name:      getRegistryServerName(rr.Pkg.PackageName),
		Namespace: namespace,
	}
	err := rr.Client.KubeClient.Get(ctx, depKey, &appsv1.Deployment{})
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
func (rr *RegistryResources) IsRegistryDataStale(ctx context.Context, namespace string) (bool, error) {
	configMaps, err := rr.getRegistryConfigMaps(ctx, namespace)
	if err != nil {
		return false, err
	}
	// Simple length comparison for packages + package manifest ConfigMaps.
	if len(configMaps) != len(rr.Bundles)+1 {
		return true, nil
	}

	binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(rr.Pkg, rr.Bundles)
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

// CreatePackageManifestsRegistry creates all registry objects required to serve
// manifests from rr.manifests in namespace.
func (rr *RegistryResources) CreatePackageManifestsRegistry(ctx context.Context, namespace string) error {
	pkgName := rr.Pkg.PackageName
	labels := makeRegistryLabels(pkgName)

	binaryDataByConfigMap, err := makeConfigMapsForPackageManifests(rr.Pkg, rr.Bundles)
	if err != nil {
		return err
	}

	// Objects to create.
	objs := make([]runtime.Object, 0, len(binaryDataByConfigMap)+2)
	// Options for creating a Deployment, since we need to mount all package
	// ConfigMaps as volumes into pods.
	opts := make([]func(*appsv1.Deployment), 0, 2*len(binaryDataByConfigMap)+1)
	opts = append(opts, withRegistryGRPCContainer(pkgName))
	// Build all package ConfigMaps.
	for cmName, binaryData := range binaryDataByConfigMap {
		cm := newConfigMap(cmName, namespace, withBinaryData(binaryData))
		cm.SetLabels(labels)
		objs = append(objs, cm)

		volName := k8sutil.TrimDNS1123Label(cmName + "-volume")
		opts = append(opts,
			withConfigMapVolume(volName, cmName),
			withContainerVolumeMounts(volName, path.Join(containerManifestsDir, cmName)),
		)
	}

	// Add registry Deployment and Service to objects.
	dep := newRegistryDeployment(pkgName, namespace, opts...)
	dep.SetLabels(labels)
	service := newRegistryService(pkgName, namespace, withTCPPort("grpc", registryGRPCPort))
	service.SetLabels(labels)
	objs = append(objs, dep, service)

	if err := rr.Client.DoCreate(ctx, objs...); err != nil {
		return fmt.Errorf("error creating operator %q registry-server objects: %w", pkgName, err)
	}

	// Wait for registry Deployment rollout.
	depKey := types.NamespacedName{
		Name:      dep.GetName(),
		Namespace: namespace,
	}
	log.Infof("Waiting for Deployment %q rollout to complete", depKey)
	if err := rr.Client.DoRolloutWait(ctx, depKey); err != nil {
		return fmt.Errorf("error waiting for Deployment %q to roll out: %w", depKey, err)
	}

	return nil
}

// DeletePackageManifestsRegistry deletes all registry objects serving manifests
// for an operator in namespace.
// TODO: delete by owner reference.
func (rr *RegistryResources) DeletePackageManifestsRegistry(ctx context.Context, namespace string) error {

	// List all registry ConfigMaps by label.
	configMaps, err := rr.getRegistryConfigMaps(ctx, namespace)
	if err != nil {
		return err
	}

	// Delete registry Deployment, Service, and ConfigMaps by type.
	objs := make([]runtime.Object, len(configMaps)+2)
	for i := range configMaps {
		objs[i] = &configMaps[i]
	}
	pkgName := rr.Pkg.PackageName
	objs[len(objs)-2] = newRegistryDeployment(pkgName, namespace)
	objs[len(objs)-1] = newRegistryService(pkgName, namespace)
	err = rr.Client.DoDelete(ctx, objs...)
	if err != nil {
		return fmt.Errorf("error deleting operator %q registry-server objects: %w", pkgName, err)
	}

	return nil
}

// GetRegistryServiceAddr returns a Service's DNS name + port for a given
// pkgName and namespace.
func GetRegistryServiceAddr(pkgName, namespace string) string {
	name := getRegistryServerName(pkgName)
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", name, namespace, registryGRPCPort)
}

// makeRegistryLabels creates a set of labels to identify operator-registry objects.
func makeRegistryLabels(pkgName string) map[string]string {
	labels := map[string]string{
		"package-name": k8sutil.TrimDNS1123Label(pkgName),
	}
	for k, v := range SDKLabels {
		labels[k] = v
	}
	return labels
}
